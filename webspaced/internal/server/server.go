package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	lxd "github.com/lxc/lxd/client"
)

// Server is the main webspaced server struct
type Server struct {
	lxd  lxd.InstanceServer
	http *http.Server
}

// NewServer returns an initialized Server
func NewServer() *Server {
	r := mux.NewRouter()
	httpSrv := &http.Server{
		Handler: r,
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

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello, world!\n")
}
func (s *Server) containers(w http.ResponseWriter, r *http.Request) {
	list, err := s.lxd.GetContainers()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}

	json.NewEncoder(w).Encode(list)
}
