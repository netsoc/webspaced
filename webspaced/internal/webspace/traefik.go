package webspace

import (
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-redis/redis/v7"
	"github.com/netsoc/webspace-ng/webspaced/internal/config"
	log "github.com/sirupsen/logrus"
)

// Traefik manages webspace configuration for Traefik
type Traefik struct {
	config *config.Config
	redis  *redis.Client
}

// NewTraefik creates a new Traefik instance
func NewTraefik(cfg *config.Config) (*Traefik, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Traefik.Redis.Addr,
		DB:   cfg.Traefik.Redis.DB,
	})

	return &Traefik{
		cfg,
		client,
	}, nil
}

// UpdateConfig generates new Traefik configuration for a webspace
func (t *Traefik) UpdateConfig(ws *Webspace, running bool) error {
	n := ws.InstanceName()

	if _, err := t.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.Del(
			fmt.Sprintf("traefik/http/services/%v/loadbalancer/servers/0/url", n),

			fmt.Sprintf("traefik/http/routers/%v/service", n),
			fmt.Sprintf("traefik/http/routers/%v/rule", n),
			fmt.Sprintf("traefik/http/routers/%v/entrypoints/0", n),

			fmt.Sprintf("traefik/http/services/%v-https/loadbalancer/servers/0/url", n),

			fmt.Sprintf("traefik/http/routers/%v-https/service", n),
			fmt.Sprintf("traefik/http/routers/%v-https/rule", n),
			fmt.Sprintf("traefik/http/routers/%v-https/entrypoints/0", n),

			fmt.Sprintf("traefik/http/routers/%v-https/tls", n),
			fmt.Sprintf("traefik/http/routers/%v-https/tls/domains/0/main", n),
			fmt.Sprintf("traefik/http/routers/%v-https/tls/certresolver", n),
		)

		if len(t.config.Traefik.SANs) > 0 {
			keys := make([]string, len(t.config.Traefik.SANs))
			for i := 0; i < len(t.config.Traefik.SANs); i++ {
				keys[i] = fmt.Sprintf("traefik/http/routers/%v-https/tls/domains/0/sans/%v", n, i)
			}
			pipe.Del(keys...)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete redis keys: %w", err)
	}

	if !running {
		return nil
	}

	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = 20 * time.Second

	var addr string
	if err := backoff.Retry(func() error {
		var err error
		addr, err = ws.GetIP()
		return err
	}, back); err != nil {
		return fmt.Errorf("failed to get instance IP address: %w", err)
	}

	rules := make([]string, len(ws.Domains))
	for i, d := range ws.Domains {
		rules[i] = fmt.Sprintf("Host(`%v`)", d)
	}
	rule := strings.Join(rules, " || ")

	// TODO: https (ssl termination _and_ passthrough)
	httpBackend := fmt.Sprintf("http://%v:%v", addr, ws.Config.HTTPPort)
	if _, err := t.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.Set(fmt.Sprintf("traefik/http/services/%v/loadbalancer/servers/0/url", n), httpBackend, 0)

		pipe.Set(fmt.Sprintf("traefik/http/routers/%v/service", n), n, 0)
		pipe.Set(fmt.Sprintf("traefik/http/routers/%v/rule", n), rule, 0)
		pipe.Set(fmt.Sprintf("traefik/http/routers/%v/entrypoints/0", n), t.config.Traefik.HTTPEntryPoint, 0)

		if ws.Config.HTTPSPort == 0 {
			if len(ws.Domains) > 1 {
				log.WithField("user", ws.User).Warn("Using SSL termination with custom domains - these will be ignored")
			}

			pipe.Set(fmt.Sprintf("traefik/http/services/%v-https/loadbalancer/servers/0/url", n), httpBackend, 0)

			pipe.Set(fmt.Sprintf("traefik/http/routers/%v-https/service", n), n, 0)
			pipe.Set(
				fmt.Sprintf("traefik/http/routers/%v-https/rule", n),
				fmt.Sprintf("Host(`%v.%v`)", ws.User, t.config.Webspaces.Domain),
				0,
			)
			pipe.Set(fmt.Sprintf("traefik/http/routers/%v-https/entrypoints/0", n), t.config.Traefik.HTTPSEntryPoint, 0)

			pipe.Set(fmt.Sprintf("traefik/http/routers/%v-https/tls", n), "true", 0)
			pipe.Set(
				fmt.Sprintf("traefik/http/routers/%v-https/tls/domains/0/main", n),
				"*."+t.config.Webspaces.Domain,
				0,
			)
			if t.config.Traefik.CertResolver != "" {
				pipe.Set(
					fmt.Sprintf("traefik/http/routers/%v-https/tls/certresolver", n),
					t.config.Traefik.CertResolver,
					0,
				)
			}
			for i, san := range t.config.Traefik.SANs {
				pipe.Set(
					fmt.Sprintf("traefik/http/routers/%v-https/tls/domains/0/sans/%v", n, i),
					san,
					0,
				)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to set redis values: %w", err)
	}

	return nil
}
