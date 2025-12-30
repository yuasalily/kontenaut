package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type NavBar struct {
	metas []PageMeta

	baseStyle   lipgloss.Style
	activeStyle lipgloss.Style
}

func NewNavBar(metas []PageMeta) NavBar {
	return NavBar{
		metas:       metas,
		baseStyle:   lipgloss.NewStyle().Padding(0, 1).Faint(true),
		activeStyle: lipgloss.NewStyle().Padding(0, 1).Bold(true),
	}
}

func (n NavBar) Height() int {
	if len(n.metas) == 0 {
		return 0
	}
	return 1
}

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

func (n NavBar) PageIDFromKey(k string) (pageID, bool) {
	for _, meta := range pageMetas() {
		if meta.Key == k {
			return meta.ID, true
		}
	}
	return 0, false
}
