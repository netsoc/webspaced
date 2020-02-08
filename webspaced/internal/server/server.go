package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

// Server is the main webspaced server struct
type Server struct {
	http *http.Server
}

// NewServer returns an initialized Server
func NewServer() *Server {
	r := mux.NewRouter()
	httpSrv := &http.Server{
		Handler: r,
	}

	srv := &Server{
		http: httpSrv,
	}
	r.HandleFunc("/", srv.index)

	return srv
}

// Start begins listening
func (s *Server) Start(sockPath string) error {
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}

	return s.http.Serve(listener)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello, world!\n")
}
