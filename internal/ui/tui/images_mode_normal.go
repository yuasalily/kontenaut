package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// imagesNormalMode implements the spec's Normal mode:
// - No SEL column
// 'd' deletes the cursor row (confirm is handled by router)
// 'D' enters delete mode
// - enter is noop (Images only)
type imagesNormalMode struct{}

// compile-time interface check
var _ imagesMode = (*imagesNormalMode)(nil)

func newImagesNormalMode() imagesMode { return &imagesNormalMode{} }

func (m *imagesNormalMode) ID() imagesModeID { return imagesModeNormal }

func (m *imagesNormalMode) Update(ctx imagesCtx, v imagesView, msg tea.Msg) (imagesAction, bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return actNone{}, false
	}

	switch {
	case key.Matches(km, ctx.km.Refresh):
		return actRefresh{}, true

	case key.Matches(km, ctx.km.EnterDeleteMode):
		return actSwitchMode{to: imagesModeDelete}, true

	case key.Matches(km, ctx.km.DeleteSingle):
		if !v.HasCursor || v.CursorID == "" {
			return actNone{}, true
		}
		return actRequestDelete{ids: []string{v.CursorID}}, true
	}

	// Images Normal: enter is noop. Swallow it so it doesn't accidentally trigger table action.
	if strings.ToLower(km.String()) == "enter" {
		return actNone{}, true
	}

	return actNone{}, false
}
