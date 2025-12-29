package tui

import tea "github.com/charmbracelet/bubbletea"

type pageID int

const (
	pageStart pageID = iota
	pageContainers
)

type navigateMsg struct{to pageID}

type Page interface {
	Init() tea.Cmd
	Update(tea.Msg) (Page, tea.Cmd)
	View() string
}