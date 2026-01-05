package tui

import "github.com/charmbracelet/bubbles/key"

type containersKeyMap struct {
	Refresh key.Binding
	Select  key.Binding
	Delete  key.Binding
	Logs    key.Binding
}

func newContainersKeyMap() containersKeyMap {
	return containersKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Select: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "select"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Logs: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "logs"),
		),
	}
}
