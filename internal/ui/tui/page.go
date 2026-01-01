package tui

import tea "github.com/charmbracelet/bubbletea"

type pageID int

const (
	pageOverview pageID = iota
	pageImages
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
		{ID: pageImages, Title: "Images", Key: "2"},
		{ID: pageContainers, Title: "Containers", Key: "3"},
	}
}

type navigateMsg struct{ to pageID }

type Page interface {
	Init() tea.Cmd
	Update(tea.Msg) (Page, tea.Cmd)
	View() string
}

// PageCloser is an optional lifecycle hook.
// Router calls Close() right before replacing the current page.
// Close() is expected to stop background tasks started by the page (e.g. streaming logs).
// Most pages can ignore this and implement nothing.
// 
// Close() implementation guideline:
//  - DO: capture what you need at Close() call time.
//    Example:
//      func (p *logsPage) Close() tea.Cmd {
//        cancel := p.cancel
//        return func() tea.Msg { cancel(); return nil }
//      }
//
//  - DON'T: read routerModel/currentPage or other global mutable state
//    inside the returned command. Cmd is executed later, after navigation,
//    so global state may already point to another page.
//
//  - If you need to notify the model, include all necessary data in the Msg
//    (do not rely on currentPage at Msg handling time).
type PageCloser interface {
	Close() tea.Cmd
}