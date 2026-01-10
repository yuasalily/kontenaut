package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
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

// ConfirmID identifies a confirm dialog request so the caller can match the resolved result.
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

	km dialogKeyMap

	width  int
	height int
}

func newDialog(kind dialogKind, title, body string) *dialogModel {
	return &dialogModel{
		kind:  kind,
		title: title,
		body:  body,
		km:    newDialogKeyMap(kind),
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
			if key.Matches(msg, d.km.Yes) {
				return true, true
			}
			if key.Matches(msg, d.km.No) {
				return true, false
			}

		case dialogInfo, dialogError:
			if key.Matches(msg, d.km.Close) {
				return true, false
			}
		}
	}
	return false, false
}

func (d *dialogModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true)

	frame := lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder())

	footer := lipgloss.NewStyle().Faint(true).Render(dialogFooter(d.kind, d.km))

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
