package webspace

// Traefik represents a method of programming Traefik router configuration
type Traefik interface {
	// ClearConfig cleans out any configuration for an instance
	ClearConfig(n string) error
	GenerateConfig(ws *Webspace, addr string) error
}
