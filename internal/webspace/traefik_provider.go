package webspace

import "context"

// Traefik represents a method of programming Traefik router configuration
type Traefik interface {
	// ClearConfig cleans out any configuration for an instance
	ClearConfig(ctx context.Context, n string) error
	// GenerateConfig generates configuration for an instance
	GenerateConfig(ctx context.Context, ws *Webspace, addr string) error
}
