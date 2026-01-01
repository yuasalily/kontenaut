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
	dialogConfirm
)

type openDialogMsg struct {
	kind  dialogKind
	title string
	body  string
}

func openDialogCmd(kind dialogKind, title, body string) tea.Cmd {
	return func() tea.Msg {
		return openDialogMsg{
			kind:  kind,
			title: title,
			body:  body,
		}
	}
}

type ConfirmID string

type openConfirmDialogMsg struct {
	id    ConfirmID
	title string
	body  string
}

func openConfirmDialogCmd(id ConfirmID, title, body string) tea.Cmd {
	return func() tea.Msg {
		return openConfirmDialogMsg{
			id:    id,
			title: title,
			body:  body,
		}
	}
}

type confirmDialogResolvedMsg struct {
	id ConfirmID
	ok bool
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

func (d *dialogModel) Update(msg tea.Msg) (closed bool, ok bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = msg.Width, msg.Height
		return false, false
	case tea.KeyMsg:
		switch d.kind {
		case dialogConfirm:
			switch msg.String() {
			case "enter", "y":
				return true, true
			case "esc", "n":
				return true, false
			}
		case dialogInfo, dialogError:
			switch msg.String() {
			case "enter", "esc":
				return true, false
			}
		}
	}
	return false, false
}

func footerTextForDialog(kind dialogKind) string {
	switch kind {
	case dialogConfirm:
		return "y/enter: yes  n/esc: no"
	case dialogInfo, dialogError:
		return "enter/esc: close"
	default:
		return "enter/esc: close"
	}
}

func (d *dialogModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true)

	frame := lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder())

	footer := lipgloss.NewStyle().Faint(true).Render(footerTextForDialog(d.kind))

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
