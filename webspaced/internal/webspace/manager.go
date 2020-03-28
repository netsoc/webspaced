package webspace

import (
	"encoding/json"
	"fmt"
	"regexp"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/netsoc/webspace-ng/webspaced/internal/config"
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
			return fmt.Errorf("failed to retrieve LXD instance state: %w", err)
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
		return fmt.Errorf("failed to get LXD event listener: %w", err)
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
		log.WithFields(log.Fields{
			"user": user,
			"err":  err,
		}).Error("Failed to clear Traefik config")
		return
	}

	all, err := m.GetAll()
	if err != nil {
		log.WithField("err", err).Error("Failed to get all webspaces")
		return
	}
	if err := m.ports.Trim(all); err != nil {
		log.WithField("err", err).Error("Failed to trim port forwards")
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
		log.WithFields(log.Fields{
			"user": user,
			"err":  err,
		}).Error("Failed to retrieve webspace")
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
			log.WithField("err", err).Error("Failed to retrieve LXD instance state")
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
			log.WithField("err", err).Error("Failed to get instance IP address")
			return
		}
	}

	log.WithFields(log.Fields{
		"user":    user,
		"running": running,
	}).Debug("Updating Traefik / port forward config")

	if err := m.traefik.GenerateConfig(w, addr); err != nil {
		log.WithFields(log.Fields{
			"user": user,
			"err":  err,
		}).Error("Failed to update Traefik config")
		return
	}

	if err := m.ports.AddAll(w, addr); err != nil {
		log.WithFields(log.Fields{
			"user": user,
			"err":  err,
		}).Error("Failed to update port forwards")
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
		if err := w.Boot(); err != nil {
			return nil, err
		}
	}
	if password != "" {
		if err := w.simpleExec(fmt.Sprintf(`echo "root:%v" | chpasswd`, password)); err != nil {
			return nil, fmt.Errorf("failed to set root password: %w", err)
		}
	}
	if sshKey != "" {
		// TODO: install sshd
		cmd := fmt.Sprintf(`mkdir -p /root/.ssh && echo "%v" > /root/.ssh/authorized_keys`, sshKey)
		if err := w.simpleExec(cmd); err != nil {
			return nil, fmt.Errorf("failed to store ssh public key: %w", err)
		}

		if _, err := w.AddPort(0, 22); err != nil {
			return nil, fmt.Errorf("failed to add SSH port forward: %w", err)
		}
	}
	if password != "" || sshKey != "" {
		if err := w.Shutdown(); err != nil {
			return nil, err
		}
	}

	return w, nil
}
