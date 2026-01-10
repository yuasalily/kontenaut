package main

import (
	"flag"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/infra/engine/docker"
	"github.com/yuasalily/kontenaut/internal/ui/tui"
	"github.com/yuasalily/kontenaut/internal/usecase"
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
		"Path to config file (JSON). If empty, config file is not loaded.",
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

func run(opts options) error {
	eng, err := newEngine(opts)
	if err != nil {
		return err
	}
	defer func() { _ = eng.Close() }()

	containerUC := usecase.NewContainerUsecase(eng)
	imageUC := usecase.NewImageUsecase(eng)
	daemonUC := usecase.NewDaemonUsecase(eng)

	p := tea.NewProgram(tui.New(containerUC, imageUC, daemonUC), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func newEngine(opts options) (engine.Engine, error) {
	// Interpretation of endpoint:
	// - empty: rely on docker SDK env resolution (DOCKER_HOST/DOCKER_*).
	// - non-empty: explicitly override the host.
	return docker.New(docker.WithEndpoint(opts.Endpoint))

}
