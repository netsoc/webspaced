package server

import (
	"encoding/binary"
	"errors"
	"net"
	"time"
)

const readTimeout = time.Second

const (
	reqGetPwUID uint8 = iota
	reqUserIsMember
)
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
		return "", errors.New("pw_gr_proxy GETPWUID returned non-ok status")
	}
	return string(buf[1:n]), nil
}

// UserIsMember checks if the given user is the member of the given group
func (p *PwGrProxy) UserIsMember(user string, group string) (bool, error) {
	c, err := net.Dial("unixpacket", p.SockPath)
	if err != nil {
		return false, err
	}
	defer c.Close()

	buf := make([]byte, 4096)
	buf[0] = reqUserIsMember
	copy(buf[1:], user)
	copy(buf[1+len(user)+1:], group)

	if _, err := c.Write(buf[:1+len(user)+1+len(group)+1]); err != nil {
		return false, err
	}

	c.SetReadDeadline(time.Now().Add(readTimeout))
	if _, err := c.Read(buf); err != nil {
		return false, err
	}

	if buf[0] != resOk {
		return false, errors.New("pw_gr_proxy USER_IS_MEMBER returned non-ok status")
	}

	if buf[1] == 1 {
		return true, nil
	}
	return false, nil
}
