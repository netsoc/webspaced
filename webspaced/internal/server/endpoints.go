package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/netsoc/webspace-ng/webspaced/internal/webspace"
)

type simpleImageRes struct {
	Aliases     []lxdApi.ImageAlias `json:"aliases"`
	Fingerprint string              `json:"fingerprint"`
	Properties  map[string]string   `json:"properties"`
	Size        int64               `json:"size"`
}

func (s *Server) apiImages(w http.ResponseWriter, r *http.Request) {
	images, err := s.lxd.GetImages()
	if err != nil {
		JSONErrResponse(w, err, http.StatusInternalServerError)
		return
	}

	resImages := make([]simpleImageRes, len(images))
	for i, image := range images {
		resImages[i] = simpleImageRes{
			Aliases:     image.Aliases,
			Fingerprint: image.Fingerprint,
			Properties:  image.ImagePut.Properties,
			Size:        image.Size,
		}
	}

	JSONResponse(w, resImages, http.StatusOK)
}

type createWebspaceReq struct {
	Image    string `json:"image"`
	Password string `json:"password"`
	SSHKey   string `json:"sshKey"`
}
type createWebspaceRes struct {
	SSHPort uint16 `json:"sshPort"`
}

func wsErrorToStatus(err error) int {
	switch {
	case errors.Is(err, webspace.ErrNotFound), errors.Is(err, webspace.ErrNotRunning):
		return http.StatusNotFound
	case errors.Is(err, webspace.ErrExists), errors.Is(err, webspace.ErrRunning), errors.Is(err, webspace.ErrUsed):
		return http.StatusConflict
	case errors.Is(err, webspace.ErrDomainUnverified), errors.Is(err, webspace.ErrBadPort),
		errors.Is(err, webspace.ErrTooManyPorts), errors.Is(err, webspace.ErrDefaultDomain):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (s *Server) getWebspaceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(keyUser).(string)
		ws, err := s.Webspaces.Get(user)
		if err != nil {
			JSONErrResponse(w, err, wsErrorToStatus(err))
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), keyWebspace, ws))
		next.ServeHTTP(w, r)
	})
}

func (s *Server) apiGetWebspace(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	JSONResponse(w, ws, http.StatusOK)
}
func (s *Server) apiCreateWebspace(w http.ResponseWriter, r *http.Request) {
	var body createWebspaceReq
	if err := ParseJSONBody(&body, w, r); err != nil {
		return
	}

	user := r.Context().Value(keyUser).(string)
	ws, err := s.Webspaces.Create(user, body.Image, body.Password, body.SSHKey)
	if err != nil {
		JSONErrResponse(w, err, wsErrorToStatus(err))
		return
	}

	JSONResponse(w, ws, http.StatusCreated)
}
func (s *Server) apiDeleteWebspace(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	if err := ws.Delete(); err != nil {
		JSONErrResponse(w, err, wsErrorToStatus(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiWebspaceState(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	var err error
	switch r.Method {
	case "POST":
		err = ws.Boot()
	case "PUT":
		err = ws.Reboot()
	case "DELETE":
		err = ws.Shutdown()
	}
	if err != nil {
		JSONErrResponse(w, err, wsErrorToStatus(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiGetWebspaceConfig(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	JSONResponse(w, ws.Config, http.StatusOK)
}
func (s *Server) apiUpdateWebspaceConfig(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	oldConf := ws.Config

	if err := ParseJSONBody(&ws.Config, w, r); err != nil {
		return
	}
	if err := ws.Save(); err != nil {
		JSONErrResponse(w, err, wsErrorToStatus(err))
		return
	}

	JSONResponse(w, oldConf, http.StatusOK)
}

func (s *Server) apiGetWebspaceDomains(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	JSONResponse(w, ws.Domains, http.StatusOK)
}
func (s *Server) apiWebspaceDomain(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	d := mux.Vars(r)["domain"]

	var err error
	switch r.Method {
	case "POST":
		err = ws.AddDomain(d)
	case "DELETE":
		err = ws.RemoveDomain(d)
	}
	if err != nil {
		JSONErrResponse(w, err, wsErrorToStatus(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type addImplicitPortRes struct {
	EPort uint16 `json:"ePort"`
}

func (s *Server) apiGetWebspacePorts(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	JSONResponse(w, ws.Ports, http.StatusOK)
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
		JSONErrResponse(w, webspace.ErrBadPort, http.StatusBadRequest)
		return
	}
	external := uint16(e)

	i, err := strconv.ParseUint(iStr, 10, 16)
	if err != nil {
		JSONErrResponse(w, webspace.ErrBadPort, http.StatusBadRequest)
		return
	}
	internal := uint16(i)

	if explicit && external == 0 {
		JSONErrResponse(w, webspace.ErrBadPort, http.StatusBadRequest)
		return
	}

	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)
	switch r.Method {
	case "POST":
		external, err = ws.AddPort(external, internal)
		if err != nil {
			JSONErrResponse(w, err, wsErrorToStatus(err))
			return
		}

		if !explicit {
			JSONResponse(w, addImplicitPortRes{external}, http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	case "DELETE":
		err = ws.RemovePort(external)
		if err != nil {
			JSONErrResponse(w, err, wsErrorToStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
