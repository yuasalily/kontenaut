package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type overviewPage struct {
	width  int
	height int
}

// compile-time interface check
var _ Page = overviewPage{}

func newOverviewPage() Page { return overviewPage{} }

func (p overviewPage) Init() tea.Cmd { return nil }

func (p overviewPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil
	}
	return p, nil
}

func (p overviewPage) View() string {
	var b strings.Builder
	b.WriteString("Overview\n\n")
	b.WriteString("q: quit\n")
	return b.String()
}
