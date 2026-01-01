package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
)

type dialogKeyMap struct {
	Yes   key.Binding
	No    key.Binding
	Close key.Binding
}

func newDialogKeyMap(kind dialogKind) dialogKeyMap {
	switch kind {
	case dialogConfirm:
		return dialogKeyMap{
			Yes: key.NewBinding(
				key.WithKeys("y", "enter"),
				key.WithHelp("y/enter", "yes"),
			),
			No: key.NewBinding(
				key.WithKeys("n", "esc"),
				key.WithHelp("n/esc", "no"),
			),
		}
	case dialogInfo, dialogError:
		return dialogKeyMap{
			Close: key.NewBinding(
				key.WithKeys("enter", "esc"),
				key.WithHelp("enter/esc", "close"),
			),
		}
	default:
		return dialogKeyMap{
			Close: key.NewBinding(
				key.WithKeys("enter", "esc"),
				key.WithHelp("enter/esc", "close"),
			),
		}
	}
}

func dialogFooter(kind dialogKind, km dialogKeyMap) string {
	switch kind {
	case dialogConfirm:
		y := km.Yes.Help()
		n := km.No.Help()
		return fmt.Sprintf("%s: %s    %s: %s", y.Key, y.Desc, n.Key, n.Desc)

	case dialogInfo, dialogError:
		c := km.Close.Help()
		return fmt.Sprintf("%s: %s", c.Key, c.Desc)

	default:
		c := km.Close.Help()
		return fmt.Sprintf("%s: %s", c.Key, c.Desc)
	}
}
