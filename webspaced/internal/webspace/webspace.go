package webspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"regexp"

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

// ErrUsed indicates that the requested resource is already in use by a webspace
var ErrUsed = errors.New("used by a webspace")

// ErrNotRunning indicates that a webspace is not running
var ErrNotRunning = errors.New("not running")

// ErrRunning indicates that a webspace is already running
var ErrRunning = errors.New("already running")

// ErrDomainUnverified indicates that the request domain could not be verified
var ErrDomainUnverified = errors.New("verification failed")

// ErrDefaultDomain indicates an attempt to remove the default domain
var ErrDefaultDomain = errors.New("cannot remove the default domain")

// ErrTooManyPorts indicates that too many port forwards are configured
var ErrTooManyPorts = errors.New("port forward limit reached")

// ErrBadPort indicates that the provided port is invalid
var ErrBadPort = errors.New("invalid port")

// ErrInterface indicates the default interface is missing
var ErrInterface = errors.New("default network interface not present")

// ErrAddress indicates the interface didn't have an IPv4 address
var ErrAddress = errors.New("ipv4 address not found")

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

var lxdEventUserRegexTpl = `^/1\.0/\S+/(\S+)%v$`
var lxdEventStateRegex = regexp.MustCompile(`^\S+-(\S+)$`)

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

func (w *Webspace) simpleExec(cmd string) error {
	n := w.InstanceName()

	op, err := w.manager.lxd.ExecInstance(n, lxdApi.InstanceExecPost{
		Command: []string{"sh", "-c", cmd},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to execute command in LXD instance: %w", convertLXDError(err))
	}
	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to execute command in LXD instance: %w", convertLXDError(err))
	}

	code := op.Get().Metadata["return"]
	if code != 0. {
		return fmt.Errorf("failed to execute command in LXD instance: non-zero exit code %v", code)
	}
	return nil
}

// InstanceName uses the suffix to calculate the name of the instance
func (w *Webspace) InstanceName() string {
	return w.User + w.manager.config.Webspaces.InstanceSuffix
}

// Delete deletes the webspace
func (w *Webspace) Delete() error {
	n := w.InstanceName()

	state, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return fmt.Errorf("failed to get LXD instance state: %w", err)
	}

	if state.StatusCode == lxdApi.Running {
		if err := w.Shutdown(); err != nil {
			return err
		}
	}

	op, err := w.manager.lxd.DeleteInstance(n)
	if err != nil {
		return fmt.Errorf("failed to delete LXD instance: %w", err)
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to delete LXD instance: %w", err)
	}

	return nil
}

// Boot starts the webspace
func (w *Webspace) Boot() error {
	if err := w.manager.lxdState(w.InstanceName(), "start"); err != nil {
		return err
	}
	return nil
}

// Reboot restarts the webspace
func (w *Webspace) Reboot() error {
	if err := w.manager.lxdState(w.InstanceName(), "restart"); err != nil {
		return err
	}
	return nil
}

// Shutdown stops the webspace
func (w *Webspace) Shutdown() error {
	if err := w.manager.lxdState(w.InstanceName(), "stop"); err != nil {
		return err
	}
	return nil
}

// Save updates the stored LXD configuration
func (w *Webspace) Save() error {
	n := w.InstanceName()

	i, _, err := w.manager.lxd.GetInstance(n)
	if err != nil {
		return fmt.Errorf("failed to get instance from LXD: %w", convertLXDError(err))
	}

	lxdConf, err := w.lxdConfig()
	if err != nil {
		return err
	}

	i.InstancePut.Config[lxdConfigKey] = lxdConf
	op, err := w.manager.lxd.UpdateInstance(n, i.InstancePut, "")
	if err != nil {
		return fmt.Errorf("failed to update LXD instance: %w", convertLXDError(err))
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to update LXD instance: %w", convertLXDError(err))
	}
	return nil
}

// AddDomain verifies and adds a new domain
func (w *Webspace) AddDomain(domain string) error {
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
		return ErrDomainUnverified
	}

	webspaces, err := w.manager.GetAll()
	if err != nil {
		return err
	}

	for _, w := range webspaces {
		for _, d := range w.Domains {
			if d == domain {
				return ErrUsed
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
	if domain == fmt.Sprintf("%v.%v", w.User, w.manager.config.Webspaces.Domain) {
		return ErrDefaultDomain
	}

	for i, d := range w.Domains {
		if d == domain {
			e := len(w.Domains) - 1
			w.Domains[e], w.Domains[i] = w.Domains[i], w.Domains[e]
			w.Domains = w.Domains[:e]

			return w.Save()
		}
	}

	return ErrNotFound
}

// AddPort creates a port forwarding
func (w *Webspace) AddPort(external uint16, internal uint16) (uint16, error) {
	if len(w.Ports) == int(w.manager.config.Webspaces.Ports.Max) {
		return 0, ErrTooManyPorts
	}
	if internal == 0 {
		return 0, fmt.Errorf("%w (internal port cannot be 0)", ErrBadPort)
	}
	if external != 0 &&
		(external < w.manager.config.Webspaces.Ports.Start || external > w.manager.config.Webspaces.Ports.End) {
		return 0, fmt.Errorf("%w (external port out of range %v-%v)", ErrBadPort,
			w.manager.config.Webspaces.Ports.Start, w.manager.config.Webspaces.Ports.End)
	}

	webspaces, err := w.manager.GetAll()
	if err != nil {
		return 0, err
	}

	var allPorts []uint16
	for _, w := range webspaces {
		for e := range w.Ports {
			if e == external {
				return 0, ErrUsed
			}

			if external == 0 {
				allPorts = append(allPorts, e)
			}
		}
	}

	if external == 0 {
		start := w.manager.config.Webspaces.Ports.Start
		end := (w.manager.config.Webspaces.Ports.End - uint16(len(allPorts)) + 1)
		external = start + uint16(rand.Int31n(int32(end-start)))

		for _, p := range allPorts {
			if external < p {
				break
			}
			external++
		}
	}

	w.Ports[external] = internal
	if err := w.Save(); err != nil {
		return 0, err
	}
	return external, nil
}

// RemovePort removes a port forwarding
func (w *Webspace) RemovePort(external uint16) error {
	if _, ok := w.Ports[external]; !ok {
		return ErrNotFound
	}

	delete(w.Ports, external)
	return w.Save()
}

// GetIP retrieves the webspace's primary IP address
func (w *Webspace) GetIP() (string, error) {
	n := w.InstanceName()

	state, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return "", fmt.Errorf("failed to get LXD instance state: %w", err)
	}

	iface, ok := state.Network["eth0"]
	if !ok {
		return "", ErrInterface
	}

	var addr string
	for _, info := range iface.Addresses {
		if info.Family != "inet" || info.Scope != "global" {
			continue
		}

		addr = info.Address
	}
	if addr == "" {
		return "", ErrAddress
	}

	return addr, nil
}

// Manager manages webspace containers
type Manager struct {
	config         *config.Config
	lxd            lxd.InstanceServer
	lxdWsUserRegex *regexp.Regexp
	traefik        *Traefik
}

// NewManager returns a new WebspaceManager instance
func NewManager(cfg *config.Config, l lxd.InstanceServer) (*Manager, error) {
	t, err := NewTraefik(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Traefik manager: %w", err)
	}

	m := &Manager{
		cfg,
		l,
		regexp.MustCompile(fmt.Sprintf(lxdEventUserRegexTpl, cfg.Webspaces.InstanceSuffix)),
		t,
	}

	listener, err := l.GetEvents()
	if err != nil {
		return nil, err
	}
	listener.AddHandler([]string{"lifecycle"}, m.onLxdEvent)

	return m, nil
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
	w, err := m.Get(user)
	if err != nil {
		log.WithField("err", err).Error("Failed to retrieve webspace")
		return
	}

	state, _, err := m.lxd.GetInstanceState(w.InstanceName())
	if err != nil {
		log.WithField("err", err).Error("Failed to retrieve LXD instance state")
		return
	}

	log.WithFields(log.Fields{
		"user":  user,
		"state": state.Status,
	}).Debug("Updating Traefik config")
	if err := m.traefik.UpdateConfig(w, state.StatusCode == lxdApi.Running); err != nil {
		log.WithFields(log.Fields{
			"user": user,
			"err":  err,
		}).Error("Failed to update Traefik config")
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
