package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// NavBar renders the top navigation bar
type NavBar struct {
	metas []PageMeta

	baseStyle   lipgloss.Style
	activeStyle lipgloss.Style
}

// NewNavBar constructs a NavBar for the given pages.
func NewNavBar(metas []PageMeta) NavBar {
	return NavBar{
		metas:       metas,
		baseStyle:   lipgloss.NewStyle().Padding(0, 1).Faint(true),
		activeStyle: lipgloss.NewStyle().Padding(0, 1).Bold(true),
	}
}

// Height returns the number of terminal rows used by the navbar.
func (n NavBar) Height() int {
	if len(n.metas) == 0 {
		return 0
	}
	return 1
}

// View renders the navbar with the given page marked as active.
func (n NavBar) View(current pageID) string {
	if len(n.metas) == 0 {
		return ""
	}
	parts := make([]string, 0, len(n.metas))
	for _, meta := range n.metas {
		label := "[" + meta.Key + "] " + meta.Title
		if meta.ID == current {
			parts = append(parts, n.activeStyle.Render(label))
		} else {
			parts = append(parts, n.baseStyle.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

// PageIDFromKey returns the pageID for the given key.
func (n NavBar) PageIDFromKey(k string) (pageID, bool) {
	for _, meta := range n.metas {
		if meta.Key == k {
			return meta.ID, true
		}
	}
	return 0, false
}
