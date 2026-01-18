package tui

import "github.com/charmbracelet/bubbles/key"

type containersKeyMap struct {
	Refresh key.Binding

	// Normal mode
	DeleteSingle    key.Binding
	EnterDeleteMode key.Binding
	Logs            key.Binding

	// Delete mode
	Select  key.Binding
	Execute key.Binding
	Exit    key.Binding
}

func newContainersKeyMap() containersKeyMap {
	return containersKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		DeleteSingle: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		EnterDeleteMode: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "delete mode"),
		),
		Logs: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "logs"),
		),
		Select: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "select"),
		),
		Execute: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute"),
		),
		Exit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit"),
		),
	}
}
