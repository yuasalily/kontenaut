package tui

import tea "github.com/charmbracelet/bubbletea"

type pageID int

const (
	pageStart pageID = iota
	pageContainers
)

type PageMeta struct {
	ID    pageID
	Title string
	Key   string
}

func pageMetas() []PageMeta {
	return []PageMeta{
		{ID: pageStart, Title: "Start", Key: "1"},
		{ID: pageContainers, Title: "Containers", Key: "2"},
	}
}

type navigateMsg struct{ to pageID }

type Page interface {
	Init() tea.Cmd
	Update(tea.Msg) (Page, tea.Cmd)
	View() string
}
