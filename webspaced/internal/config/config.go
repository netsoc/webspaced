package config

import (
	"html/template"
	"reflect"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

// StringToLogLevelHookFunc returns a mapstructure.DecodeHookFunc which parses a logrus Level from a string
func StringToLogLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.TypeOf(log.InfoLevel).Kind() {
			return data, nil
		}

		var level log.Level
		err := level.UnmarshalText([]byte(data.(string)))
		return level, err
	}
}

// StringToTemplateHookFunc returns a mapstructure.DecodeHookFunc which parses a template.Template from a string
func StringToTemplateHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.TypeOf(template.Template{}).Kind() {
			return data, nil
		}

		return template.New("anonymous").Parse(data.(string))
	}
}

// DecoderOptions enables necessary mapstructure decode hook functions
func DecoderOptions(config *mapstructure.DecoderConfig) {
	config.ErrorUnused = true
	config.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		config.DecodeHook,
		StringToLogLevelHookFunc(),
		StringToTemplateHookFunc(),
	)
}

// WebspaceConfig describes a webspace's basic key = value configuration
type WebspaceConfig struct {
	StartupDelay float64 `json:"startupDelay" mapstructure:"startup_delay"`
	HTTPPort     uint16  `json:"httpPort" mapstructure:"http_port"`
	HTTPSPort    uint16  `json:"httpsPort" mapstructure:"https_port"`
}

// Config describes the configuration for Server
type Config struct {
	LogLevel        log.Level `mapstructure:"log_level"`
	BindSocket      string    `mapstructure:"bind_socket"`
	PwGrProxySocket string    `mapstructure:"pw_gr_proxy_socket"`
	LXD             struct {
		Socket  string
		Network string
	}
	Webspaces struct {
		AdminGroup      string `mapstructure:"admin_group"`
		Profile         string
		InstanceSuffix  string         `mapstructure:"instance_suffix"`
		Domain          string         `mapstructure:"domain"`
		ConfigDefaults  WebspaceConfig `mapstructure:"config_defaults"`
		MaxStartupDelay uint16         `mapstructure:"max_startup_delay"`
		RunLimit        uint           `mapstructure:"run_limit"`
		Ports           struct {
			Start uint16
			End   uint16
			Max   uint16
		}
	}
	Traefik struct {
		Redis struct {
			Addr string
			DB   int
		}
		HTTPEntryPoint  string `mapstructure:"http_entry_point"`
		HTTPSEntryPoint string `mapstructure:"https_entry_point"`
		CertResolver    string `mapstructure:"cert_resolver"`
		SANs            []string
		WebspacedSocket string `mapstructure:"webspaced_socket"`
	}
}
