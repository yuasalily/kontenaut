package tui

import tea "github.com/charmbracelet/bubbletea"

type pageID int

const (
	pageOverview pageID = iota
	pageContainers
)

type PageMeta struct {
	ID    pageID
	Title string
	Key   string
}

func pageMetas() []PageMeta {
	return []PageMeta{
		{ID: pageOverview, Title: "Overview", Key: "1"},
		{ID: pageContainers, Title: "Containers", Key: "2"},
	}
}

type navigateMsg struct{ to pageID }

type Page interface {
	Init() tea.Cmd
	Update(tea.Msg) (Page, tea.Cmd)
	View() string
}
