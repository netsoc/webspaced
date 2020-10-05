package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/netsoc/webspaced/internal/config"
	"github.com/netsoc/webspaced/internal/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var srv *server.Server

func init() {
	// Config defaults
	viper.SetDefault("log_level", log.InfoLevel)

	viper.SetDefault("iam.url", "https://iam.netsoc.ie/v1")
	viper.SetDefault("iam.token", "")
	viper.SetDefault("iam.allow_insecure", false)

	viper.SetDefault("lxd.url", "https://localhost")
	viper.SetDefault("lxd.tls.ca", "")
	viper.SetDefault("lxd.tls.ca_file", "")
	viper.SetDefault("lxd.tls.server_cert", "")
	viper.SetDefault("lxd.tls.server_cert_file", "")
	viper.SetDefault("lxd.tls.client_cert", "")
	viper.SetDefault("lxd.tls.client_cert_file", "")
	viper.SetDefault("lxd.tls.client_key", "")
	viper.SetDefault("lxd.tls.client_key_file", "")
	viper.SetDefault("lxd.tls.trust_password", "")
	viper.SetDefault("lxd.tls.trust_password_file", "")
	viper.SetDefault("lxd.tls.allow_insecure", false)

	viper.SetDefault("webspaces.lxd_profile", "webspace")
	viper.SetDefault("webspaces.instance_prefix", "ws-")
	viper.SetDefault("webspaces.domain", "ng.localhost")
	viper.SetDefault("webspaces.config_defaults.startup_delay", 3)
	viper.SetDefault("webspaces.config_defaults.http_port", 80)
	viper.SetDefault("webspaces.config_defaults.sni_passthrough", false)
	viper.SetDefault("webspaces.max_startup_delay", 60)
	viper.SetDefault("webspaces.run_limit", 32)
	viper.SetDefault("webspaces.ports.start", 49152)
	viper.SetDefault("webspaces.ports.end", 65535)
	viper.SetDefault("webspaces.ports.max", 64)
	viper.SetDefault("webspaces.ports.kubernetes_service", "")

	viper.SetDefault("http.listen_address", ":80")
	viper.SetDefault("http.cors.allowed_origins", []string{"*"})

	viper.SetDefault("traefik.provider", "redis")
	viper.SetDefault("traefik.redis.addr", "127.0.0.1:6379")
	viper.SetDefault("traefik.redis.db", 0)
	viper.SetDefault("traefik.redis.cert_resolver", "")
	viper.SetDefault("traefik.kubernetes.namespace", "webspace-ng")
	viper.SetDefault("traefik.kubernetes.default_secret", "")
	viper.SetDefault("traefik.kubernetes.cluster_issuer", "")
	viper.SetDefault("traefik.https_entrypoint", "https")
	viper.SetDefault("traefik.default_sans", []string{})
	viper.SetDefault("traefik.webspaced_url", "http://localhost:8080")
	viper.SetDefault("traefik.iam_token", "")

	// Config file loading
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("/run/config")
	viper.AddConfigPath(".")

	// Config from environment
	viper.SetEnvPrefix("WSD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Config from flags
	pflag.StringP("log_level", "l", "info", "log level")
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.WithError(err).Fatal("Failed to bind pflags to config")
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.WithField("err", err).Warn("Failed to read configuration")
	}
}

func reload() {
	if srv != nil {
		stop()
		srv = nil
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg, config.DecoderOptions); err != nil {
		log.WithField("err", err).Fatal("Failed to parse configuration")
	}

	if err := cfg.ReadSecrets(); err != nil {
		log.WithError(err).Fatal("Failed to read config secrets from files")
	}

	log.SetLevel(cfg.LogLevel)
	cJSON, err := json.Marshal(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to encode config as JSON")
	}
	log.WithField("config", string(cJSON)).Debug("Got config")

	srv = server.NewServer(cfg)

	log.Info("Starting server")
	go func() {
		if err := srv.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()
}

func stop() {
	log.Info("Stopping server")
	if err := srv.Stop(); err != nil {
		log.WithError(err).Fatal("Failed to stop server")
	}
}

func main() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.WithField("file", e.Name).Info("Config changed, reloading")
		reload()
	})
	viper.WatchConfig()
	reload()

	<-sigs
	stop()
}
