package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// imagesModeController defines mode-specific UI behaviors for the Images page.
type imagesModeController interface {
	Title(p imagesPage) string
	Columns(totalWidth int) []table.Column
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
