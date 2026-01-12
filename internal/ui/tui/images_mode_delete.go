package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// imagesDeleteMode implements the spec's Delete mode:
// - SEL column is visible
// - space toggles selection
// - enter executes (confirm required by router)
type imagesDeleteMode struct{}

// compile-time interface check
var _ imagesMode = (*imagesDeleteMode)(nil)

func newImagesDeleteMode() imagesMode { return &imagesDeleteMode{} }

func (m *imagesDeleteMode) ID() imagesModeID { return imagesModeDelete }

func (m *imagesDeleteMode) Update(ctx imagesCtx, v imagesView, msg tea.Msg) (imagesAction, bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return actNone{}, false
	}

	switch {
	case key.Matches(km, ctx.km.Refresh):
		return actRefresh{}, true

	case key.Matches(km, ctx.km.Exit):
		return actSwitchMode{to: imagesModeNormal}, true

	case key.Matches(km, ctx.km.Select):
		if !v.HasCursor || v.CursorID == "" {
			return actNone{}, true
		}
		// Spec: locked/busy rows are not selectable.
		if !v.CursorSelectable {
			return actNone{}, true
		}
		return actToggleSelect{id: v.CursorID}, true

	case key.Matches(km, ctx.km.Execute):
		// Spec: do nothing when none selected.
		if !v.HasSelection || len(v.SelectedIDs) == 0 {
			return actNone{}, true
		}
		return actRequestDelete{ids: v.SelectedIDs}, true
	}

	return actNone{}, false
}
