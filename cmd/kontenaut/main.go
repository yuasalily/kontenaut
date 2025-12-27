package main

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/engine/docker"
)

type model struct{}

// compole-time interface check
var _ tea.Model = model{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return "kontenaut q to quit"
}

func main() {

	eng, err := docker.New()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = eng.Close() }()
	items, err := eng.ListContainers(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range items {
		fmt.Printf("%s\t%s\t%s", c.ID[:12], c.Image, c.Status)
	}

	// p := tea.NewProgram(model{})
	// if _, err := p.Run(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
}
