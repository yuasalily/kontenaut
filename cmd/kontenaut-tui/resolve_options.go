package main

import (
	"fmt"
	"net/url"
)

const envKontenautEndpoint = "KONTENAUT_ENDPOINT"
const envKontenautConfig = "KONTENAUT_CONFIG"

// lookupEnvFunc matches os.LookupEnv.
type lookupEnvFunc func(key string) (string, bool)

// resolveOptions resolves runtime options with precedence:
// defaults < config file < env vars < CLI flags
func resolveOptions(cli options, lookup lookupEnvFunc) (options, error) {
	// defaults
	out := options{}

	// Resolve config path (selection-only) with precedence:
	// env < CLI flag
	//
	// NOTE:
	// - If neither is set, config file is not loaded.
	// - CLI flag must win over env var.
	configPath := ""
	if lookup != nil {
		if v, ok := lookup(envKontenautConfig); ok {
			configPath = v
		}
	}
	if cli.ConfigPath != "" {
		configPath = cli.ConfigPath
	}

	// config file (if provided via env/flag)
	if configPath != "" {
		cfg, err := loadConfigFile(configPath)
		if err != nil {
			return options{}, err
		}
		out = mergeOptions(out, cfg)
	}

	// env vars
	if lookup != nil {
		if v, ok := lookup(envKontenautEndpoint); ok {
			out.Endpoint = v
		}
	}

	// CLI flags
	out = mergeOptions(out, cli)

	// Normalize (ConfigPath is selection-only, but keep the resolved one for visibility)
	out.ConfigPath = configPath

	// validate
	if err := validateOptions(out); err != nil {
		return options{}, err
	}

	return out, nil
}

func mergeOptions(base, override options) options {
	out := base
	// NOTE: ConfigPath is CLI-only; do not merge from config/env.
	if override.Endpoint != "" {
		out.Endpoint = override.Endpoint
	}
	return out
}

func validateOptions(opts options) error {
	// Endpoint:
	// - empty: OK (resolve via docker SDK env vars)
	// - non-empty: basic sanity checks only (avoid over-validation)
	if opts.Endpoint == "" {
		return nil
	}

	u, err := url.Parse(opts.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint %q: %w", opts.Endpoint, err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("invalid endpoint %q: missing scheme (expected scheme://...)", opts.Endpoint)
	}

	switch u.Scheme {
	case "unix":
		// Examples: unix:///var/run/docker.sock
		// Note: host may be empty for unix sockets.
		if u.Path == "" {
			return fmt.Errorf("invalid endpoint %q: unix endpoint must include a socket path (e.g. unix:///var/run/docker.sock)", opts.Endpoint)
		}
		return nil

	case "tcp", "ssh", "npipe":
		// Examples:
		// - tcp://127.0.0.1:2375
		// - ssh://user@host
		// - npipe:////./pipe/docker_engine
		//
		// For these, require some non-empty "host" component when parsed as URL.
		// (npipe on Windows may parse in a slightly odd way depending on slashes,
		// but in typical forms Host is present; this is still "minimum".)
		if u.Host == "" {
			return fmt.Errorf("invalid endpoint %q: %s endpoint must include a host (e.g. %s://host...)", opts.Endpoint, u.Scheme, u.Scheme)
		}
		return nil

	default:
		return fmt.Errorf("invalid endpoint %q: unsupported scheme %q (supported: unix, tcp, ssh, npipe)", opts.Endpoint, u.Scheme)
	}
}
