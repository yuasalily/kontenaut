package tui

import "github.com/charmbracelet/bubbles/key"

type globalKeyMap struct {
	NavOverview   key.Binding
	NavImages     key.Binding
	NavContainers key.Binding
	Quit          key.Binding
}

func newGlobalKeyMap() globalKeyMap {
	return globalKeyMap{
		NavOverview: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "overview"),
		),
		NavImages: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "images"),
		),
		NavContainers: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "containers"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "q"),
		),
	}
}
