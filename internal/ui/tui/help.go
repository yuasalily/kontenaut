package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// renderHelp renders Key.Binding help texts in a compact one-line form.
//
// Example output:
//   ↑/k: up, ↓/j: down, r: refresh, q: quit
//
// It uses each binding's Help() text (set by key.WithHelp) as th single source of truth.
func renderHelp(bindings ...key.Binding) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		h := b.Help()
		if strings.TrimSpace(h.Key) == "" || strings.TrimSpace(h.Desc) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", h.Key, h.Desc))
	}
	return strings.Join(parts, ", ")
}
