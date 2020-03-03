package server

import (
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
	case errors.Is(err, webspace.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, webspace.ErrExists):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (s *Server) apiCreateWebspace(w http.ResponseWriter, r *http.Request) {
	var body createWebspaceReq
	if err := ParseJSONBody(&body, w, r); err != nil {
		return
	}

	user := r.Context().Value(keyUser).(string)
	_, err := s.Webspaces.Create(user, body.Image, body.Password, body.SSHKey)
	if err != nil {
		JSONErrResponse(w, err, wsErrorToStatus(err))
		return
	}

	// TODO: Return SSH port forward
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiDeleteWebspace(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
