package tui

import "github.com/charmbracelet/bubbles/key"

type overviewKeyMap struct {
	Refresh key.Binding
}

func newOverviewKeyMap() overviewKeyMap {
	return overviewKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}
