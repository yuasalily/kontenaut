package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func (m *imagesDeleteMode) Title() string {
	return lipgloss.NewStyle().Bold(true).Render("Images [DELETE MODE]")
}

func (m *imagesDeleteMode) Columns(totalWidth int) []table.Column {
	return columnsForImagesDeleteWidth(totalWidth)
}

func (m *imagesDeleteMode) Rows(st *imagesState) []table.Row {
	return rowsFromImageSummariesDelete(st.items, st.table.Columns(), st.selected, st.locked, st.busy)
}

func (m *imagesDeleteMode) FooterKeys(ctx imagesCtx, st *imagesState) []key.Binding {
	return []key.Binding{
		st.table.KeyMap.LineUp,
		st.table.KeyMap.LineDown,
		ctx.km.Select,
		ctx.km.Execute,
		ctx.km.Exit,
		ctx.km.Refresh,
		ctx.gkm.Quit,
	}
}

func (m *imagesDeleteMode) Update(ctx imagesCtx, st *imagesState, msg tea.Msg) (imagesMode, imagesAction, bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, actNone{}, false
	}

	switch {
	case key.Matches(km, ctx.km.Refresh):
		return nil, actRefresh{}, true

	case key.Matches(km, ctx.km.Exit):
		return newImagesNormalMode(), actExitMode{}, true

	case key.Matches(km, ctx.km.Select):
		id, ok := st.cursorImageID()
		if !ok {
			return nil, actNone{}, true
		}
		// Spec: locked/busy rows are not selectable.
		if st.isLocked(id) || st.isBusy(id) {
			return nil, actNone{}, true
		}
		return nil, actToggleSelect{id: id}, true

	case key.Matches(km, ctx.km.Execute):
		// Spec: do nothing when none selected.
		if len(st.selected) == 0 {
			return nil, actNone{}, true
		}
		return nil, actExecuteDelete{}, true
	}

	return nil, actNone{}, false
}
