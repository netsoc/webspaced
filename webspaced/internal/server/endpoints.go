package server

import (
	"net/http"

	lxdApi "github.com/lxc/lxd/shared/api"
)

type simpleImage struct {
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

	resImages := make([]simpleImage, len(images))
	for i, image := range images {
		resImages[i] = simpleImage{
			Aliases:     image.Aliases,
			Fingerprint: image.Fingerprint,
			Properties:  image.ImagePut.Properties,
			Size:        image.Size,
		}
	}

	JSONResponse(w, resImages, http.StatusOK)
}
