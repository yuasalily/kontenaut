package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config file schema (JSON):
//
//	{
//	   "endpoint": "unix:///var/run/docker.sock"
//	}
type fileConfig struct {
	Endpoint string `json:"endpoint"`
}

func loadConfigFile(path string) (options, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return options{}, wrapConfigError(path, err)
	}

	var cfg fileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return options{}, wrapConfigError(path, err)
	}

	return options{
		Endpoint: cfg.Endpoint,
	}, nil
}

func wrapConfigError(path string, err error) error {
	return fmt.Errorf("failed to load config file %q: %w", path, err)
}
