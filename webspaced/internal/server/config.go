package server

import (
	"html/template"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/netsoc/webspace-ng/webspaced/internal/webspace"
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

// ConfigDecoderOptions enables necessary mapstructure decode hook functions
func ConfigDecoderOptions(config *mapstructure.DecoderConfig) {
	config.ErrorUnused = true
	config.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		config.DecodeHook,
		StringToLogLevelHookFunc(),
		StringToTemplateHookFunc(),
	)
}

// Config describes the configuration for Server
type Config struct {
	LogLevel        log.Level `mapstructure:"log_level"`
	BindSocket      string    `mapstructure:"bind_socket"`
	PwGrProxySocket string    `mapstructure:"pw_gr_proxy_socket"`
	LXD             struct {
		Socket         string
		Profile        string
		InstanceSuffix string `mapstructure:"instance_suffix"`
		Network        string
	}
	Webspaces struct {
		AdminGroup      string            `mapstructure:"admin_group"`
		NameTemplate    template.Template `mapstructure:"name_template"`
		DomainSuffix    string            `mapstructure:"domain_suffix"`
		ConfigDefaults  webspace.Config   `mapstructure:"config_defaults"`
		MaxStartupDelay uint16            `mapstructure:"max_startup_delay"`
		RunLimit        uint              `mapstructure:"run_limit"`
		Ports           struct {
			Start uint16
			End   uint16
			Max   uint16
		}
	}
}
