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
	const (
		selW     = 4
		idW      = 12
		sizeW    = 10
		createdW = 12
	)

	repoW := 24
	if totalWidth > 0 {
		rest := totalWidth - (selW + idW + sizeW + createdW) - 8
		if rest > repoW {
			repoW = rest
		}
	}

	return []table.Column{
		{Title: "SEL", Width: selW},
		{Title: "ID", Width: idW},
		{Title: "REPO:TAG", Width: repoW},
		{Title: "SIZE", Width: sizeW},
		{Title: "CREATED", Width: createdW},
	}
}

func (m *imagesDeleteMode) FooterKeys(ctx imagesCtx) []key.Binding {
	return []key.Binding{
		ctx.km.Select,
		ctx.km.Execute,
		ctx.km.Exit,
		ctx.km.Refresh,
		ctx.gkm.Quit,
	}
}

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
