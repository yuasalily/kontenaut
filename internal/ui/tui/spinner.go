package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// newLoadingSpinner Spinner creates a spinner for loading states.
//
// Why:
// - Multiple pages share "Loading..." UI.
// - Centralize spinner choice/style for consistency.
func newLoadingSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Faint(true)
	return s
}
