package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type startPage struct {
	width  int
	height int
}

func newStartPage() Page { return startPage{} }

func (p startPage) Init() tea.Cmd { return nil }

func (p startPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "s":
			return p, func() tea.Msg { return navigateMsg{to: pageContainers} }
		}
	}
	return p, nil
}

func (p startPage) View() string {
	var b strings.Builder
	b.WriteString("kontenaut\n\n")
	b.WriteString("Press Enter to start\n")
	b.WriteString("  - lists Docker containers\n\n")
	b.WriteString("(enter: start, q: quit)\n")
	return b.String()
}
