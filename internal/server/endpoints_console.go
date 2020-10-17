package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/netsoc/webspaced/internal/webspace"
	"github.com/netsoc/webspaced/pkg/util"
	log "github.com/sirupsen/logrus"
)

type consoleResizeReq struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (s *Server) apiConsole(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		util.JSONErrResponse(w, util.ErrWebsocket, 0)
		return
	}

	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("Failed to upgrade HTTP connection")
	}

	var initialSize consoleResizeReq
	if err := conn.ReadJSON(&initialSize); err != nil {
		util.WSCloseError(conn, fmt.Errorf("failed to parse resize request: %w", err))
		return
	}

	rw, resize, err := ws.Console(initialSize.Width, initialSize.Height)
	if err != nil {
		util.WSCloseError(conn, fmt.Errorf("failed to connect to console: %w", err))
		return
	}
	defer rw.Close()

	sockRW := util.NewWebsocketIO(conn, func(s string, rw *util.WebsocketIO) {
		var req consoleResizeReq
		if err := json.Unmarshal([]byte(s), &req); err != nil {
			rw.Mutex.Lock()
			defer rw.Mutex.Unlock()

			util.WSCloseError(rw.Conn, err)
			return
		}

		if err := resize(req.Width, req.Height); err != nil {
			rw.Mutex.Lock()
			defer rw.Mutex.Unlock()

			util.WSCloseError(rw.Conn, err)
			return
		}
	})

	errChan := make(chan error)
	pipe := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errChan <- err
	}
	go pipe(rw, sockRW)
	go pipe(sockRW, rw)

	if err := <-errChan; err != nil {
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == websocket.CloseNormalClosure {
			return
		}

		log.WithError(err).Error("Console session failed")
	}
}

type execReq struct {
	Command string `json:"command"`
}
type execRes struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func (s *Server) apiExec(w http.ResponseWriter, r *http.Request) {
	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	var body execReq
	if err := util.ParseJSONBody(&body, w, r); err != nil {
		return
	}

	code, stdout, stderr, err := ws.Exec(body.Command, true)
	if err != nil {
		util.JSONErrResponse(w, err, 0)
		return
	}

	util.JSONResponse(w, execRes{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: code,
	}, http.StatusOK)
}
