package server

import (
	"encoding/binary"
	"errors"
	"net"
	"time"
)

const readTimeout = time.Second

const reqGetPwUID uint8 = 0
const (
	resOk uint8 = iota
	resErr
)

// HACK: Should really evaluate endianness even if this is most likely only ever going to run on amd64 and _maybe_ ARM
var hostEndian binary.ByteOrder = binary.LittleEndian

// PwGrProxy proxies requests for libc functions like `getpwuid()`
type PwGrProxy struct {
	SockPath string
}

// NewPwGrProxy creates a connection to a passwd / groups proxy
func NewPwGrProxy(sockPath string) *PwGrProxy {
	return &PwGrProxy{sockPath}
}

// LookupUID retrieves the username for a UID
func (p *PwGrProxy) LookupUID(uid uint32) (string, error) {
	c, err := net.Dial("unixpacket", p.SockPath)
	if err != nil {
		return "", err
	}
	defer c.Close()

	buf := make([]byte, 4096)
	buf[0] = reqGetPwUID
	hostEndian.PutUint32(buf[1:], uid)

	if _, err := c.Write(buf[:5]); err != nil {
		return "", err
	}

	c.SetReadDeadline(time.Now().Add(readTimeout))
	n, err := c.Read(buf)
	if err != nil {
		return "", err
	}

	if buf[0] != resOk {
		return "", errors.New("pw_gr_proxy returned non-ok status")
	}
	return string(buf[1:n]), nil
}
