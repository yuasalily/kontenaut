package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine/docker"
	"github.com/yuasalily/kontenaut/internal/ui/tui"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

func main() {
	eng, err := docker.New()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = eng.Close() }()
	containerUC := usecase.NewContainerUsecase(eng)
	imageUC := usecase.NewImageUsecase(eng)
	p := tea.NewProgram(tui.New(containerUC, imageUC), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
