package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/infra/engine/docker"
	"github.com/yuasalily/kontenaut/internal/ui/tui"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

func run(opts options) error {
	eng, err := newEngine(opts)
	if err != nil {
		return err
	}
	defer func() { _ = eng.Close() }()

	// Why usecases:
	// - UI consumes usecases only; it does not depend on engine implementations.
	// - This keeps Bubble Tea pages testable and keeps infra concerns out of UI.
	containerUC := usecase.NewContainerUsecase(eng)
	imageUC := usecase.NewImageUsecase(eng)
	daemonUC := usecase.NewDaemonUsecase(eng)

	// Why AltScreen:
	// - Provide a clean full-screen TUI experience.
	// - Restore the previous terminal contents on exit.
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
	//
	// Why:
	// - Empty endpoint preserves compatibility with Docker CLI behavior.
	// - Override is opt-in and intended for explicit endpoint selection only.
	return docker.New(docker.WithEndpoint(opts.Endpoint))
}
