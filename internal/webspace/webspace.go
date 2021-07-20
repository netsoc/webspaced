package webspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"regexp"
	"time"

	"github.com/cenkalti/backoff/v4"
	lxdApi "github.com/lxc/lxd/shared/api"
	iam "github.com/netsoc/iam/client"
	"github.com/netsoc/webspaced/internal/config"
	"github.com/netsoc/webspaced/pkg/util"
	log "github.com/sirupsen/logrus"
)

const lxdConfigKey = "user._webspaced"

// convertLXDError is a HACK: LXD doesn't seem to return a code we can use to determine the error...
func convertLXDError(err error) error {
	switch err.Error() {
	case "not found", "No such object":
		return util.ErrGenericNotFound
	case "Create instance: Add instance info to the database: This instance already exists":
		return util.ErrExists
	case "The container is already stopped":
		return util.ErrNotRunning
	case "Common start logic: The container is already running":
		return util.ErrRunning

	default:
		return err
	}
}

var lxdEventUserRegexTpl = `^/1\.0/\S+/%vu(\d+)$`
var lxdEventActionRegex = regexp.MustCompile(`^\S+-(\S+)$`)
var lxdLogFilenameRegex = regexp.MustCompile(`/1.0/instances/\S+/logs/(\S+)`)

// Webspace represents a webspace with all of its configuration and state
type Webspace struct {
	manager *Manager
	user    *iam.User

	UserID  int                   `json:"user"`
	Config  config.WebspaceConfig `json:"config"`
	Domains []string              `json:"domains"`
	Ports   map[uint16]uint16     `json:"ports"`
}

// GetUser gets the IAM user associated with this webspace
func (w *Webspace) GetUser(ctx context.Context) (*iam.User, error) {
	if w.user != nil {
		return w.user, nil
	}

	ctx = context.WithValue(ctx, iam.ContextAccessToken, w.manager.config.IAM.Token)
	user, _, err := w.manager.iam.UsersApi.GetUserByID(ctx, int32(w.UserID))
	if err != nil {
		return nil, err
	}

	w.user = &user
	return w.user, err
}

func (w *Webspace) lxdConfig() (string, error) {
	if w.Config.StartupDelay < 0 {
		return "", util.ErrBadValue
	}

	confJSON, err := json.Marshal(w)
	if err != nil {
		return "", fmt.Errorf("failed to serialize webspace config for LXD: %w", err)
	}

	return string(confJSON), nil
}

// Exec runs a command in a webspace non-interactively
func (w *Webspace) Exec(cmd string, ensureBooted bool) (int, string, string, error) {
	n := w.InstanceName()

	if ensureBooted {
		state, _, err := w.manager.lxd.GetInstanceState(n)
		if err != nil {
			return -1, "", "", fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
		}

		if state.StatusCode != lxdApi.Running {
			if err := w.Boot(); err != nil {
				return -1, "", "", fmt.Errorf("failed to start webspace: %w", err)
			}
		}
	}

	asyncOp, err := w.manager.lxd.ExecInstance(n, lxdApi.InstanceExecPost{
		Command:      []string{"sh", "-c", cmd},
		RecordOutput: true,
	}, nil)
	if err != nil {
		return -1, "", "", fmt.Errorf("failed to execute command in LXD instance: %w", convertLXDError(err))
	}
	if err := asyncOp.Wait(); err != nil {
		return -1, "", "", fmt.Errorf("failed to execute command in LXD instance: %w", convertLXDError(err))
	}

	op := asyncOp.Get()
	output := op.Metadata["output"].(map[string]interface{})
	outReader, err := w.manager.lxd.GetInstanceLogfile(n, lxdLogFilenameRegex.FindStringSubmatch(output["1"].(string))[1])
	if err != nil {
		return -1, "", "", fmt.Errorf("failed to retrieve LXD command stdout: %w", convertLXDError(err))
	}
	errReader, err := w.manager.lxd.GetInstanceLogfile(n, lxdLogFilenameRegex.FindStringSubmatch(output["2"].(string))[1])
	if err != nil {
		return -1, "", "", fmt.Errorf("failed to retrieve LXD command stderr: %w", convertLXDError(err))
	}

	outData, err := ioutil.ReadAll(outReader)
	if err != nil {
		return -1, "", "", fmt.Errorf("failed to read LXD command stdout: %w", err)
	}
	outReader.Close()

	errData, err := ioutil.ReadAll(errReader)
	if err != nil {
		return -1, "", "", fmt.Errorf("failed to read LXD command stderr: %w", err)
	}
	errReader.Close()

	stdout := string(outData)
	stderr := string(errData)

	code, ok := op.Metadata["return"].(float64)
	if !ok {
		return -1, stdout, stderr, errors.New("failed to get exit code")
	}
	return int(code), stdout, stderr, nil
}

func (w *Webspace) simpleExec(cmd string) (string, string, error) {
	c, so, se, err := w.Exec(cmd, false)
	if err != nil {
		return so, se, err
	}

	if c != 0 {
		return so, se, fmt.Errorf("exit with non-zero exit status: %v", c)
	}

	return so, se, nil
}

// InstanceName uses the suffix to calculate the name of the instance
func (w *Webspace) InstanceName() string {
	return w.manager.lxdInstanceName(w.UserID)
}

// Delete deletes the webspace
func (w *Webspace) Delete() error {
	w.manager.Lock(w.UserID)
	defer w.manager.Unlock(w.UserID)
	n := w.InstanceName()

	state, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
	}

	if state.StatusCode == lxdApi.Running {
		if err := w.Shutdown(); err != nil {
			return err
		}
	}

	op, err := w.manager.lxd.DeleteInstance(n)
	if err != nil {
		return fmt.Errorf("failed to delete LXD instance: %w", convertLXDError(err))
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to delete LXD instance: %w", convertLXDError(err))
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

// DefaultDomain returns the default domain for the webspace
func (w *Webspace) DefaultDomain(ctx context.Context) (string, error) {
	user, err := w.GetUser(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	return user.Username + "." + w.manager.config.Webspaces.Domain, nil
}

// GetDomains gets all domains (including the default one, which can change because of usernames!)
func (w *Webspace) GetDomains(ctx context.Context) ([]string, error) {
	domains := make([]string, len(w.Domains))
	for i, d := range w.Domains {
		domains[i] = d
	}

	defaultDomain, err := w.DefaultDomain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default domain: %w", err)
	}

	domains = append([]string{defaultDomain}, domains...)
	return domains, nil
}

// AddDomain verifies and adds a new domain
func (w *Webspace) AddDomain(domain string) error {
	records, err := net.LookupTXT(domain)
	if err != nil {
		return fmt.Errorf("failed to lookup TXT records: %w", err)
	}

	correct := fmt.Sprintf("webspace:id:%v", w.UserID)
	verified := false
	for _, r := range records {
		if r == correct {
			verified = true
		}
	}
	if !verified {
		return util.ErrDomainUnverified
	}

	webspaces, err := w.manager.GetAll()
	if err != nil {
		return err
	}

	for _, w := range webspaces {
		for _, d := range w.Domains {
			if d == domain {
				return util.ErrUsed
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
func (w *Webspace) RemoveDomain(ctx context.Context, domain string) error {
	u, err := w.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if domain == fmt.Sprintf("%v.%v", u.Username, w.manager.config.Webspaces.Domain) {
		return util.ErrDefaultDomain
	}

	for i, d := range w.Domains {
		if d == domain {
			e := len(w.Domains) - 1
			w.Domains[e], w.Domains[i] = w.Domains[i], w.Domains[e]
			w.Domains = w.Domains[:e]

			return w.Save()
		}
	}

	return util.ErrGenericNotFound
}

// AddPort creates a port forwarding
func (w *Webspace) AddPort(external uint16, internal uint16) (uint16, error) {
	if len(w.Ports) == int(w.manager.config.Webspaces.Ports.Max) {
		return 0, util.ErrTooManyPorts
	}
	if internal == 0 {
		return 0, fmt.Errorf("%w (internal port cannot be 0)", util.ErrBadPort)
	}
	if external != 0 &&
		(external < w.manager.config.Webspaces.Ports.Start || external > w.manager.config.Webspaces.Ports.End) {
		return 0, fmt.Errorf("%w (external port out of range %v-%v)", util.ErrBadPort,
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
				return 0, util.ErrUsed
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
		return util.ErrGenericNotFound
	}

	delete(w.Ports, external)
	return w.Save()
}

// GetIP retrieves the webspace's primary IP address
func (w *Webspace) GetIP(state *lxdApi.InstanceState) (string, error) {
	n := w.InstanceName()

	if state == nil {
		var err error
		state, _, err = w.manager.lxd.GetInstanceState(n)
		if err != nil {
			return "", fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
		}
	}

	iface, ok := state.Network["eth0"]
	if !ok {
		return "", util.ErrInterface
	}

	var addr string
	for _, info := range iface.Addresses {
		if info.Family != "inet" || info.Scope != "global" {
			continue
		}

		addr = info.Address
	}
	if addr == "" {
		return "", util.ErrAddress
	}

	return addr, nil
}

// AwaitIP attempts to retrieve the webspace's IP with exponential backoff
func (w *Webspace) AwaitIP() (string, error) {
	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = w.manager.config.Webspaces.IPTimeout

	var addr string
	if err := backoff.RetryNotify(func() error {
		var err error
		addr, err = w.GetIP(nil)
		return err
	}, back, func(_ error, retry time.Duration) {
		log.WithFields(log.Fields{
			"uid":   w.UserID,
			"retry": retry,
		}).Debug("Failed to get webspace IP")
	}); err != nil {
		return "", err
	}

	return addr, nil
}

// EnsureStarted starts a webspace if it isn't running (delaying by the startup delay) and returns its IP address after
func (w *Webspace) EnsureStarted() (string, error) {
	state, _, err := w.manager.lxd.GetInstanceState(w.InstanceName())
	if err != nil {
		return "", fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
	}

	if state.StatusCode == lxdApi.Running {
		ip, err := w.AwaitIP()
		if err != nil {
			return "", fmt.Errorf("failed to get webspace IP: %w", err)
		}

		return ip, nil
	}

	if err := w.Boot(); err != nil {
		return "", fmt.Errorf("failed to start webspace: %w", err)
	}

	ip, err := w.AwaitIP()
	if err != nil {
		return "", fmt.Errorf("failed to get webspace IP: %w", err)
	}

	time.Sleep(time.Duration(w.Config.StartupDelay * float64(time.Second)))
	return ip, nil
}

// InterfaceAddress describes a network interface's address
type InterfaceAddress struct {
	Family  string `json:"family"`
	Address string `json:"address"`
	Netmask string `json:"netmask"`
	Scope   string `json:"scope"`
}

// InterfaceCounters describes a network interface's statistics
type InterfaceCounters struct {
	BytesReceived int64 `json:"bytesReceived"`
	BytesSent     int64 `json:"bytesSent"`

	PacketsReceived int64 `json:"packetsReceived"`
	PacketsSent     int64 `json:"packetsSent"`
}

// NetworkInterface describe's a webspace's network interface
type NetworkInterface struct {
	MAC   string `json:"mac"`
	MTU   int    `json:"mtu"`
	State string `json:"state"`

	Counters  InterfaceCounters  `json:"counters"`
	Addresses []InterfaceAddress `json:"addresses"`
}

// Usage describes a webspace's resource usage
type Usage struct {
	CPU       int64            `json:"cpu"`
	Disks     map[string]int64 `json:"disks"`
	Memory    int64            `json:"memory"`
	Processes int64            `json:"processes"`
}

// State describes a webspace's state
type State struct {
	Running           bool                        `json:"running"`
	Uptime            float64                     `json:"uptime"`
	Usage             Usage                       `json:"usage"`
	NetworkInterfaces map[string]NetworkInterface `json:"networkInterfaces"`
}

// State returns information about the webspace's state
func (w *Webspace) State() (State, error) {
	n := w.InstanceName()

	ls, _, err := w.manager.lxd.GetInstanceState(n)
	if err != nil {
		return State{}, fmt.Errorf("failed to get LXD instance state: %w", convertLXDError(err))
	}

	s := State{
		Running: ls.StatusCode == lxdApi.Running,
		Usage: Usage{
			CPU:       ls.CPU.Usage,
			Disks:     map[string]int64{},
			Memory:    ls.Memory.Usage,
			Processes: ls.Processes,
		},
		NetworkInterfaces: map[string]NetworkInterface{},
	}
	if s.Running {
		i, _, err := w.manager.lxd.GetInstance(n)
		if err != nil {
			return State{}, fmt.Errorf("failed to get instance from LXD: %w", convertLXDError(err))
		}

		s.Uptime = time.Now().Sub(i.LastUsedAt).Seconds()
	}

	for name, info := range ls.Disk {
		if info.Usage == -1 {
			continue
		}

		s.Usage.Disks[name] = info.Usage
	}

	if ls.Network != nil {
		for name, info := range ls.Network {
			if name == "lo" {
				continue
			}

			i := NetworkInterface{
				MAC:   info.Hwaddr,
				MTU:   info.Mtu,
				State: info.State,

				Counters: InterfaceCounters{
					BytesReceived:   info.Counters.BytesReceived,
					BytesSent:       info.Counters.BytesSent,
					PacketsReceived: info.Counters.PacketsReceived,
					PacketsSent:     info.Counters.PacketsSent,
				},
				Addresses: []InterfaceAddress{},
			}
			for _, addr := range info.Addresses {
				i.Addresses = append(i.Addresses, InterfaceAddress{
					Family:  addr.Family,
					Address: addr.Address,
					Netmask: addr.Netmask,
					Scope:   addr.Scope,
				})
			}

			s.NetworkInterfaces[name] = i
		}
	}

	return s, nil
}

// Log returns the webspace's `/dev/console` log
func (w *Webspace) Log() (io.ReadCloser, error) {
	log, err := w.manager.lxd.GetInstanceConsoleLog(w.InstanceName(), nil)
	if err != nil {
		return nil, convertLXDError(err)
	}

	return log, nil
}

// ClearLog clears the webspace's `/dev/console` log
func (w *Webspace) ClearLog() error {
	if err := w.manager.lxd.DeleteInstanceConsoleLog(w.InstanceName(), nil); err != nil {
		return convertLXDError(err)
	}

	return nil
}

// Sync forces configuration for a webspace to be re-generated
func (w *Webspace) Sync(ctx context.Context) error {
	w.manager.Lock(w.UserID)
	defer w.manager.Unlock(w.UserID)
	addr, _ := w.GetIP(nil)

	if err := w.manager.traefik.ClearConfig(ctx, w.InstanceName()); err != nil {
		return fmt.Errorf("failed to clear traefik config: %w", err)
	}
	if err := w.manager.traefik.GenerateConfig(ctx, w, addr); err != nil {
		return fmt.Errorf("failed to update traefik config: %w", err)
	}

	if err := w.manager.ports.AddAll(ctx, w, addr); err != nil {
		return fmt.Errorf("failed to set up port forwards: %w", err)
	}

	return nil
}
