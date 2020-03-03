package server

import (
	"context"
	"errors"
	"net/http"

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
	case errors.Is(err, webspace.ErrExists), errors.Is(err, webspace.ErrRunning):
		return http.StatusConflict
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
