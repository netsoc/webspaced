package webspace

import (
	lxd "github.com/lxc/lxd/client"
)

// Config describes a webspace's basic key = value configuration
type Config struct {
	StartupDelay float64 `json:"startupDelay" mapstructure:"startup_delay"`
	HTTPPort     uint16  `json:"httpPort" mapstructure:"http_port"`
	HTTPSPort    uint16  `json:"httpsPort" mapstructure:"https_port"`
}

// Webspace represents a webspace with all of its configuration and state
type Webspace struct {
	User    string            `json:"string"`
	Config  Config            `json:"config"`
	Domains []string          `json:"domains"`
	Ports   map[uint16]uint16 `json:"ports"`
}

// CreateWebspace creates a new webspace container via LXD
func CreateWebspace(lxd *lxd.ProtocolLXD, image string, password string, sshKey string) error {
	//lxd.CreateInstance(lxdApi.InstancePost{})
	return nil
}
