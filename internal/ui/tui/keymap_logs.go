package tui

import "github.com/charmbracelet/bubbles/key"

type logsKeyMap struct {
	Back     key.Binding
	Follow   key.Binding
	ScrollUp key.Binding
}

func newLogsKeyMap() logsKeyMap {
	return logsKeyMap{
		Back: key.NewBinding(
			key.WithKeys("esc", "b"),
			key.WithHelp("esc/b", "back"),
		),
		Follow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k", "pgup"),
			key.WithHelp("↑/k/pgup", "scroll up"),
		),
	}
}
