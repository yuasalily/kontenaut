package tui

import "github.com/charmbracelet/bubbles/key"

type containersKeyMap struct {
	Refresh key.Binding

	// Normal mode
	DeleteSingle    key.Binding
	EnterDeleteMode key.Binding
	Logs            key.Binding
	StartSingle     key.Binding
	EnterStartMode  key.Binding
	StopSingle      key.Binding
	RestartSingle   key.Binding

	// Action mode
	//
	// These bindings are shared by multi-select action pages.
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
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		StartSingle: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		EnterStartMode: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "start mode"),
		),
		StopSingle: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "stop"),
		),
		RestartSingle: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "restart"),
		),
		Select: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "select"),
		),
		Execute: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "execute"),
		),
		Exit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit"),
		),
	}
}
