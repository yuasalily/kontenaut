package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dialogKind int

const (
	dialogInfo dialogKind = iota
	dialogError
)

type showDialogMsg struct {
	kind  dialogKind
	title string
	body  string
}

func showDialogCmd(kind dialogKind, title, body string) tea.Cmd {
	return func() tea.Msg {
		return showDialogMsg{
			kind:  kind,
			title: title,
			body:  body,
		}
	}
}

type dialogModel struct {
	kind  dialogKind
	title string
	body  string

	width  int
	height int
}

func newDialog(kind dialogKind, title, body string) *dialogModel {
	return &dialogModel{
		kind:  kind,
		title: title,
		body:  body,
	}
}

func (d *dialogModel) Update(msg tea.Msg) (closed bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			return true
		}
	}
	return false
}

func (d *dialogModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true)

	frame := lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder())

	footer := lipgloss.NewStyle().Faint(true).Render("Enter/Esc: close")

	var b strings.Builder
	b.WriteString(titleStyle.Render(d.title))
	b.WriteString("\n\n")
	b.WriteString(d.body)
	b.WriteString("\n\n")
	b.WriteString(footer)

	box := frame.Render(b.String())

	w, h := d.width, d.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
