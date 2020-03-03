package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lxd "github.com/lxc/lxd/client"
	"github.com/netsoc/webspace-ng/webspaced/internal/config"
	"github.com/netsoc/webspace-ng/webspaced/internal/webspace"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type key int

const (
	keyServer key = iota
	keyPcred
	keyUser
	keyWebspace
)

func recordConnUcred(ctx context.Context, c net.Conn) context.Context {
	if unixConn, isUnix := c.(*net.UnixConn); isUnix {
		f, _ := unixConn.File()
		pcred, _ := unix.GetsockoptUcred(int(f.Fd()), unix.SOL_SOCKET, unix.SO_PEERCRED)
		f.Close()

		return context.WithValue(ctx, keyPcred, pcred)
	}
	return ctx
}

// JSONResponse Sends a JSON payload in response to a HTTP request
func JSONResponse(w http.ResponseWriter, v interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		log.WithField("err", err).Error("Failed to serialize JSON payload")

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Failed to serialize JSON payload")
	}
}

type jsonError struct {
	Message string `json:"message"`
}

// JSONErrResponse Sends an `error` as a JSON object with a `message` property
func JSONErrResponse(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	enc.Encode(jsonError{err.Error()})
}

// ParseJSONBody attempts to parse the request body as JSON
func ParseJSONBody(v interface{}, w http.ResponseWriter, r *http.Request) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		JSONErrResponse(w, fmt.Errorf("failed to parse request body: %w", err), http.StatusBadRequest)
		return err
	}

	return nil
}

// UserMiddleware is a middleware for resolving Unix socket peer credentials to a name
func UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := r.Context().Value(keyServer).(*Server)
		pcred := r.Context().Value(keyPcred).(*unix.Ucred)

		username, err := s.pwGrProxy.LookupUID(pcred.Uid)
		if err != nil {
			username = fmt.Sprintf("u%v", pcred.Uid)
			log.WithFields(log.Fields{
				"err":      err,
				"fallback": username,
			}).Warn("Coudln't find username for UID, using fallback")
		}

		isAdmin, err := s.pwGrProxy.UserIsMember(username, s.Config.Webspaces.AdminGroup)
		if err != nil {
			log.WithFields(log.Fields{
				"err":   err,
				"user":  username,
				"group": s.Config.Webspaces.AdminGroup,
			}).Warn("Failed to check if user is in admin group")
		}

		if isAdmin || pcred.Uid == 0 {
			if reqUser := r.Header.Get("X-Webspace-User"); reqUser != "" {
				username = reqUser
			}
		}

		log.WithField("username", username).Trace("User authenticated")
		r = r.WithContext(context.WithValue(r.Context(), keyUser, username))
		next.ServeHTTP(w, r)
	})
}
func writeAccessLog(w io.Writer, params handlers.LogFormatterParams) {
	user := params.Request.Context().Value(keyUser).(string)
	log.WithFields(log.Fields{
		"user":    user,
		"agent":   params.Request.UserAgent(),
		"status":  params.StatusCode,
		"resSize": params.Size,
	}).Debugf("%v %v", params.Request.Method, params.URL.RequestURI())
}

// Server is the main webspaced server struct
type Server struct {
	Config    config.Config
	Webspaces *webspace.Manager
	lxd       lxd.InstanceServer
	http      *http.Server
	pwGrProxy *PwGrProxy
}

// NewServer returns an initialized Server
func NewServer(config config.Config) *Server {
	r := mux.NewRouter()
	httpSrv := &http.Server{
		Handler:     UserMiddleware(handlers.CustomLoggingHandler(nil, r, writeAccessLog)),
		ConnContext: recordConnUcred,
	}

	s := &Server{
		Config: config,
		http:   httpSrv,
	}

	r.HandleFunc("/v1/images", s.apiImages).Methods("GET")

	r.HandleFunc("/v1/webspace", s.apiCreateWebspace).Methods("POST")
	wsOpRouter := r.PathPrefix("/v1/webspace").Subrouter()
	wsOpRouter.Use(s.getWebspaceMiddleware)
	wsOpRouter.HandleFunc("", s.apiGetWebspace).Methods("GET")
	wsOpRouter.HandleFunc("", s.apiDeleteWebspace).Methods("DELETE")

	wsOpRouter.HandleFunc("/state", s.apiWebspaceState).Methods("POST", "PUT", "DELETE")

	wsOpRouter.HandleFunc("/config", s.apiGetWebspaceConfig).Methods("GET")
	wsOpRouter.HandleFunc("/config", s.apiUpdateWebspaceConfig).Methods("PATCH")

	r.NotFoundHandler = http.HandlerFunc(s.apiNotFound)

	return s
}

// Start begins listening
func (s *Server) Start() error {
	s.pwGrProxy = NewPwGrProxy(s.Config.PwGrProxySocket)
	s.http.BaseContext = func(_ net.Listener) context.Context {
		return context.WithValue(context.Background(), keyServer, s)
	}

	var err error
	if s.lxd, err = lxd.ConnectLXDUnix(s.Config.LXD.Socket, nil); err != nil {
		return err
	}
	if _, _, err := s.lxd.GetNetwork(s.Config.LXD.Network); err != nil {
		return fmt.Errorf("LXD returned error looking for network %v: %w", s.Config.LXD.Network, err)
	}

	if s.Webspaces, err = webspace.NewManager(&s.Config, s.lxd); err != nil {
		return fmt.Errorf("failed to initialize webspace manager: %v", err)
	}

	listener, err := net.Listen("unix", s.Config.BindSocket)
	if err != nil {
		return err
	}

	// Socket needs to be u=rw,g=rw,o=rw so anyone can access it (we'll do auth later)
	err = os.Chmod(s.Config.BindSocket, 0o666)
	if err != nil {
		return err
	}

	if err := s.http.Serve(listener); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop shuts down the server and listener
func (s *Server) Stop() error {
	return s.http.Close()
}

func (s *Server) apiNotFound(w http.ResponseWriter, r *http.Request) {
	JSONErrResponse(w, errors.New("API endpoint not found"), http.StatusNotFound)
}
