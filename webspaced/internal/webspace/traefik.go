package webspace

import (
	"fmt"
	"strings"

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
func NewTraefik(cfg *config.Config) *Traefik {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Traefik.Redis.Addr,
		DB:   cfg.Traefik.Redis.DB,
	})

	return &Traefik{
		cfg,
		client,
	}
}

// UpdateConfig generates new Traefik configuration for a webspace
func (t *Traefik) UpdateConfig(ws *Webspace, addr string) error {
	n := ws.InstanceName()

	if _, err := t.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.Del(
			fmt.Sprintf("traefik/http/services/%v/loadbalancer/servers/0/url", n),

			fmt.Sprintf("traefik/http/routers/%v/service", n),
			fmt.Sprintf("traefik/http/routers/%v/rule", n),
			fmt.Sprintf("traefik/http/routers/%v/entrypoints/0", n),

			fmt.Sprintf("traefik/http/routers/%v-https/service", n),
			fmt.Sprintf("traefik/http/routers/%v-https/rule", n),
			fmt.Sprintf("traefik/http/routers/%v-https/entrypoints/0", n),

			fmt.Sprintf("traefik/http/routers/%v-https/tls", n),
			fmt.Sprintf("traefik/http/routers/%v-https/tls/domains/0/main", n),
			fmt.Sprintf("traefik/http/routers/%v-https/tls/certresolver", n),

			fmt.Sprintf("traefik/tcp/services/%v/loadbalancer/servers/0/address", n),

			fmt.Sprintf("traefik/tcp/routers/%v-https/service", n),
			fmt.Sprintf("traefik/tcp/routers/%v-https/rule", n),
			fmt.Sprintf("traefik/tcp/routers/%v-https/entrypoints/0", n),

			fmt.Sprintf("traefik/tcp/routers/%v-https/tls", n),
			fmt.Sprintf("traefik/tcp/routers/%v-https/tls/domains/0/main", n),
			fmt.Sprintf("traefik/tcp/routers/%v-https/tls/certresolver", n),
			fmt.Sprintf("traefik/tcp/routers/%v-https/tls/passthrough", n),
		)

		if len(t.config.Traefik.SANs) > 0 {
			keys := make([]string, 2*len(t.config.Traefik.SANs))
			for i := 0; i < len(t.config.Traefik.SANs); i++ {
				keys[i*2] = fmt.Sprintf("traefik/http/routers/%v-https/tls/domains/0/sans/%v", n, i)
				keys[i*2+1] = fmt.Sprintf("traefik/tcp/routers/%v-https/tls/domains/0/sans/%v", n, i)
			}
			pipe.Del(keys...)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete redis keys: %w", err)
	}

	if addr == "" {
		return nil
	}

	rules := make([]string, len(ws.Domains))
	for i, d := range ws.Domains {
		rules[i] = fmt.Sprintf("Host(`%v`)", d)
	}
	rule := strings.Join(rules, " || ")

	httpBackend := fmt.Sprintf("http://%v:%v", addr, ws.Config.HTTPPort)
	if _, err := t.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.Set(fmt.Sprintf("traefik/http/services/%v/loadbalancer/servers/0/url", n), httpBackend, 0)

		pipe.Set(fmt.Sprintf("traefik/http/routers/%v/service", n), n, 0)
		pipe.Set(fmt.Sprintf("traefik/http/routers/%v/rule", n), rule, 0)
		pipe.Set(fmt.Sprintf("traefik/http/routers/%v/entrypoints/0", n), t.config.Traefik.HTTPEntryPoint, 0)

		var rt string
		if ws.Config.HTTPSPort == 0 {
			// SSL termination
			rt = "http"

			if len(ws.Domains) > 1 {
				log.WithField("user", ws.User).Warn("Using SSL termination with custom domains - these will be ignored")
			}

			pipe.Set(
				fmt.Sprintf("traefik/http/routers/%v-https/rule", n),
				fmt.Sprintf("Host(`%v.%v`)", ws.User, t.config.Webspaces.Domain),
				0,
			)
		} else {
			// SNI passthrough
			rt = "tcp"

			pipe.Set(
				fmt.Sprintf("traefik/tcp/services/%v/loadbalancer/servers/0/address", n),
				fmt.Sprintf("%v:%v", addr, ws.Config.HTTPSPort),
				0,
			)

			sniRules := make([]string, len(ws.Domains))
			for i, d := range ws.Domains {
				sniRules[i] = fmt.Sprintf("HostSNI(`%v`)", d)
			}
			sniRule := strings.Join(sniRules, " || ")

			pipe.Set(fmt.Sprintf("traefik/tcp/routers/%v-https/rule", n), sniRule, 0)

			pipe.Set(fmt.Sprintf("traefik/tcp/routers/%v-https/tls/passthrough", n), "true", 0)
		}

		pipe.Set(fmt.Sprintf("traefik/%v/routers/%v-https/service", rt, n), n, 0)
		pipe.Set(
			fmt.Sprintf("traefik/%v/routers/%v-https/entrypoints/0", rt, n),
			t.config.Traefik.HTTPSEntryPoint,
			0,
		)

		// TLS-specific configuration
		pipe.Set(fmt.Sprintf("traefik/%v/routers/%v-https/tls", rt, n), "true", 0)
		pipe.Set(
			fmt.Sprintf("traefik/%v/routers/%v-https/tls/domains/0/main", rt, n),
			"*."+t.config.Webspaces.Domain,
			0,
		)
		if t.config.Traefik.CertResolver != "" {
			pipe.Set(
				fmt.Sprintf("traefik/%v/routers/%v-https/tls/certresolver", rt, n),
				t.config.Traefik.CertResolver,
				0,
			)
		}
		for i, san := range t.config.Traefik.SANs {
			pipe.Set(
				fmt.Sprintf("traefik/%v/routers/%v-https/tls/domains/0/sans/%v", rt, n, i),
				san,
				0,
			)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to set redis values: %w", err)
	}

	return nil
}
