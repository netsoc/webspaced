package webspace

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	iam "github.com/netsoc/iam/client"
	"github.com/netsoc/webspaced/internal/config"
	"github.com/netsoc/webspaced/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Manager manages webspace containers
type Manager struct {
	config *config.Config
	lxd    lxd.InstanceServer
	iam    *iam.APIClient

	lxdWsUserRegex *regexp.Regexp
	lxdListener    *lxd.EventListener
	lxdLastEvent   time.Time
	lxdOK          bool

	locks   sync.Map
	traefik Traefik
	ports   *PortsManager
}

// NewManager returns a new Manager instance
func NewManager(cfg *config.Config, iam *iam.APIClient, l lxd.InstanceServer) (*Manager, error) {
	var traefik Traefik
	var err error
	switch cfg.Traefik.Provider {
	case "kubernetes":
		traefik, err = NewTraefikKubernetes(cfg)
	case "redis", "":
		traefik = NewTraefikRedis(cfg)
	default:
		return nil, util.ErrTraefikProvider
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initializae Traefik config provider %v: %w", cfg.Traefik.Provider, err)
	}

	ports, err := NewPortsManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize port forwards manager: %w", err)
	}

	return &Manager{
		config: cfg,
		lxd:    l,
		iam:    iam,

		lxdWsUserRegex: regexp.MustCompile(fmt.Sprintf(lxdEventUserRegexTpl, cfg.Webspaces.InstancePrefix)),
		lxdListener:    nil,
		traefik:        traefik,
		ports:          ports,
	}, nil
}

func (m *Manager) setupLXDListener() error {
	l, err := m.lxd.GetEvents()
	if err != nil {
		return err
	}

	_, err = l.AddHandler([]string{"lifecycle"}, m.onLxdEvent)
	if err != nil {
		return fmt.Errorf("failed to add LXD event handler: %w", convertLXDError(err))
	}

	m.lxdListener = l
	return nil
}

// Lock locks a webspace
func (m *Manager) Lock(uid int) {
	v, _ := m.locks.LoadOrStore(uid, &sync.Mutex{})
	v.(*sync.Mutex).Lock()
}

// Unlock unlocks a webspace
func (m *Manager) Unlock(uid int) {
	v, _ := m.locks.Load(uid)
	v.(*sync.Mutex).Unlock()
}

func (m *Manager) syncAll(ctx context.Context) error {
	log.Debug("Clearing all existing Traefik configs")
	m.traefik.ClearAll(ctx)

	webspaces, err := m.GetAll()
	if err != nil {
		return fmt.Errorf("failed to retrieve all webspaces: %w", err)
	}
	if err := m.ports.Trim(ctx, webspaces); err != nil {
		return fmt.Errorf("failed to trim port forwards: %w", err)
	}

	var wg sync.WaitGroup
	for _, w := range webspaces {
		state, _, err := m.lxd.GetInstanceState(w.InstanceName())
		if err != nil {
			return fmt.Errorf("failed to retrieve LXD instance state: %w", convertLXDError(err))
		}

		running := state.StatusCode == lxdApi.Running
		log.WithFields(log.Fields{
			"uid":     w.UserID,
			"running": running,
		}).Debug("Syncing Traefik / port forwarding config")

		wg.Add(1)
		ws := w
		go func() {
			defer wg.Done()
			if err := func() error {
				var addr string
				if running {
					addr, err = ws.AwaitIP()
					if err != nil {
						return fmt.Errorf("failed to get instance IP address: %w", err)
					}
				}

				if err := m.traefik.ClearConfig(ctx, ws.InstanceName()); err != nil {
					return fmt.Errorf("failed to clear traefik config: %w", err)
				}
				if err := m.traefik.GenerateConfig(ctx, ws, addr); err != nil {
					return fmt.Errorf("failed to update traefik config: %w", err)
				}

				if err := m.ports.AddAll(ctx, ws, addr); err != nil {
					return fmt.Errorf("failed to set up port forwards: %w", err)
				}

				return nil
			}(); err != nil {
				log.
					WithError(err).
					WithField("uid", ws.UserID).
					Error("Failed to sync config")
			}
		}()
	}
	wg.Wait()

	return nil
}

// Start starts the webspace manager
func (m *Manager) Start(ctx context.Context) error {
	log.Info("Generating initial Traefik / port forwarding configs")
	if err := m.syncAll(ctx); err != nil {
		return fmt.Errorf("failed to sync Traefik / port forwarding configs: %w", err)
	}

	if err := m.setupLXDListener(); err != nil {
		return fmt.Errorf("failed to set up LXD event listener: %w", convertLXDError(err))
	}
	m.lxdOK = true

	go func() {
		back := backoff.NewExponentialBackOff()
		back.MaxElapsedTime = 0

		for {
			if err := m.lxdListener.Wait(); err != nil {
				log.
					WithError(err).
					Warn("LXD event listener failed, restarting...")
				m.lxdListener = nil
				m.lxdOK = false

				back.Reset()
				backoff.RetryNotify(m.setupLXDListener, back, func(err error, retry time.Duration) {
					log.
						WithError(err).
						WithField("retry", retry).
						Warn("LXD event listener reconnect failed, retrying...")
				})
				log.Info("LXD listener reconnect succeeded")

				log.Info("Re-syncing all configs after listener reconnect")
				back.Reset()
				backoff.RetryNotify(func() error {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					return m.syncAll(ctx)
				}, back, func(err error, retry time.Duration) {
					log.
						WithError(err).
						WithField("retry", retry).
						Warn("Webspace config sync failed, retrying...")
				})

				m.lxdOK = true
				continue
			}

			return
		}
	}()

	return nil
}

// Healthy returns true if the manager is healthy
func (m *Manager) Healthy() bool {
	return m.lxdOK
}

// Shutdown stops the webspace manager
func (m *Manager) Shutdown(ctx context.Context) {
	if m.lxdListener != nil {
		m.lxdListener.Disconnect()
	}
	m.ports.Shutdown(ctx)

	if err := m.traefik.ClearAll(ctx); err != nil {
		log.WithError(err).Warn("Failed to clear Traefik configs")
	}
}

func (m *Manager) lxdInstanceName(uid int) string {
	return fmt.Sprintf("%vu%v", m.config.Webspaces.InstancePrefix, uid)
}

type lxdEventDetails struct {
	Action string
	Source string
}

func (m *Manager) onLxdEvent(e lxdApi.Event) {
	if e.Timestamp == m.lxdLastEvent {
		// TODO: Why does this happen?
		log.Warn("Duplicate LXD event detected, ignoring")
		return
	}
	m.lxdLastEvent = e.Timestamp

	var details lxdEventDetails
	if err := json.Unmarshal(e.Metadata, &details); err != nil {
		// Event doesn't have the fields we want, ignore
		return
	}

	match := m.lxdWsUserRegex.FindStringSubmatch(details.Source)
	if len(match) == 0 {
		// Not a webspace instance
		return
	}
	uid, err := strconv.Atoi(match[1])
	if err != nil {
		log.WithError(err).Error("Failed to parse user ID")
		return
	}

	ctx := context.Background()
	m.Lock(uid)
	defer m.Unlock(uid)

	if err := m.traefik.ClearConfig(ctx, m.lxdInstanceName(uid)); err != nil {
		log.WithField("uid", uid).WithError(err).Error("Failed to clear Traefik config")
		return
	}

	all, err := m.GetAll()
	if err != nil {
		log.WithError(err).Error("Failed to get all webspaces")
		return
	}
	if err := m.ports.Trim(ctx, all); err != nil {
		log.WithError(err).Error("Failed to trim port forwards")
		return
	}

	match = lxdEventActionRegex.FindStringSubmatch(details.Action)
	if len(match) == 0 {
		return
	}
	action := match[1]

	if action == "deleted" {
		return
	}

	w, err := m.Get(uid, nil)
	if err != nil {
		log.WithField("uid", uid).WithError(err).Error("Failed to retrieve webspace")
		return
	}

	var running bool
	switch action {
	case "started", "restarted":
		running = true
	case "shutdown", "created":
		running = false
	case "updated":
		state, _, err := m.lxd.GetInstanceState(w.InstanceName())
		if err != nil {
			log.WithError(err).Error("Failed to retrieve LXD instance state")
			return
		}

		running = state.StatusCode == lxdApi.Running
	default:
		log.WithFields(log.Fields{
			"uid":    w.UserID,
			"action": action,
		}).Warn("Unknown LXD action")
		return
	}

	var addr string
	if running {
		addr, err = w.AwaitIP()
		if err != nil {
			log.WithError(err).Error("Failed to get instance IP address")
			return
		}
	}

	log.WithFields(log.Fields{
		"uid":     w.UserID,
		"running": running,
		"action":  action,
	}).Debug("Updating Traefik / port forward config")

	if err := m.traefik.GenerateConfig(ctx, w, addr); err != nil {
		log.WithField("user", w.UserID).WithError(err).Error("Failed to update Traefik config")
		return
	}

	if err := m.ports.AddAll(ctx, w, addr); err != nil {
		log.WithField("user", w.UserID).WithError(err).Error("Failed to update port forwards")
	}
}

func (m *Manager) instanceToWebspace(i *lxdApi.Instance) (*Webspace, error) {
	w := &Webspace{
		manager: m,
	}

	confJSON, ok := i.Config[lxdConfigKey]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve webspace instance configuration from LXD")
	}
	if err := json.Unmarshal([]byte(confJSON), w); err != nil {
		return nil, fmt.Errorf("failed to parse webspace configuration stored in LXD: %w", err)
	}

	if w.Config.StartupDelay < 0 {
		return nil, util.ErrBadValue
	}

	return w, nil
}

func (m *Manager) lxdState(name string, action string) error {
	op, err := m.lxd.UpdateContainerState(name, lxdApi.ContainerStatePut{
		Action:  action,
		Timeout: -1,
	}, "")
	if err != nil {
		return fmt.Errorf("failed to change LXD instance state: %w", convertLXDError(err))
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to change LXD instance state: %w", convertLXDError(err))
	}
	return nil
}

// Image represents an LXD image
type Image struct {
	Aliases     []lxdApi.ImageAlias `json:"aliases"`
	Fingerprint string              `json:"fingerprint"`
	Properties  map[string]string   `json:"properties"`
	Size        int64               `json:"size"`
}

// Images gets a list of available images to create webspaces from
func (m *Manager) Images() ([]Image, error) {
	lxdImages, err := m.lxd.GetImages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve images from LXD: %w", convertLXDError(err))
	}

	images := make([]Image, len(lxdImages))
	for i, li := range lxdImages {
		images[i] = Image{
			li.Aliases,
			li.Fingerprint,
			li.Properties,
			li.Size,
		}
	}

	return images, nil
}

// Get retrieves a Webspace instance from LXD
func (m *Manager) Get(uid int, userHint *iam.User) (*Webspace, error) {
	if userHint != nil && int(userHint.Id) != uid {
		return nil, util.ErrUIDMismatch
	}

	w := &Webspace{
		manager: m,
		user:    userHint,

		UserID: uid,
	}
	n := w.InstanceName()

	i, _, err := m.lxd.GetInstance(n)
	if err != nil {
		return nil, fmt.Errorf("failed to get LXD instance: %w", convertLXDError(err))
	}

	return m.instanceToWebspace(i)
}

// GetAll retrieves all the webspaces
func (m *Manager) GetAll() ([]*Webspace, error) {
	instances, err := m.lxd.GetInstances(lxdApi.InstanceTypeContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve LXD instances: %w", convertLXDError(err))
	}

	var webspaces []*Webspace
	for _, i := range instances {
		if _, ok := i.Config[lxdConfigKey]; !ok {
			continue
		}

		w, err := m.instanceToWebspace(&i)
		if err != nil {
			return nil, err
		}
		webspaces = append(webspaces, w)
	}
	return webspaces, nil
}

// Create creates a new webspace container via LXD
func (m *Manager) Create(uid int, image string, password string, sshKey string) (*Webspace, error) {
	m.Lock(uid)
	defer m.Unlock(uid)

	w := &Webspace{
		manager: m,

		UserID:  uid,
		Config:  m.config.Webspaces.ConfigDefaults,
		Domains: []string{},
		Ports:   map[uint16]uint16{},
	}
	n := w.InstanceName()

	if !util.IsSHA256(image) {
		alias, _, err := m.lxd.GetImageAlias(image)
		if err != nil {
			return nil, fmt.Errorf("failed to get image alias: %w", err)
		}

		image = alias.Target
	}

	lxdConf, err := w.lxdConfig()
	if err != nil {
		return nil, err
	}

	op, err := m.lxd.CreateInstance(lxdApi.InstancesPost{
		Type: lxdApi.InstanceTypeContainer,
		Name: n,
		Source: lxdApi.InstanceSource{
			Type:        "image",
			Fingerprint: image,
		},
		InstancePut: lxdApi.InstancePut{
			Ephemeral: false,
			Profiles:  []string{m.config.Webspaces.LXDProfile},
			Config: map[string]string{
				lxdConfigKey: lxdConf,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LXD instance: %w", convertLXDError(err))
	}

	if err := op.Wait(); err != nil {
		return nil, fmt.Errorf("failed to create LXD instance: %w", convertLXDError(err))
	}

	if password != "" || sshKey != "" {
		if _, err := w.EnsureStarted(); err != nil {
			return nil, err
		}
	}
	if password != "" {
		if stdout, stderr, err := w.simpleExec(fmt.Sprintf(`echo "root:%v" | chpasswd`, password)); err != nil {
			log.WithFields(log.Fields{
				"uid":    uid,
				"stdout": stdout,
				"stderr": stderr,
			}).WithError(err).Error("Failed to set root password")
			return nil, fmt.Errorf("failed to set root password: %w", err)
		}
	}
	if sshKey != "" {
		img, _, err := m.lxd.GetImage(image)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve LXD image info: %w", convertLXDError(err))
		}

		if os, ok := img.Properties["os"]; ok {
			var cmd string
			switch strings.ToLower(os) {
			case "alpine":
				cmd = "apk update && apk add dropbear && rc-update add dropbear"
			case "archlinux":
				cmd = "pacman -Sy --noconfirm openssh && systemctl enable sshd"
			case "ubuntu", "debian":
				cmd = "apt-get -qy update && apt-get -qy install openssh-server"
			case "fedora", "centos":
				cmd = "dnf install -qy openssh-server && systemctl enable sshd"
			default:
				log.WithField("os", os).Warn("Unknown OS, unable to install sshd")
			}

			if cmd != "" {
				if stdout, stderr, err := w.simpleExec(cmd); err != nil {
					log.WithFields(log.Fields{
						"uid":    uid,
						"stdout": stdout,
						"stderr": stderr,
					}).WithError(err).Error("Failed to install sshd")
					return nil, fmt.Errorf("failed to install sshd: %w", err)
				}

				cmd := fmt.Sprintf(`mkdir -p /root/.ssh && echo "%v" > /root/.ssh/authorized_keys`, sshKey)
				if stdout, stderr, err := w.simpleExec(cmd); err != nil {
					log.WithFields(log.Fields{
						"uid":    uid,
						"stdout": stdout,
						"stderr": stderr,
					}).WithError(err).Error("Failed to store ssh public key")
					return nil, fmt.Errorf("failed to store ssh public key: %w", err)
				}

				if _, err := w.AddPort(0, 22); err != nil {
					return nil, fmt.Errorf("failed to add SSH port forward: %w", err)
				}
			}
		} else {
			log.WithField("fingerprint", image).Warn("Image has no `os` property, unable to install sshd")
		}
	}
	if password != "" || sshKey != "" {
		if err := w.Shutdown(); err != nil {
			return nil, err
		}
	}

	return w, nil
}
