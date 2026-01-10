package main

import (
	"flag"
	"fmt"
	"io"
)

type options struct {
	ConfigPath string
	Endpoint   string
}

func parseFlags(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("kontenaut-tui", flag.ContinueOnError)
	// Avoid writing parse errors/help to stdout/stderr during tests.
	fs.SetOutput(io.Discard)

	fs.StringVar(
		&opts.ConfigPath,
		"config",
		"",
		"Path to config file (JSON). If empty, config file is not loaded. "+
			"You can also set it via KONTENAUT_CONFIG",
	)

	fs.StringVar(
		&opts.Endpoint,
		"endpoint",
		"",
		"Docker Engine API endpoint (e.g. unix:///var/run/docker.sock, tcp://123.0.0.1:2375, ssh://user@host). "+
			"If empty, DOCKER_HOST/DOCKER_* env vars are used.",
	)

	fs.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: kontenaut-tui [flags]")
		fmt.Fprintln(flag.CommandLine.Output(), "")
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return opts, nil
}
