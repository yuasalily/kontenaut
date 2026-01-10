package main

const envKontenautEndpoint = "KONTENAUT_ENDPOINT"

// lookupEnvFunc matches os.LookupEnv.
type lookupEnvFunc func(key string) (string, bool)

// resolveOptions resolves runtime options with precedence:
// defaults < config file < env vars < CLI flags
func resolveOptions(cli options, lookup lookupEnvFunc) (options, error) {
	// defaults
	out := options{}

	// config file (if provided)
	if cli.ConfigPath != "" {
		cfg, err := loadConfigFile(cli.ConfigPath)
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

	// Normalize (keep ConfigPath from CLI only; it's not a runtime setting itself)
	out.ConfigPath = cli.ConfigPath

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
	// Placeholder for future validations.
	// Example: ensure endpoint scheme looks valid if provided.
	_ = opts
	return nil
}
