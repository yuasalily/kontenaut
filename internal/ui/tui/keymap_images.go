package tui

import "github.com/charmbracelet/bubbles/key"

type imagesKeyMap struct {
	Refresh key.Binding
	Select  key.Binding
	Delete  key.Binding
}

func newImagesKeyMap() imagesKeyMap {
	return imagesKeyMap{
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
	}
}
