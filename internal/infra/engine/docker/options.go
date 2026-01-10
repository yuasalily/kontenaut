package docker

type config struct {
	endpoint string
}

// Option configures DockerEngine construction.
type Option func(*config)

// WithEndpoint sets Docker Engine API endpoint.
//
// Empty string is allowed and means "resolve via docker SDK env vars (DOCKER_HOST/DOCKER_*)"
//
// Why:
// - We keep endpoint selection at startup only.
// - When empty, we intentionally delegate to Docker SDK env resolution for compatibility.
func WithEndpoint(endpoint string) Option {
	return func(cfg *config) {
		if cfg == nil {
			return
		}
		cfg.endpoint = endpoint
	}
}
