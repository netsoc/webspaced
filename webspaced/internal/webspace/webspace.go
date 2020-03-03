package webspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/netsoc/webspace-ng/webspaced/internal/config"
	log "github.com/sirupsen/logrus"
)

const lxdConfigKey = "user._webspaced"

// ErrNotFound indicates that a resource was not found
var ErrNotFound = errors.New("not found")

// ErrExists indicates that a resource already exists
var ErrExists = errors.New("already exists")

// ErrNotRunning indicates that a webspace is not running
var ErrNotRunning = errors.New("not running")

// ErrRunning indicates that a webspace is already running
var ErrRunning = errors.New("already running")

// ErrDomainUnverified indicates that the request domain could not be verified
var ErrDomainUnverified = errors.New("verification failed")

// ErrDomainUsed indicates that the requested domain is already in use
var ErrDomainUsed = errors.New("used by a webspace")

// convertLXDError is a HACK: LXD doesn't seem to return a code we can use to determine the error...
func convertLXDError(err error) error {
	switch err.Error() {
	case "not found", "No such object":
		return ErrNotFound
	case "Create instance: Add instance info to the database: This instance already exists":
		return ErrExists
	case "The container is already stopped":
		return ErrNotRunning
	case "Common start logic: The container is already running":
		return ErrRunning

	default:
		return err
	}
}

// Webspace represents a webspace with all of its configuration and state
type Webspace struct {
	manager *Manager

	User    string                `json:"user"`
	Config  config.WebspaceConfig `json:"config"`
	Domains []string              `json:"domains"`
	Ports   map[uint16]uint16     `json:"ports"`
}

func (w *Webspace) lxdConfig() (string, error) {
	confJSON, err := json.Marshal(w)
	if err != nil {
		return "", fmt.Errorf("failed to serialize webspace config for LXD: %w", err)
	}

	return string(confJSON), nil
}

// InstanceName uses the template to calculate the name of the instance
func (w *Webspace) InstanceName() (string, error) {
	buf := bytes.NewBufferString("")
	if err := w.manager.config.Webspaces.NameTemplate.Execute(buf, w); err != nil {
		return "", fmt.Errorf("failed to execute container name template: %w", err)
	}

	return buf.String(), nil
}

// Delete deletes the webspace
func (w *Webspace) Delete() error {
	n, err := w.InstanceName()
	if err != nil {
		return err
	}

	state, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return fmt.Errorf("failed to get webspace state: %w", err)
	}

	if state.StatusCode == lxdApi.Running {
		if err := w.Shutdown(); err != nil {
			return err
		}
	}

	op, err := w.manager.lxd.DeleteInstance(n)
	if err != nil {
		return fmt.Errorf("failed to delete webspace instance: %w", err)
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to delete webspace instance: %w", err)
	}

	return nil
}

// Boot starts the webspace
func (w *Webspace) Boot() error {
	n, err := w.InstanceName()
	if err != nil {
		return err
	}

	if err := w.manager.lxdState(n, "start"); err != nil {
		return fmt.Errorf("failed to boot webspace: %w", convertLXDError(err))
	}
	return nil
}

// Reboot restarts the webspace
func (w *Webspace) Reboot() error {
	n, err := w.InstanceName()
	if err != nil {
		return err
	}

	if err := w.manager.lxdState(n, "restart"); err != nil {
		return fmt.Errorf("failed to reboot webspace: %w", convertLXDError(err))
	}
	return nil
}

// Shutdown stops the webspace
func (w *Webspace) Shutdown() error {
	n, err := w.InstanceName()
	if err != nil {
		return err
	}

	if err := w.manager.lxdState(n, "stop"); err != nil {
		return fmt.Errorf("failed to shutdown webspace: %w", convertLXDError(err))
	}
	return nil
}

// Save updates the stored LXD configuration
func (w *Webspace) Save() error {
	n, err := w.InstanceName()
	if err != nil {
		return err
	}

	i, _, err := w.manager.lxd.GetInstance(n)
	if err != nil {
		return fmt.Errorf("failed to get webspace config from LXD: %w", convertLXDError(err))
	}

	lxdConf, err := w.lxdConfig()
	if err != nil {
		return err
	}

	i.InstancePut.Config[lxdConfigKey] = lxdConf
	op, err := w.manager.lxd.UpdateInstance(n, i.InstancePut, "")
	if err != nil {
		return fmt.Errorf("failed to update LXD config: %w", convertLXDError(err))
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to update LXD config: %w", convertLXDError(err))
	}
	return nil
}

// AddDomain verifies and adds a new domain
func (w *Webspace) AddDomain(domain string) error {
	webspaces, err := w.manager.GetAll()
	if err != nil {
		return err
	}

	records, err := net.LookupTXT(domain)
	if err != nil {
		return fmt.Errorf("failed to lookup TXT records: %w", err)
	}

	correct := fmt.Sprintf("webspace:%v", w.User)
	verified := false
	for _, r := range records {
		if r == correct {
			verified = true
		}
	}
	if !verified {
		return fmt.Errorf("failed to add custom domain: %w", ErrDomainUnverified)
	}

	for _, w := range webspaces {
		for _, d := range w.Domains {
			if d == domain {
				return fmt.Errorf("failed to add custom domain: %w", ErrDomainUsed)
			}
		}
	}

	w.Domains = append(w.Domains, domain)
	if err := w.Save(); err != nil {
		return err
	}
	return nil
}

// RemoveDomain removes an existing domain
func (w *Webspace) RemoveDomain(domain string) error {
	for i, d := range w.Domains {
		if d == domain {
			e := len(w.Domains) - 1
			w.Domains[e], w.Domains[i] = w.Domains[i], w.Domains[e]
			w.Domains = w.Domains[:e]

			return w.Save()
		}
	}

	return fmt.Errorf("failed to remove domain: %w", ErrNotFound)
}

// Manager manages webspace containers
type Manager struct {
	config *config.Config
	lxd    lxd.InstanceServer
}

// NewManager returns a new WebspaceManager instance
func NewManager(cfg *config.Config, l lxd.InstanceServer) (*Manager, error) {
	m := &Manager{
		cfg,
		l,
	}

	listener, err := l.GetEvents()
	if err != nil {
		return nil, err
	}
	listener.AddHandler([]string{"lifecycle"}, m.onLxdEvent)

	return m, nil
}

func (m *Manager) onLxdEvent(e lxdApi.Event) {
	var details map[string]interface{}
	json.Unmarshal(e.Metadata, &details)
	log.WithFields(log.Fields{
		"type":    e.Type,
		"details": details,
	}).Debug("lxd event")
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

	return w, nil
}

func (m *Manager) lxdState(name string, action string) error {
	op, err := m.lxd.UpdateContainerState(name, lxdApi.ContainerStatePut{
		Action:  action,
		Timeout: -1,
	}, "")
	if err != nil {
		return nil
	}

	if err := op.Wait(); err != nil {
		return err
	}
	return nil
}

// Get retrieves a Webspace instance from LXD
func (m *Manager) Get(user string) (*Webspace, error) {
	w := &Webspace{
		manager: m,
		User:    user,
	}
	n, err := w.InstanceName()
	if err != nil {
		return nil, err
	}

	i, _, err := m.lxd.GetInstance(n)
	if err != nil {
		return nil, fmt.Errorf("failed to get webspace instance: %w", convertLXDError(err))
	}

	return m.instanceToWebspace(i)
}

// GetAll retrieves all the webspaces
func (m *Manager) GetAll() ([]*Webspace, error) {
	instances, err := m.lxd.GetInstances(lxdApi.InstanceTypeContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve webspaces from LXD: %w", convertLXDError(err))
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
		Domains: []string{},
		Ports:   map[uint16]uint16{},
	}
	n, err := w.InstanceName()
	if err != nil {
		return nil, err
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
			Profiles:  []string{m.config.Webspaces.Profile},
			Config: map[string]string{
				lxdConfigKey: lxdConf,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webspace instance: %w", convertLXDError(err))
	}

	if err := op.Wait(); err != nil {
		return nil, fmt.Errorf("failed to create webspace instance: %w", convertLXDError(err))
	}

	if password != "" {
		if err := w.Boot(); err != nil {
			return nil, err
		}

		op, err := m.lxd.ExecInstance(n, lxdApi.InstanceExecPost{
			Command: []string{"sh", "-c", fmt.Sprintf(`echo "root:%v" | chpasswd`, password)},
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to set root password: %w", convertLXDError(err))
		}
		if err := op.Wait(); err != nil {
			return nil, fmt.Errorf("failed to set root password: %w", convertLXDError(err))
		}

		code := op.Get().Metadata["return"]
		if code != 0. {
			return nil, fmt.Errorf("failed to change root password: exit code %v", code)
		}

		if err := w.Shutdown(); err != nil {
			return nil, err
		}
	}

	// TODO: Add SSH Key
	return w, nil
}
