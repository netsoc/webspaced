package webspace

import (
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
)

// PortHook represents a function to run before connecting to the backend
type PortHook func(f *PortForward) error

// PortForward represents an active port forwarding
type PortForward struct {
	ePort       uint16
	backendAddr *net.TCPAddr
	hook        PortHook
	listener    *net.TCPListener
	shutdown    chan struct{}
}

// NewPortForward creates and starts a port forward
func NewPortForward(e uint16, backendAddr *net.TCPAddr, hook PortHook) (*PortForward, error) {
	frontendAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%v", e))
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", frontendAddr)
	if err != nil {
		return nil, err
	}

	return &PortForward{
		e,
		backendAddr,
		hook,
		listener,
		make(chan struct{}),
	}, nil
}

func (f *PortForward) handleClient(client *net.TCPConn) {
	defer client.Close()
	if err := f.hook(f); err != nil {
		log.WithFields(log.Fields{
			"ePort":   f.ePort,
			"backend": f.backendAddr,
			"err":     err,
		}).Warn("Port forward hook execution failed")
		return
	}

	backend, err := net.DialTCP("tcp", nil, f.backendAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"ePort":   f.ePort,
			"backend": f.backendAddr,
			"err":     err,
		}).Warn("Port forward backend connection failed")
		return
	}
	defer backend.Close()

	var wg sync.WaitGroup
	pipe := func(dst, src *net.TCPConn) {
		io.Copy(dst, src)
		src.CloseRead()
		dst.CloseWrite()
		wg.Done()
	}

	wg.Add(2)
	go pipe(client, backend)
	go pipe(backend, client)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-f.shutdown:
	}
}

// Run starts the port forward
func (f *PortForward) Run() {
	for {
		client, err := f.listener.AcceptTCP()
		if err != nil {
			log.WithFields(log.Fields{
				"ePort":   f.ePort,
				"backend": f.backendAddr,
				"err":     err,
			}).Info("Ending port forward")
			close(f.shutdown)
			return
		}

		go f.handleClient(client)
	}
}

// Stop shuts down the port forward
func (f *PortForward) Stop() {
	f.listener.Close()
}

// PortsManager manages TCP port forwarding
type PortsManager struct {
	forwards map[uint16]*PortForward
}

// NewPortsManager creates a new TCP port forward manager
func NewPortsManager() *PortsManager {
	return &PortsManager{map[uint16]*PortForward{}}
}

// Add creates a new port forwarding
func (p *PortsManager) Add(e uint16, backendAddr *net.TCPAddr, hook PortHook) error {
	if _, ok := p.forwards[e]; ok {
		return ErrUsed
	}

	forward, err := NewPortForward(e, backendAddr, hook)
	if err != nil {
		return err
	}

	go forward.Run()
	p.forwards[e] = forward
	return nil
}

// Remove stops and removes a port forwarding
func (p *PortsManager) Remove(e uint16) error {
	forward, ok := p.forwards[e]
	if !ok {
		return ErrNotFound
	}

	forward.Stop()
	delete(p.forwards, e)
	return nil
}

// Trim removes port forwards that have been deleted
func (p *PortsManager) Trim(all []*Webspace) error {
	allPorts := make(map[uint16]struct{})
	for _, w := range all {
		for e := range w.Ports {
			if _, ok := allPorts[e]; ok {
				return fmt.Errorf("more than one webspace uses external port %v", e)
			}

			allPorts[e] = struct{}{}
		}
	}

	for e := range p.forwards {
		if _, ok := allPorts[e]; !ok {
			p.Remove(e)
		}
	}
	return nil
}

// AddAll adds / updates port forwards for a given webspace
func (p *PortsManager) AddAll(w *Webspace, addr string) error {
	for e := range w.Ports {
		if _, ok := p.forwards[e]; ok {
			p.Remove(e)
		}

		hook := func(p *PortForward) error {
			if err := w.EnsureStarted(); err != nil {
				return fmt.Errorf("failed to ensure webspace was started: %w", err)
			}

			addr, err := w.AwaitIP()
			if err != nil {
				return fmt.Errorf("failed to await webspace IP: %w", err)
			}

			backendAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", addr, e))
			if err != nil {
				panic(err)
			}
			p.backendAddr = backendAddr
			return nil
		}

		var backendAddr *net.TCPAddr
		if addr != "" {
			var err error
			backendAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", addr, e))
			if err != nil {
				panic(err)
			}

			// Only ensure started if we're not running already
			hook = func(_ *PortForward) error { return nil }
		}

		if err := p.Add(e, backendAddr, hook); err != nil {
			return fmt.Errorf("failed to add port forward for: %w", err)
		}
	}

	return nil
}

// Shutdown stops and removes all port forwards
func (p *PortsManager) Shutdown() {
	for _, forward := range p.forwards {
		forward.Stop()
	}
}
