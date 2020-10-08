package util

import (
	"io"
	"io/ioutil"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// WSCloseError closes a websocket with an error message
func WSCloseError(conn *websocket.Conn, err error) {
	msg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error())
	if err := conn.WriteMessage(websocket.CloseMessage, msg); err != nil {
		log.WithError(err).Error("Failed to send console close message")
	}

	conn.Close()
}

// WebsocketIO is a wrapper implementing ReadWriteCloser on top of websocket
type WebsocketIO struct {
	Conn  *websocket.Conn
	Mutex sync.Mutex

	textHandler func(string, *WebsocketIO)
	reader      io.Reader
}

// NewWebsocketIO creates a new websocket ReadWriteCloser wrapper
func NewWebsocketIO(c *websocket.Conn, textHandler func(string, *WebsocketIO)) *WebsocketIO {
	return &WebsocketIO{
		Conn: c,

		textHandler: textHandler,
	}
}

func (w *WebsocketIO) Read(p []byte) (n int, err error) {
	for {
		// First read from this message
		if w.reader == nil {
			var mt int

			mt, w.reader, err = w.Conn.NextReader()
			if err != nil {
				return -1, err
			}

			if mt == websocket.CloseMessage {
				return 0, io.EOF
			}

			if mt == websocket.TextMessage {
				d, err := ioutil.ReadAll(w.reader)
				if err != nil {
					if err == io.EOF {
						err = io.ErrUnexpectedEOF
					}
					return -1, err
				}

				w.textHandler(string(d), w)
				continue
			}
		}

		// Perform the read itself
		n, err := w.reader.Read(p)
		if err == io.EOF {
			// At the end of the message, reset reader
			w.reader = nil
			return n, nil
		}

		if err != nil {
			return -1, err
		}

		return n, nil
	}
}

func (w *WebsocketIO) Write(p []byte) (n int, err error) {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	wr, err := w.Conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return -1, err
	}
	defer wr.Close()

	n, err = wr.Write(p)
	if err != nil {
		return -1, err
	}

	return n, nil
}

// Close sends a control message indicating the stream is finished, but it does not actually close
// the socket.
func (w *WebsocketIO) Close() error {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()

	// Target expects to get a control message indicating stream is finished.
	w.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Stream shutting down"))
	return w.Conn.Close()
}
