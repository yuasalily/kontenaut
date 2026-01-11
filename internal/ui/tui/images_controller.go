package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// imagesModeController defines mode-specific UI behaviors for the Images page.
type imagesModeController interface {
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
	return p.handleKeyNormal(msg)
}

type imagesDeleteController struct{}

// compile-time interface check
var _ imagesModeController = (*imagesDeleteController)(nil)

func newImagesDeleteController() imagesModeController {
	return &imagesDeleteController{}
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
	return p.handleKeyDelete(msg)
}
