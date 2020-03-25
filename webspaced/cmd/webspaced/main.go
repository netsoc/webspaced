package main

import (
	"os"
	"os/signal"

	"github.com/netsoc/webspace-ng/webspaced/internal/config"
	"github.com/netsoc/webspace-ng/webspaced/internal/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
)

func init() {
	viper.SetDefault("log_level", log.InfoLevel)
	viper.SetDefault("bind_socket", "/run/webspaced/server.sock")
	viper.SetDefault("pw_gr_proxy_socket", "/run/webspaced/pw_gr_proxy.sock")
	viper.SetDefault("lxd.socket", "/run/lxd.sock")
	viper.SetDefault("lxd.network", "lxdbr0")
	viper.SetDefault("webspaces.admin_group", "webspace-admin")
	viper.SetDefault("webspaces.profile", "webspace")
	viper.SetDefault("webspaces.instance_suffix", "-ws")
	viper.SetDefault("webspaces.domain", "ng.localhost")
	viper.SetDefault("webspaces.config_defaults.startup_delay", 3)
	viper.SetDefault("webspaces.config_defaults.http_port", 80)
	viper.SetDefault("webspaces.config_defaults.https_port", 0)
	viper.SetDefault("webspaces.max_startup_delay", 60)
	viper.SetDefault("webspaces.run_limit", 32)
	viper.SetDefault("webspaces.ports.start", 49152)
	viper.SetDefault("webspaces.ports.end", 65535)
	viper.SetDefault("webspaces.ports.max", 64)
	viper.SetDefault("traefik.redis.addr", "redis:6379")
	viper.SetDefault("traefik.redis.db", 0)
	viper.SetDefault("traefik.http_entry_point", "http")
	viper.SetDefault("traefik.https_entry_point", "https")

	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/webspaced")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		log.WithField("err", err).Fatal("Failed to read configuration")
	}
}
func main() {
	var cfg config.Config
	if err := viper.Unmarshal(&cfg, config.DecoderOptions); err != nil {
		log.WithField("err", err).Fatal("Failed to parse configuration")
	}

	log.SetLevel(cfg.LogLevel)
	srv := server.NewServer(cfg)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM)

	go func() {
		log.Info("Starting server...")
		if err := srv.Start(); err != nil {
			log.WithField("error", err).Fatal("Failed to start server")
		}
	}()

	<-sigs
	srv.Stop()
}
