package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/engine"
	"github.com/yuasalily/kontenaut/internal/engine/docker"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type model struct {
	uc *usecase.ContainerUsecase

	loading bool
	items   []engine.ContainerSummary
	err     error
}

// compile-time interface check
var _ tea.Model = model{}

func (m model) Init() tea.Cmd {
	return listContainersCmd(m.uc)
}

type containerMsg []engine.ContainerSummary
type errMsg struct{ err error }

func listContainersCmd(uc *usecase.ContainerUsecase) tea.Cmd {
	return func() tea.Msg {
		items, err := uc.List(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return containerMsg(items)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case containerMsg:
		m.loading = false
		m.items = []engine.ContainerSummary(msg)
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if m.loading {
		return "Loading..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\n(q to quit)\n", m.err)
	}

	var b strings.Builder
	b.WriteString("Containers\n")
	b.WriteString("ID IMAGE STATUS NAME\n")
	b.WriteString("------------------------\n")

	for _, c := range m.items {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}

		b.WriteString(fmt.Sprintf(
			"%-12s %-12s %-12s %s\n",
			id, trunc(c.Image, 12),
			trunc(c.Status, 12),
			c.Name,
		))
	}

	b.WriteString("\n(q to quit)\n")
	return b.String()
}

func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "..."
}

func main() {
	eng, err := docker.New()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = eng.Close() }()
	uc := usecase.NewContainerUsecase(eng)
	p := tea.NewProgram(model{uc: uc, loading: true})
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
