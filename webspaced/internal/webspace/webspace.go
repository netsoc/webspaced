package webspace

import (
	"bytes"
	"encoding/json"
	"fmt"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/netsoc/webspace-ng/webspaced/internal/config"
	log "github.com/sirupsen/logrus"
)

// Webspace represents a webspace with all of its configuration and state
type Webspace struct {
	manager *Manager

	User    string                `json:"string"`
	Config  config.WebspaceConfig `json:"config"`
	Domains []string              `json:"domains"`
	Ports   map[uint16]uint16     `json:"ports"`
}

// InstanceName uses the template to calculate the name of the instance
func (w *Webspace) InstanceName() (string, error) {
	buf := bytes.NewBufferString("")
	if err := w.manager.config.Webspaces.NameTemplate.Execute(buf, w); err != nil {
		return "", fmt.Errorf("Failed to execute container name template: %w", err)
	}

	return buf.String(), nil
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

// Create creates a new webspace container via LXD
func (m *Manager) Create(user string, image string, password string, sshKey string) (*Webspace, error) {
	w := &Webspace{
		manager: m,
		User:    user,
	}
	n, err := w.InstanceName()
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
			Profiles:  []string{m.config.LXD.Profile},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create webspace instance: %w", err)
	}

	if err := op.Wait(); err != nil {
		return nil, fmt.Errorf("Failed to wait for webspace instance creation: %w", err)
	}

	if password != "" {
		if err := m.lxdState(n, "start"); err != nil {
			return nil, fmt.Errorf("Failed to start webspace: %w", err)
		}

		op, err := m.lxd.ExecInstance(n, lxdApi.InstanceExecPost{
			Command: []string{"sh", "-c", fmt.Sprintf(`echo "root:%v" | chpasswd`, password)},
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("Failed to set root password: %w", err)
		}
		if err := op.Wait(); err != nil {
			return nil, fmt.Errorf("Failed to wait for password setting to complete: %v", err)
		}

		code := op.Get().Metadata["return"]
		if code != 0. {
			return nil, fmt.Errorf("Failed to change root password: exit code %v", code)
		}

		if err := m.lxdState(n, "stop"); err != nil {
			return nil, fmt.Errorf("Failed to stop webspace: %w", err)
		}
	}

	// TODO: Add SSH Key
	return w, nil
}
