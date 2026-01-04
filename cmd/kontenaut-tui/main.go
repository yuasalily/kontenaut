package main

import (
	"flag"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine/docker"
	"github.com/yuasalily/kontenaut/internal/ui/tui"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

func main() {
	var endpoint string
	flag.StringVar(
		&endpoint,
		"endpoint",
		"",
		"Docker Engine API endpoint (e.g. unix:///var/run/docker.sock, tcp://123.0.0.1:2375, ssh://user@host). "+
			"If empty, DOCKER_HOST/DOCKER_* env vars are used.",
	)
	flag.Parse()

	var (
		eng *docker.DockerEngine
		err error
	)
	if endpoint == "" {
		eng, err = docker.New()
	} else {
		eng, err = docker.NewWithEndpoint(endpoint)
	}

	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = eng.Close() }()
	containerUC := usecase.NewContainerUsecase(eng)
	imageUC := usecase.NewImageUsecase(eng)
	daemonUC := usecase.NewDaemonUsecase(eng)
	p := tea.NewProgram(tui.New(containerUC, imageUC, daemonUC), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
