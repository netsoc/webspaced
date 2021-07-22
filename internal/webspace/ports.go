package webspace

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"

	k8sCore "k8s.io/api/core/v1"
	k8sMeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	k8sTypedCore "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	k8sRetry "k8s.io/client-go/util/retry"

	"github.com/netsoc/webspaced/internal/config"
	"github.com/netsoc/webspaced/pkg/util"
)

// PortHook represents a function to run before connecting to the backend
type PortHook func(f *PortForward) error

// PortForward represents an active port forwarding
type PortForward struct {
	ePort       uint16
	backendAddr *net.TCPAddr
	hook        PortHook
	listener    *net.TCPListener
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
	}, nil
}

func (f *PortForward) handleClient(client *net.TCPConn) {
	defer client.Close()
	if err := f.hook(f); err != nil {
		log.WithFields(log.Fields{
			"ePort":   f.ePort,
			"backend": f.backendAddr,
		}).WithError(err).Warn("Port forward hook execution failed")
		return
	}

	log.WithFields(log.Fields{
		"ePort":   f.ePort,
		"backend": f.backendAddr,
	}).Trace("Forwarding connection...")
	backend, err := net.DialTCP("tcp", nil, f.backendAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"ePort":   f.ePort,
			"backend": f.backendAddr,
		}).WithError(err).Warn("Port forward backend connection failed")
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

	wg.Wait()
	log.WithFields(log.Fields{
		"ePort":   f.ePort,
		"backend": f.backendAddr,
	}).Trace("Forwarded connection ended normally")
}

// Run starts the port forward
func (f *PortForward) Run() {
	for {
		client, err := f.listener.AcceptTCP()
		if err != nil {
			log.WithFields(log.Fields{
				"ePort":   f.ePort,
				"backend": f.backendAddr,
			}).WithError(err).Info("Ending port forward")
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
	svcName string
	svcAPI  k8sTypedCore.ServiceInterface

	forwards map[uint16]*PortForward
}

// NewPortsManager creates a new TCP port forward manager
func NewPortsManager(cfg *config.Config) (*PortsManager, error) {
	p := PortsManager{
		forwards: map[uint16]*PortForward{},
	}

	if cfg.Webspaces.Ports.KubernetesService != "" {
		k8sConf, err := clientcmd.BuildConfigFromFlags("", os.Getenv(clientcmd.RecommendedConfigPathEnvVar))
		if err != nil {
			return nil, fmt.Errorf("failed to load Kubernetes config: %w", err)
		}

		k8s, err := kubernetes.NewForConfig(k8sConf)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		p.svcAPI = k8s.CoreV1().Services(cfg.Traefik.Kubernetes.Namespace)
		p.svcName = cfg.Webspaces.Ports.KubernetesService
	}

	return &p, nil
}

// Add creates a new port forwarding
func (p *PortsManager) Add(ctx context.Context, e uint16, backendAddr *net.TCPAddr, hook PortHook) error {
	if _, ok := p.forwards[e]; ok {
		return util.ErrUsed
	}

	forward, err := NewPortForward(e, backendAddr, hook)
	if err != nil {
		return err
	}

	go forward.Run()
	p.forwards[e] = forward

	if p.svcName != "" {
		if err := k8sRetry.RetryOnConflict(k8sRetry.DefaultRetry, func() error {
			svc, err := p.svcAPI.Get(ctx, p.svcName, k8sMeta.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get Kubernetes Service: %w", err)
			}

			svcPort := k8sCore.ServicePort{
				Name:       "ws-fwd-" + strconv.Itoa(int(e)),
				Port:       int32(e),
				Protocol:   k8sCore.ProtocolTCP,
				TargetPort: intstr.FromInt(int(e)),
			}

			existing := false
			for i, sp := range svc.Spec.Ports {
				if sp.Port == int32(e) {
					log.WithFields(log.Fields{
						"ePort":   e,
						"backend": backendAddr,
					}).Debug("Kubernetes Service port already existed, overwriting")
					svc.Spec.Ports[i] = svcPort
					existing = true
				}
			}
			if !existing {
				svc.Spec.Ports = append(svc.Spec.Ports, svcPort)
			}

			_, err = p.svcAPI.Update(ctx, svc, k8sMeta.UpdateOptions{})
			return err
		}); err != nil {
			return fmt.Errorf("failed to update Kubernetes Service: %w", err)
		}
	}

	return nil
}

// Remove stops and removes a port forwarding
func (p *PortsManager) Remove(ctx context.Context, e uint16, updateK8s bool) error {
	forward, ok := p.forwards[e]
	if !ok {
		return util.ErrNotFound
	}

	if p.svcName != "" && updateK8s {
		if err := k8sRetry.RetryOnConflict(k8sRetry.DefaultRetry, func() error {
			svc, err := p.svcAPI.Get(ctx, p.svcName, k8sMeta.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get Kubernetes Service: %w", err)
			}

			index := -1
			for i, p := range svc.Spec.Ports {
				if p.Port == int32(e) {
					index = i
					break
				}
			}

			if index != -1 {
				end := len(svc.Spec.Ports) - 1
				svc.Spec.Ports[end], svc.Spec.Ports[index] = svc.Spec.Ports[index], svc.Spec.Ports[end]
				svc.Spec.Ports = svc.Spec.Ports[:end]

				if _, err := p.svcAPI.Update(ctx, svc, k8sMeta.UpdateOptions{}); err != nil {
					return err
				}
			} else {
				log.WithFields(log.Fields{
					"ePort":   e,
					"backend": forward.backendAddr,
				}).Warn("Kubernetes Service port not found")
			}

			return nil
		}); err != nil {
			return fmt.Errorf("failed to update Kubernetes Service: %w", err)
		}
	}

	forward.Stop()
	delete(p.forwards, e)
	return nil
}

// Trim removes port forwards that have been deleted
func (p *PortsManager) Trim(ctx context.Context, all []*Webspace) error {
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
			p.Remove(ctx, e, true)
		}
	}
	return nil
}

// AddAll adds / updates port forwards for a given webspace
func (p *PortsManager) AddAll(ctx context.Context, w *Webspace, addr string) error {
	for e, i := range w.Ports {
		// Using an existing port forward is validated externally - if this exists it belongs to us
		if _, ok := p.forwards[e]; ok {
			// Don't trigger a change in Kubernetes!
			if err := p.Remove(ctx, e, false); err != nil {
				return fmt.Errorf("failed to remove existing port forward: %w", err)
			}
		}

		hook := func(f *PortForward) error {
			log.WithFields(log.Fields{
				"uid":   w.UserID,
				"ePort": e,
				"iPort": i,
			}).Debug("Waiting for webspace to start to forward port")

			addr, err := w.EnsureStarted()
			if err != nil {
				return fmt.Errorf("failed to ensure webspace was started: %w", err)
			}

			backendAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", addr, i))
			if err != nil {
				panic(err)
			}
			f.backendAddr = backendAddr
			return nil
		}

		var backendAddr *net.TCPAddr
		if addr != "" {
			var err error
			backendAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", addr, i))
			if err != nil {
				panic(err)
			}

			// Only ensure started if we're not running already
			hook = func(_ *PortForward) error { return nil }
		}

		if err := p.Add(ctx, e, backendAddr, hook); err != nil {
			return fmt.Errorf("failed to add port forward for: %w", err)
		}
	}

	return nil
}

// Shutdown stops and removes all port forwards
func (p *PortsManager) Shutdown(ctx context.Context) {
	for e := range p.forwards {
		p.Remove(ctx, e, true)
	}
}
