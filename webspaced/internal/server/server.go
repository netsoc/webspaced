package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type key int

const (
	keyPcred key = iota
	keyUser
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

// UserMiddleware is a middleware for resolving Unix socket peer credentials to a name
func UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pcred := r.Context().Value(keyPcred).(*unix.Ucred)
		// TODO: check for membership of `webspace-admin` group
		if pcred.Uid == 0 {
			// TODO: use host passwd / groups proxy to resolve name
			u := "root"
			if reqUser := r.Header.Get("X-Webspace-User"); reqUser != "" {
				u = reqUser
			}

			r = r.WithContext(context.WithValue(r.Context(), keyUser, u))
			next.ServeHTTP(w, r)
		} else {
			JSONErrResponse(w, errors.New("Only root can execute commands right now"), http.StatusNotImplemented)
		}
	})
}

// Server is the main webspaced server struct
type Server struct {
	lxd  lxd.InstanceServer
	http *http.Server
}

// NewServer returns an initialized Server
func NewServer() *Server {
	r := mux.NewRouter()
	r.Use(UserMiddleware)
	httpSrv := &http.Server{
		Handler:     r,
		ConnContext: recordConnUcred,
	}

	s := &Server{
		http: httpSrv,
	}
	r.HandleFunc("/", s.index)
	r.HandleFunc("/containers", s.containers)

	return s
}

// Start begins listening
func (s *Server) Start(sockPath string) error {
	var err error
	s.lxd, err = lxd.ConnectLXDUnix("", nil)
	if err != nil {
		return err
	}

	var l *lxd.EventListener
	l, err = s.lxd.GetEvents()
	if err != nil {
		return err
	}
	l.AddHandler([]string{"lifecycle"}, s.onLxdEvent)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}

	// Socket needs to be u=rw,g=rw,o=rw so anyone can access it (we'll do auth later)
	err = os.Chmod(sockPath, 0o666)
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

func (s *Server) onLxdEvent(e lxdApi.Event) {
	var details map[string]interface{}
	json.Unmarshal(e.Metadata, &details)
	log.WithFields(log.Fields{
		"type":    e.Type,
		"details": details,
	}).Debug("lxd event")
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	u := r.Context().Value(keyUser).(string)
	JSONResponse(w, map[string]string{"user": u}, http.StatusOK)
}
func (s *Server) containers(w http.ResponseWriter, r *http.Request) {
	list, err := s.lxd.GetContainers()
	if err != nil {
		JSONErrResponse(w, err, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, list, http.StatusOK)
}
