package webspace

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gorilla/websocket"
	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	log "github.com/sirupsen/logrus"
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
				log.WithFields(log.Fields{
					"uid": w.UserID,
					"req": req,
				}).Trace("Making console resize request")

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

// ExecOptions specifies options for executing a command in a webspace
type ExecOptions struct {
	Command []string `json:"command"`

	User             uint32            `json:"user"`
	Group            uint32            `json:"group"`
	Environment      map[string]string `json:"environment"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	WorkingDirectory string            `json:"workingDirectory"`
}

// ExecSession represents an interactive webspace exec session
type ExecSession interface {
	IO() io.ReadWriteCloser
	Resize(int, int) error
	Signal(int) error
	Await() (int, error)
	ExitCode() int
}

type execSession struct {
	ws   *Webspace
	rw   io.ReadWriteCloser
	done chan bool
	op   lxd.Operation

	resizeChan chan resizeReq
	signalChan chan int

	resizeErrChan chan error
	signalErrChan chan error
	exitCode      int
}

func (s *execSession) onControl(conn *websocket.Conn) {
	for {
		select {
		case req := <-s.resizeChan:
			log.WithFields(log.Fields{
				"uid": s.ws.UserID,
				"req": req,
			}).Trace("Making exec resize request")
			s.resizeErrChan <- conn.WriteJSON(lxdApi.InstanceExecControl{
				Command: "window-resize",
				Args: map[string]string{
					"width":  strconv.Itoa(req.width),
					"height": strconv.Itoa(req.height),
				},
			})
		case sig := <-s.signalChan:
			log.WithFields(log.Fields{
				"uid":    s.ws.UserID,
				"signal": sig,
			}).Trace("Making signal request")
			s.signalErrChan <- conn.WriteJSON(lxdApi.InstanceExecControl{
				Command: "signal",
				Signal:  sig,
			})
		case <-s.done:
			return
		}
	}
}

func (s *execSession) IO() io.ReadWriteCloser {
	return s.rw
}

func (s *execSession) Resize(width, height int) error {
	s.resizeChan <- resizeReq{width, height}
	return <-s.resizeErrChan
}

func (s *execSession) Signal(sig int) error {
	s.signalChan <- sig
	return <-s.signalErrChan
}

func (s *execSession) Await() (int, error) {
	if err := s.op.Wait(); err != nil {
		return -1, fmt.Errorf("exec failed: %w", err)
	}

	opAPI := s.op.Get()
	<-s.done

	s.exitCode = int(opAPI.Metadata["return"].(float64))
	return s.exitCode, nil
}
func (s *execSession) ExitCode() int {
	return s.exitCode
}

// ExecInteractive runs a command in a webspace (with a PTY)
func (w *Webspace) ExecInteractive(opts ExecOptions) (ExecSession, error) {
	n := w.InstanceName()

	state, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return nil, fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
	}

	if state.StatusCode != lxdApi.Running {
		if err := w.Boot(); err != nil {
			return nil, fmt.Errorf("failed to start webspace: %w", err)
		}
	}

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	doneChan := make(chan bool)

	rw := consoleTerm{pr: outR, pw: inW}
	session := execSession{
		ws:   w,
		rw:   &rw,
		done: doneChan,

		resizeChan: make(chan resizeReq),
		signalChan: make(chan int),

		resizeErrChan: make(chan error),
		signalErrChan: make(chan error),
	}

	session.op, err = w.manager.lxd.ExecInstance(n, lxdApi.InstanceExecPost{
		WaitForWS:    true,
		Interactive:  true,
		RecordOutput: false,

		Command:     opts.Command,
		User:        opts.User,
		Group:       opts.Group,
		Environment: opts.Environment,
		Width:       opts.Width,
		Height:      opts.Height,
		Cwd:         opts.WorkingDirectory,
	}, &lxd.InstanceExecArgs{
		Stdin:  inR,
		Stdout: outW,
		Stderr: outW,

		Control:  session.onControl,
		DataDone: doneChan,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up exec: %w", err)
	}

	return &session, nil
}
