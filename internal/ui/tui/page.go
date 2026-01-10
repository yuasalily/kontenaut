package tui

import tea "github.com/charmbracelet/bubbletea"

type pageID int

const (
	pageOverview pageID = iota
	pageImages
	pageContainers
)

// PageMeta describes a page for navigation UI.
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

type openLogsMsg struct {
	// id is a container ID.
	//
	// Why include ID in the message:
	// - The logs page needs a stable reference even if the container list changes after navigation.
	// - It decouples the logs page from table row indices.
	id string
	name string
}

// Page is a Bubble Tea sub-model that renders one screen and handles its own messages.
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
//
//   - DO: capture what you need at Close() call time.
//     Example:
//     func (p *logsPage) Close() tea.Cmd {
//     cancel := p.cancel
//     return func() tea.Msg { cancel(); return nil }
//     }
//
//   - DON'T: read routerModel/currentPage or other global mutable state
//     inside the returned command. Cmd is executed later, after navigation,
//     so global state may already point to another page.
//
//   - If you need to notify the model, include all necessary data in the Msg
//     (do not rely on currentPage at Msg handling time).
//
// Why:
// - Some pages start background work (e.g. log streaming) that must be stopped on navigation.
type PageCloser interface {
	Close() tea.Cmd
}
