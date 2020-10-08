package webspace

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gorilla/websocket"
	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/sirupsen/logrus"
)

type consoleTerm struct {
	pr io.Reader
	pw io.Writer
}

func (t *consoleTerm) Read(p []byte) (n int, err error) {
	return t.pr.Read(p)
}
func (t *consoleTerm) Write(p []byte) (n int, err error) {
	return t.pw.Write(p)
}
func (t *consoleTerm) Close() error {
	return nil
}

type consoleRW struct {
	close chan bool
	pr    io.Reader
	pw    io.Writer
}

func (rw *consoleRW) Read(p []byte) (n int, err error) {
	return rw.pr.Read(p)
}
func (rw *consoleRW) Write(p []byte) (n int, err error) {
	return rw.pw.Write(p)
}
func (rw *consoleRW) Close() error {
	close(rw.close)
	return nil
}

type resizeReq struct {
	width  int
	height int
}

// Console attaches to the webspace's `/dev/console`
func (w *Webspace) Console(width, height int) (io.ReadWriteCloser, func(int, int) error, error) {
	n := w.InstanceName()

	state, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
	}

	if state.StatusCode != lxdApi.Running {
		if err := w.Boot(); err != nil {
			return nil, nil, fmt.Errorf("failed to start webspace: %w", err)
		}
	}

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	rw := consoleRW{close: make(chan bool), pr: outR, pw: inW}
	resizeChan := make(chan resizeReq)
	resizeErrChan := make(chan error)
	controlHandler := func(conn *websocket.Conn) {
		for {
			select {
			case req := <-resizeChan:
				logrus.WithFields(logrus.Fields{
					"uid": w.UserID,
					"req": req,
				}).Trace("Making resize request")

				resizeErrChan <- conn.WriteJSON(lxdApi.InstanceConsoleControl{
					Command: "window-resize",
					Args: map[string]string{
						"width":  strconv.Itoa(req.width),
						"height": strconv.Itoa(req.height),
					},
				})
			case <-rw.close:
				return
			}
		}
	}

	_, err = w.manager.lxd.ConsoleInstance(n, lxdApi.InstanceConsolePost{
		Width:  width,
		Height: height,
	}, &lxd.InstanceConsoleArgs{
		Terminal:          &consoleTerm{pr: inR, pw: outW},
		ConsoleDisconnect: rw.close,
		Control:           controlHandler,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set up console: %w", err)
	}

	resize := func(w, h int) error {
		resizeChan <- resizeReq{w, h}
		return <-resizeErrChan
	}

	return &rw, resize, nil
}
