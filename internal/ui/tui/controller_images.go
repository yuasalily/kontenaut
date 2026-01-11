package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// imagesModeController defines mode-specific UI behaviors for the Images page.
type imagesModeController interface {
	ID() imagesMode
	Title(p imagesPage) string
	Columns(totalWidth int) []table.Column
	NewTable() table.Model
	FooterKeys(p imagesPage) []key.Binding
	Rows(p imagesPage) []table.Row
	HandleKey(p imagesPage, msg tea.KeyMsg) (imagesPage, tea.Cmd, bool)
}

type imagesNormalController struct{}

// compile-time interface check
var _ imagesModeController = (*imagesNormalController)(nil)

func newImagesNormalController() imagesModeController {
	return &imagesNormalController{}
}

func (c *imagesNormalController) ID() imagesMode {
	return imagesModeNormal
}

func (c *imagesNormalController) Title(p imagesPage) string {
	return "Images"
}

func (c *imagesNormalController) Columns(totalWidth int) []table.Column {
	return columnsForImagesNormalWidth(totalWidth)
}

func (c *imagesNormalController) NewTable() table.Model {
	return newImagesTableNormal()
}

func (c *imagesNormalController) FooterKeys(p imagesPage) []key.Binding {
	return []key.Binding{
		p.imagesTable.KeyMap.LineUp,
		p.imagesTable.KeyMap.LineDown,
		p.km.DeleteSingle,
		p.km.EnterDeleteMode,
		p.km.Refresh,
		p.gkm.Quit,
	}
}

func (c *imagesNormalController) Rows(p imagesPage) []table.Row {
	return rowsFromImageSummariesNormal(p.images, p.imagesTable.Columns())
}

func (c *imagesNormalController) HandleKey(p imagesPage, msg tea.KeyMsg) (imagesPage, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, p.km.Refresh):
		// Refresh keeps mode; selection is cleared.
		p.loading = true
		p = p.resetTransientState(true)
		return p, p.Init(), true

	case key.Matches(msg, p.km.EnterDeleteMode):
		p = p.switchMode(imagesModeDelete)
		return p, nil, true

	case key.Matches(msg, p.km.DeleteSingle):
		id, ok := p.cursorImageID()
		if !ok {
			return p, nil, true
		}
		if p.isLocked(id) {
			return p, openDialogCmd(dialogInfo, "Images", "this image is in use and cannot be deleted."), true
		}
		p.pendingDeleteIDs = []string{id}
		return p, openConfirmDialogCmd(confirmDeleteImages, "Images", "Delete 1 image?"), true
	}

	return p, nil, false
}

type imagesDeleteController struct{}

// compile-time interface check
var _ imagesModeController = (*imagesDeleteController)(nil)

func newImagesDeleteController() imagesModeController {
	return &imagesDeleteController{}
}

func (c *imagesDeleteController) ID() imagesMode {
	return imagesModeDelete
}

func (c *imagesDeleteController) Title(p imagesPage) string {
	return lipgloss.NewStyle().Bold(true).Render("Images [DELETE MODE]")
}

func (c *imagesDeleteController) Columns(totalWidth int) []table.Column {
	return columnsForImagesDeleteWidth(totalWidth)
}

func (c *imagesDeleteController) NewTable() table.Model {
	return newImagesTableDelete()
}

func (c *imagesDeleteController) FooterKeys(p imagesPage) []key.Binding {
	return []key.Binding{
		p.imagesTable.KeyMap.LineUp,
		p.imagesTable.KeyMap.LineDown,
		p.km.Select,
		p.km.Execute,
		p.km.Exit,
		p.km.Refresh,
		p.gkm.Quit,
	}
}

func (c *imagesDeleteController) Rows(p imagesPage) []table.Row {
	return rowsFromImageSummariesDelete(p.images, p.imagesTable.Columns(), p.selected, p.locked, p.busy)
}

func (c *imagesDeleteController) HandleKey(p imagesPage, msg tea.KeyMsg) (imagesPage, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, p.km.Refresh):
		// Refresh keeps delete mode; selection is cleared.
		p.loading = true
		p = p.resetTransientState(true)
		return p, p.Init(), true

	case key.Matches(msg, p.km.Exit):
		p = p.switchMode(imagesModeNormal)
		return p, nil, true

	case key.Matches(msg, p.km.Select):
		id, ok := p.cursorImageID()
		if !ok {
			return p, nil, true
		}
		if p.isLocked(id) {
			// Locked images are not selectable/deletable.
			return p, nil, true
		}
		p.toggleSelected(id)
		p.imagesTable.SetRows(p.rowsForMode())
		return p, nil, true

	case key.Matches(msg, p.km.Execute):
		ids := p.selectedDeletableIDs()
		if len(ids) == 0 {
			// Spec: do nothing when none selected.
			return p, nil, true
		}
		p.pendingDeleteIDs = ids
		body := fmt.Sprintf("Delete %d image(s)", len(ids))
		return p, openConfirmDialogCmd(confirmDeleteImages, "Images", body), true
	}

	return p, nil, false
}
