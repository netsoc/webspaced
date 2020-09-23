package webspace

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/netsoc/webspaced/internal/config"
	log "github.com/sirupsen/logrus"
)

// Manager manages webspace containers
type Manager struct {
	config         *config.Config
	lxd            lxd.InstanceServer
	lxdWsUserRegex *regexp.Regexp
	lxdListener    *lxd.EventListener
	traefik        *Traefik
	ports          *PortsManager
}

// NewManager returns a new Manager instance
func NewManager(cfg *config.Config, l lxd.InstanceServer) *Manager {
	return &Manager{
		cfg,
		l,
		regexp.MustCompile(fmt.Sprintf(lxdEventUserRegexTpl, cfg.Webspaces.InstanceSuffix)),
		nil,
		NewTraefik(cfg),
		NewPortsManager(),
	}
}

// Start starts the webspace manager
func (m *Manager) Start() error {
	webspaces, err := m.GetAll()
	if err != nil {
		return fmt.Errorf("failed to retrieve all webspaces: %w", err)
	}
	for _, w := range webspaces {
		state, _, err := m.lxd.GetInstanceState(w.InstanceName())
		if err != nil {
			return fmt.Errorf("failed to retrieve LXD instance state: %w", convertLXDError(err))
		}

		running := state.StatusCode == lxdApi.Running
		log.WithFields(log.Fields{
			"user":    w.User,
			"running": running,
		}).Debug("Generating initial Traefik / port forwarding config")

		var addr string
		if running {
			addr, err = w.AwaitIP()
			if err != nil {
				return fmt.Errorf("failed to get instance IP address: %w", err)
			}
		}

		if err := m.traefik.GenerateConfig(w, addr); err != nil {
			return fmt.Errorf("failed to update traefik config: %w", err)
		}

		if err := m.ports.AddAll(w, addr); err != nil {
			return fmt.Errorf("failed to set up port forwards: %w", err)
		}
	}

	m.lxdListener, err = m.lxd.GetEvents()
	if err != nil {
		return fmt.Errorf("failed to get LXD event listener: %w", convertLXDError(err))
	}
	m.lxdListener.AddHandler([]string{"lifecycle"}, m.onLxdEvent)

	return nil
}

// Shutdown stops the webspace manager
func (m *Manager) Shutdown() {
	m.lxdListener.Disconnect()
	m.ports.Shutdown()
}

func (m *Manager) lxdInstanceName(user string) string {
	return user + m.config.Webspaces.InstanceSuffix
}

type lxdEventDetails struct {
	Action string
	Source string
}

func (m *Manager) onLxdEvent(e lxdApi.Event) {
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
	user := match[1]

	if err := m.traefik.ClearConfig(m.lxdInstanceName(user)); err != nil {
		log.WithField("user", user).WithError(err).Error("Failed to clear Traefik config")
		return
	}

	all, err := m.GetAll()
	if err != nil {
		log.WithError(err).Error("Failed to get all webspaces")
		return
	}
	if err := m.ports.Trim(all); err != nil {
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

	w, err := m.Get(user)
	if err != nil {
		log.WithField("user", user).WithError(err).Error("Failed to retrieve webspace")
		return
	}

	var running bool
	switch action {
	case "started":
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
			"user":   user,
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
		"user":    user,
		"running": running,
	}).Debug("Updating Traefik / port forward config")

	if err := m.traefik.GenerateConfig(w, addr); err != nil {
		log.WithField("user", user).WithError(err).Error("Failed to update Traefik config")
		return
	}

	if err := m.ports.AddAll(w, addr); err != nil {
		log.WithField("user", user).WithError(err).Error("Failed to update port forwards")
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
		return nil, ErrBadValue
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
func (m *Manager) Get(user string) (*Webspace, error) {
	w := &Webspace{
		manager: m,
		User:    user,
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
func (m *Manager) Create(user string, image string, password string, sshKey string) (*Webspace, error) {
	w := &Webspace{
		manager: m,
		User:    user,
		Config:  m.config.Webspaces.ConfigDefaults,
		Domains: []string{fmt.Sprintf("%v.%v", user, m.config.Webspaces.Domain)},
		Ports:   map[uint16]uint16{},
	}
	n := w.InstanceName()

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
			Profiles:  []string{m.config.Webspaces.Profile},
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
				"user":   user,
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
						"user":   user,
						"stdout": stdout,
						"stderr": stderr,
					}).WithError(err).Error("Failed to install sshd")
					return nil, fmt.Errorf("failed to install sshd: %w", err)
				}

				cmd := fmt.Sprintf(`mkdir -p /root/.ssh && echo "%v" > /root/.ssh/authorized_keys`, sshKey)
				if stdout, stderr, err := w.simpleExec(cmd); err != nil {
					log.WithFields(log.Fields{
						"user":   user,
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
