package config

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

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
		mapstructure.StringToTimeDurationHookFunc(),
		StringToLogLevelHookFunc(),
		StringToTemplateHookFunc(),
	)
}

// WebspaceConfig describes a webspace's basic key = value configuration
type WebspaceConfig struct {
	StartupDelay   float64 `json:"startupDelay" mapstructure:"startup_delay"`
	HTTPPort       uint16  `json:"httpPort" mapstructure:"http_port"`
	SNIPassthrough bool    `json:"sniPassthrough" mapstructure:"sni_passthrough"`
}

// Config describes the configuration for Server
type Config struct {
	LogLevel log.Level `mapstructure:"log_level"`

	IAM struct {
		URL           string
		Token         string
		AllowInsecure bool `mapstructure:"allow_insecure"`
	}

	LXD struct {
		URL string
		TLS struct {
			// Set CA if using PKI mode
			CA     string
			CAFile string `mapstructure:"ca_file"`

			// Set server certificate if not using PKI
			ServerCert     string `mapstructure:"server_cert"`
			ServerCertFile string `mapstructure:"server_cert_file"`

			ClientCert     string `mapstructure:"client_cert"`
			ClientCertFile string `mapstructure:"client_cert_file"`

			ClientKey     string `mapstructure:"client_key"`
			ClientKeyFile string `mapstructure:"client_key_file"`

			TrustPassword     string `mapstructure:"trust_password"`
			TrustPasswordFile string `mapstructure:"trust_password_file"`

			AllowInsecure bool `mapstructure:"allow_insecure"`
		}
	}

	Webspaces struct {
		LXDProfile      string         `mapstructure:"lxd_profile"`
		InstancePrefix  string         `mapstructure:"instance_prefix"`
		Domain          string         `mapstructure:"domain"`
		ConfigDefaults  WebspaceConfig `mapstructure:"config_defaults"`
		MaxStartupDelay uint16         `mapstructure:"max_startup_delay"`
		IPTimeout       time.Duration  `mapstructure:"ip_timeout"`
		RunLimit        uint           `mapstructure:"run_limit"`

		Ports struct {
			Start uint16
			End   uint16
			Max   uint16

			KubernetesService string `mapstructure:"kubernetes_service"`
		}
	}

	HTTP struct {
		ListenAddress string `mapstructure:"listen_address"`

		CORS struct {
			AllowedOrigins []string `mapstructure:"allowed_origins"`
		}
	}

	Traefik struct {
		Provider string

		Redis struct {
			Addr         string
			DB           int
			CertResolver string `mapstructure:"cert_resolver"`
		}
		Kubernetes struct {
			Namespace     string
			DefaultSecret string `mapstructure:"default_secret"`
			ClusterIssuer string `mapstructure:"cluster_issuer"`
		}

		HTTPSEntryPoint string   `mapstructure:"https_entrypoint"`
		DefaultSANs     []string `mapstructure:"default_sans"`

		WebspacedURL string `mapstructure:"webspaced_url"`
		IAMToken     string `mapstructure:"iam_token"`
	}
}

func loadSecret(parent interface{}, field string) error {
	v := reflect.ValueOf(parent).Elem()
	t := v.Type()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("%v is not a struct", t.Name())
	}

	if _, ok := t.FieldByName(field); !ok {
		return fmt.Errorf("%v field %v not found", t.Name(), field)
	}
	f := v.FieldByName(field)

	if _, ok := t.FieldByName(field + "File"); !ok {
		return fmt.Errorf("%v file field %v not found", t.Name(), field)
	}
	fileField := v.FieldByName(field + "File")

	if fileField.Kind() != reflect.String {
		return fmt.Errorf("%v file field %v is not a string", t.Name(), fileField)
	}

	file := fileField.String()
	if file == "" {
		return nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read %v for field %v", file, field)
	}

	if f.Kind() == reflect.String {
		f.SetString(strings.TrimSpace(string(data)))
	} else if t == reflect.SliceOf(reflect.TypeOf(byte(0))) {
		f.SetBytes(data)
	} else {
		return fmt.Errorf("invalid type %v for field %v", t, field)
	}

	return nil
}

// ReadSecrets loads values for secret config options from files
func (c *Config) ReadSecrets() error {
	tls := []string{"CA", "ServerCert", "ClientCert", "ClientKey", "TrustPassword"}
	for _, f := range tls {
		if err := loadSecret(&c.LXD.TLS, f); err != nil {
			return err
		}
	}

	return nil
}
