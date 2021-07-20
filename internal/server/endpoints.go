package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	iam "github.com/netsoc/iam/client"
	"github.com/netsoc/webspaced/internal/webspace"
	"github.com/netsoc/webspaced/pkg/util"
)

var upgrader = websocket.Upgrader{}

func (s *Server) apiImages(w http.ResponseWriter, r *http.Request) {
	images, err := s.Webspaces.Images()
	if err != nil {
		util.JSONErrResponse(w, err, http.StatusInternalServerError)
		return
	}

	util.JSONResponse(w, images, http.StatusOK)
}

type createWebspaceReq struct {
	Image    string `json:"image"`
	Password string `json:"password"`
	SSH      bool   `json:"ssh"`
}
type createWebspaceRes struct {
	SSHPort uint16 `json:"sshPort"`
}

func (s *Server) apiGetWebspace(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	util.JSONResponse(w, ws, http.StatusOK)
}
func (s *Server) apiCreateWebspace(w http.ResponseWriter, r *http.Request) {
	var body createWebspaceReq
	if err := util.ParseJSONBody(&body, w, r); err != nil {
		return
	}

	user := r.Context().Value(keyUser).(*iam.User)
	sshKey := ""
	if body.SSH {
		if user.SshKey == nil {
			util.JSONErrResponse(w, util.ErrSSHKey, 0)
			return
		}
		sshKey = *user.SshKey
	}

	ws, err := s.Webspaces.Create(int(user.Id), body.Image, body.Password, sshKey)
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	util.JSONResponse(w, ws, http.StatusCreated)
}
func (s *Server) apiDeleteWebspace(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	if err := ws.Delete(); err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiSetWebspaceState(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	var err error
	switch r.Method {
	case "POST":
		err = ws.Boot()
	case "PATCH":
		err = ws.Sync(r.Context())
	case "PUT":
		err = ws.Reboot()
	case "DELETE":
		err = ws.Shutdown()
	}
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) apiGetWebspaceState(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	state, err := ws.State()
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	util.JSONResponse(w, state, http.StatusOK)
}

func (s *Server) apiGetWebspaceConfig(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	util.JSONResponse(w, ws.Config, http.StatusOK)
}
func (s *Server) apiUpdateWebspaceConfig(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	oldConf := ws.Config

	if err := util.ParseJSONBody(&ws.Config, w, r); err != nil {
		return
	}
	if err := ws.Save(); err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	util.JSONResponse(w, oldConf, http.StatusOK)
}

func (s *Server) apiGetWebspaceDomains(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	domains, err := ws.GetDomains(r.Context())
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	util.JSONResponse(w, domains, http.StatusOK)
}
func (s *Server) apiWebspaceDomain(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	d := mux.Vars(r)["domain"]

	var err error
	switch r.Method {
	case "POST":
		err = ws.AddDomain(d)
	case "DELETE":
		err = ws.RemoveDomain(r.Context(), d)
	}
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type addImplicitPortRes struct {
	EPort uint16 `json:"ePort"`
}

func (s *Server) apiGetWebspacePorts(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	util.JSONResponse(w, ws.Ports, http.StatusOK)
}
func (s *Server) apiWebspacePorts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	explicit := false
	eStr, iStr := "0", "0"
	if e, ok := vars["ePort"]; ok {
		explicit = true
		eStr = e
		iStr = vars["iPort"]
	} else {
		if r.Method == "DELETE" {
			eStr = vars["port"]
		}
		iStr = vars["port"]
	}

	e, err := strconv.ParseUint(eStr, 10, 16)
	if err != nil {
		util.JSONErrResponse(w, util.ErrBadPort, http.StatusBadRequest)
		return
	}
	external := uint16(e)

	i, err := strconv.ParseUint(iStr, 10, 16)
	if err != nil {
		util.JSONErrResponse(w, util.ErrBadPort, http.StatusBadRequest)
		return
	}
	internal := uint16(i)

	if explicit && external == 0 {
		util.JSONErrResponse(w, util.ErrBadPort, http.StatusBadRequest)
		return
	}

	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	switch r.Method {
	case "POST":
		external, err = ws.AddPort(external, internal)
		if err != nil {
			util.JSONErrResponse(w, err, 0)
			return
		}

		if !explicit {
			util.JSONResponse(w, addImplicitPortRes{external}, http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	case "DELETE":
		err = ws.RemovePort(external)
		if err != nil {
			util.JSONErrResponse(w, err, 0)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) apiConsoleLog(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	c, err := ws.Log()
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	if _, err := io.Copy(w, c); err != nil {
		util.JSONErrResponse(w, err, http.StatusInternalServerError)
	}
}

func (s *Server) apiClearConsoleLog(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	if err := ws.ClearLog(); err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) internalAPIEnsureStarted(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	ip, err := ws.EnsureStarted()
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	fmt.Fprint(w, ip)
}
