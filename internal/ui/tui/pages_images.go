package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type imagesPage struct {
	width  int
	height int
}

// complile-time interface check
var _ Page = imagesPage{}

func newImagesPage() Page { return imagesPage{} }

func (p imagesPage) Init() tea.Cmd { return nil }

func (p imagesPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil
	}
	return p, nil
}

func (p imagesPage) View() string {
	var b strings.Builder
	b.WriteString("Images\n\n")
	b.WriteString("q: quit\n")
	return b.String()
}
