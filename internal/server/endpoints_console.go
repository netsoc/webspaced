package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

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
	sockRW.Close()
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

type execInteractiveControl struct {
	Resize consoleResizeReq `json:"resize"`

	Signal int `json:"signal"`
}

func (s *Server) apiExecInteractive(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		util.JSONErrResponse(w, util.ErrWebsocket, 0)
		return
	}

	ws := r.Context().Value(keyWebspace).(*webspace.Webspace)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("Failed to upgrade HTTP connection")
	}

	var opts webspace.ExecOptions
	if err := conn.ReadJSON(&opts); err != nil {
		util.WSCloseError(conn, fmt.Errorf("failed to parse exec  request: %w", err))
		return
	}

	session, err := ws.ExecInteractive(opts)
	if err != nil {
		util.WSCloseError(conn, fmt.Errorf("failed to start interactive exec: %w", err))
		return
	}

	sockRW := util.NewWebsocketIO(conn, func(s string, rw *util.WebsocketIO) {
		var req execInteractiveControl
		if err := json.Unmarshal([]byte(s), &req); err != nil {
			rw.Mutex.Lock()
			defer rw.Mutex.Unlock()

			util.WSCloseError(rw.Conn, err)
			return
		}

		if req.Resize.Width != 0 && req.Resize.Height != 0 {
			if err := session.Resize(req.Resize.Width, req.Resize.Height); err != nil {
				rw.Mutex.Lock()
				defer rw.Mutex.Unlock()

				util.WSCloseError(rw.Conn, err)
				return
			}
		}
		if req.Signal != 0 {
			if err := session.Signal(req.Signal); err != nil {
				rw.Mutex.Lock()
				defer rw.Mutex.Unlock()

				util.WSCloseError(rw.Conn, err)
				return
			}
		}
	})

	errChan := make(chan error)

	go func() {
		exitCode, err := session.Await()
		if err != nil {
			errChan <- err
			return
		}

		sockRW.Mutex.Lock()
		defer sockRW.Mutex.Unlock()

		sockRW.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, strconv.Itoa(exitCode)))
		sockRW.Conn.Close()
		<-errChan
	}()

	pipe := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errChan <- err
	}
	go pipe(session.IO(), sockRW)
	go pipe(sockRW, session.IO())

	if err := <-errChan; err != nil {
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == websocket.CloseNormalClosure {
			return
		}

		sockRW.Mutex.Lock()
		defer sockRW.Mutex.Unlock()

		util.WSCloseError(sockRW.Conn, err)
		log.WithError(err).Error("Exec session failed")
	}
}
