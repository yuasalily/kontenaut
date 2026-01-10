package docker

type config struct {
	endpoint string
}

type Option func(*config)

// WithEndpoint sets Docker Engine API endpoint.
//
// Empty string is allowed and means "resolve via docker SDK env vars (DOCKER_HOST/DOCKER_*)"
func WithEndpoint(endpoint string) Option {
	return func(cfg *config) {
		if cfg == nil {
			return
		}
		cfg.endpoint = endpoint
	}
}
