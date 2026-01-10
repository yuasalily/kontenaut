package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
)

// renderHelpBlock renders help texts as a footer block with wrapping.
//
// Output format:
// (↑/k: up, ↓/j: down, r: refresh, q: quit)
//
// When the content is too long, it wraps to multiple lines via lipgloss.Wrap.
// maxWidth is the available width in terminal cells. If maxWidth <= 0, no wrapping is applied.
func renderHelpBlock(maxWidth int, bindings ...key.Binding) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		h := b.Help()
		if strings.TrimSpace(h.Key) == "" || strings.TrimSpace(h.Desc) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", h.Key, h.Desc))
	}
	if len(parts) == 0 {
		return ""
	}

	s := strings.Join(parts, ", ")

	// Always wrap with parentheses for footer consistency.
	if maxWidth <= 0 {
		return "(" + s + ")"
	}

	// Reserve 2 cells for "(" and ")".
	w := max(maxWidth - 2, 1)
	wrapped := ansi.Wrap(s, w, ",")
	return "(" + wrapped + ")"
}
