package tui

import tea "github.com/charmbracelet/bubbletea"

// setGlobalBusyMsg toggles the router-level busy state
//
// Why router-level busy exists:
// - Some global keys (e.g. navigation 1/2/3) are handled by the router before pages.
// - During destructive operations, allowing navigation can cause context mismatch (dialogs/results).
// - We keep it minimal: it only blocks navigation for now.
type setGlobalBusyMsg struct{ on bool }

func setGlobalBusyCmd(on bool) tea.Cmd {
	return func() tea.Msg {
		return setGlobalBusyMsg{on: on}
	}
}
