package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	httpswagger "github.com/devplayer0/http-swagger"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lxd "github.com/lxc/lxd/client"
	iam "github.com/netsoc/iam/client"
	"github.com/rs/cors"

	"github.com/netsoc/webspaced/internal/config"
	"github.com/netsoc/webspaced/internal/data"
	"github.com/netsoc/webspaced/internal/webspace"
	"github.com/netsoc/webspaced/pkg/util"
)

// Server is the main webspaced server struct
type Server struct {
	Config    config.Config
	Webspaces *webspace.Manager

	iam  *iam.APIClient
	lxd  lxd.InstanceServer
	http *http.Server
}

// NewServer returns an initialized Server
func NewServer(config config.Config) *Server {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: config.HTTP.CORS.AllowedOrigins,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	r := mux.NewRouter()
	httpSrv := &http.Server{
		Addr:    config.HTTP.ListenAddress,
		Handler: claimsMiddleware(handlers.CustomLoggingHandler(nil, corsMiddleware.Handler(r), writeAccessLog)),
	}

	cfg := iam.NewConfiguration()
	cfg.BasePath = config.IAM.URL
	if config.IAM.AllowInsecure {
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	s := &Server{
		Config: config,

		iam:  iam.NewAPIClient(cfg),
		http: httpSrv,
	}

	r.HandleFunc("/v1/images", s.apiImages).Methods("GET")

	authM := authMiddleware{IAM: s.iam}
	wsRouter := r.PathPrefix("/v1/webspace/{username}").Subrouter()
	wsRouter.Use(authM.Middleware)
	wsRouter.HandleFunc("", s.apiCreateWebspace).Methods("POST")

	wsOpRouter := wsRouter.NewRoute().Subrouter()
	wsOpRouter.Use(s.getWebspaceMiddleware)
	wsOpRouter.HandleFunc("", s.apiGetWebspace).Methods("GET")
	wsOpRouter.HandleFunc("", s.apiDeleteWebspace).Methods("DELETE")

	wsOpRouter.HandleFunc("/state", s.apiGetWebspaceState).Methods("GET")
	wsOpRouter.HandleFunc("/state", s.apiSetWebspaceState).Methods("POST", "PATCH", "PUT", "DELETE")

	wsOpRouter.HandleFunc("/config", s.apiGetWebspaceConfig).Methods("GET")
	wsOpRouter.HandleFunc("/config", s.apiUpdateWebspaceConfig).Methods("PATCH")

	wsOpRouter.HandleFunc("/domains", s.apiGetWebspaceDomains).Methods("GET")
	wsOpRouter.HandleFunc("/domains/{domain}", s.apiWebspaceDomain).Methods("POST", "DELETE")

	wsOpRouter.HandleFunc("/ports", s.apiGetWebspacePorts).Methods("GET")
	wsOpRouter.HandleFunc("/ports/{ePort}/{iPort}", s.apiWebspacePorts).Methods("POST")
	wsOpRouter.HandleFunc("/ports/{port}", s.apiWebspacePorts).Methods("POST", "DELETE")

	wsOpRouter.HandleFunc("/log", s.apiConsoleLog).Methods("GET")
	wsOpRouter.HandleFunc("/log", s.apiClearConsoleLog).Methods("DELETE")
	wsOpRouter.HandleFunc("/console", s.apiConsole).Methods("GET")
	wsOpRouter.HandleFunc("/exec", s.apiExec).Methods("POST")
	wsOpRouter.HandleFunc("/exec", s.apiExecInteractive).Methods("GET")

	adminAuthM := authMiddleware{IAM: s.iam, NeedAdmin: true}
	internalWsOpRouter := r.PathPrefix("/internal/{username}").Subrouter()
	internalWsOpRouter.Use(adminAuthM.Middleware, s.getWebspaceMiddleware)
	internalWsOpRouter.HandleFunc("/ensure-started", s.internalAPIEnsureStarted).Methods("POST")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(data.AssetFile())))
	r.PathPrefix("/swagger/").Handler(httpswagger.Handler(
		httpswagger.URL("/static/api.yaml"),
		httpswagger.PersistAuth(true),
	))

	r.HandleFunc("/health", s.healthCheck)

	r.NotFoundHandler = http.HandlerFunc(apiNotFound)
	r.MethodNotAllowedHandler = http.HandlerFunc(apiMethodNotAllowed)

	return s
}

// Start begins listening
func (s *Server) Start(ctx context.Context) error {
	var err error
	s.lxd, err = lxd.ConnectLXD(s.Config.LXD.URL, &lxd.ConnectionArgs{
		TLSCA:              s.Config.LXD.TLS.CA,
		TLSServerCert:      s.Config.LXD.TLS.ServerCert,
		TLSClientCert:      s.Config.LXD.TLS.ClientCert,
		TLSClientKey:       s.Config.LXD.TLS.ClientKey,
		InsecureSkipVerify: s.Config.LXD.TLS.AllowInsecure,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to LXD: %w", err)
	}

	s.Webspaces, err = webspace.NewManager(&s.Config, s.iam, s.lxd)
	if err != nil {
		return fmt.Errorf("failed to create webspace manager: %w", err)
	}
	if err := s.Webspaces.Start(ctx); err != nil {
		return fmt.Errorf("failed to start webspace manager: %w", err)
	}

	if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop shuts down the server and listener
func (s *Server) Stop(ctx context.Context) error {
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop HTTP server: %w", err)
	}

	s.Webspaces.Shutdown(ctx)

	s.lxd.Disconnect()

	return nil
}

func apiNotFound(w http.ResponseWriter, r *http.Request) {
	util.JSONErrResponse(w, util.ErrNotFound, http.StatusNotFound)
}

func apiMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	util.JSONErrResponse(w, util.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	if !s.Webspaces.Healthy() {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
