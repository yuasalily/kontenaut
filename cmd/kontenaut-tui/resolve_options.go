package main

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
	// Placeholder for future validations.
	// Example: ensure endpoint scheme looks valid if provided.
	_ = opts
	return nil
}
