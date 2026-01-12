package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
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

func (m *imagesNormalMode) Title() string {
	return "Images"
}

func (m *imagesNormalMode) Columns(totalWidth int) []table.Column {
	return columnsForImagesNormalWidth(totalWidth)
}

func (m *imagesNormalMode) Rows(st *imagesState) []table.Row {
	return rowsFromImageSummariesNormal(st.items, st.table.Columns())
}

func (m *imagesNormalMode) FooterKeys(ctx imagesCtx, st *imagesState) []key.Binding {
	return []key.Binding{
		st.table.KeyMap.LineUp,
		st.table.KeyMap.LineDown,
		ctx.km.DeleteSingle,
		ctx.km.EnterDeleteMode,
		ctx.km.Refresh,
		ctx.gkm.Quit,
	}
}

func (m *imagesNormalMode) Update(ctx imagesCtx, st *imagesState, msg tea.Msg) (imagesMode, imagesAction, bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, actNone{}, false
	}

	switch {
	case key.Matches(km, ctx.km.Refresh):
		return nil, actRefresh{}, true

	case key.Matches(km, ctx.km.EnterDeleteMode):
		return newImagesDeleteMode(), actEnterDeleteMode{}, true

	case key.Matches(km, ctx.km.DeleteSingle):
		id, ok := st.cursorImageID()
		if !ok {
			return nil, actNone{}, true
		}
		return nil, actRequestDelete{ids: []string{id}}, true
	}

	// Images Normal: enter is noop. Swallow it so it doesn't accidentally trigger table action.
	if strings.ToLower(km.String()) == "enter" {
		return nil, actNone{}, true
	}

	return nil, actNone{}, false
}
