package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type deleteSingleImagesConfirmedMsg struct {
	id string
}

// imagesPage renders Images list (normal mode).
type imagesPage struct {
	imageUC *usecase.ImageUsecase

	loading  bool
	deleting bool

	images []engine.ImageSummary

	width  int
	height int

	imagesTable table.Model

	locked map[string]struct{}

	gkm globalKeyMap
	km  imagesKeyMap
}

// compile-time interface check
var _ Page = imagesPage{}

func newImagesPage(gkm globalKeyMap, imageUC *usecase.ImageUsecase) Page {
	return imagesPage{
		imageUC:     imageUC,
		loading:     true,
		imagesTable: newImagesTableNormal(),
		locked:      map[string]struct{}{},
		gkm:         gkm,
		km:          newImagesKeyMap(),
	}
}

func (p imagesPage) Init() tea.Cmd {
	return tea.Batch(
		listImagesCmd(p.imageUC),
		listLockedImagesCmd(p.imageUC),
	)
}

func (p imagesPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		p = p.applyTableLayout()
		return p, nil

	case tea.KeyMsg:
		np, cmd, handled := p.handleKey(msg)
		if handled {
			return np, cmd
		}

	case imagesLoadedMsg:
		p = p.setIdle()
		p.images = []engine.ImageSummary(msg)
		p.imagesTable.SetRows(rowsFromImageSummariesNormal(p.images, p.imagesTable.Columns()))
		return p, nil

	case imagesLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case lockedImagesLoadedMsg:
		p.locked = map[string]struct{}(msg)
		return p, nil

	case lockedImagesLoadFailedMsg:
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case deleteSingleImagesConfirmedMsg:
		if msg.id == "" {
			return p, nil
		}
		p.deleting = true
		return p, deleteImagesCmd(p.imageUC, []string{msg.id})

	case imagesDeletedMsg:
		p.deleting = false
		p.loading = true
		p.locked = map[string]struct{}{}

		var dlt tea.Cmd
		if msg.failed == 0 {
			dlt = openDialogCmd(dialogInfo, "Images", fmt.Sprintf("Deleted %d image(s)", msg.deleted))
		} else {
			body := fmt.Sprintf("Deleted %d image(s). Failed %d image(s).", msg.deleted, msg.failed)
			if msg.firstErr != nil {
				body = fmt.Sprintf("%s\n\n%s", body, msg.firstErr.Error())
			}
			dlt = openDialogCmd(dialogError, "Images", body)
		}
		return p, tea.Sequence(tea.Batch(listImagesCmd(p.imageUC), listLockedImagesCmd(p.imageUC)), dlt)
	}

	var cmd tea.Cmd
	p.imagesTable, cmd = p.imagesTable.Update(msg)
	return p, cmd
}

func (p imagesPage) View() string {
	if p.loading {
		return "Loading..."
	}
	if p.deleting {
		return "Deleting..."
	}
	var b strings.Builder
	b.WriteString("Images\n")

	b.WriteString(p.imagesTable.View())
	footer := renderHelpBlock(
		p.width,
		p.imagesTable.KeyMap.LineUp,
		p.imagesTable.KeyMap.LineDown,
		p.km.DeleteSingle,
		p.km.EnterDeleteMode,
		p.km.Refresh,
		p.gkm.Quit,
	)
	if footer != "" {
		b.WriteString("\n" + footer + "\n")
	}
	return b.String()
}

func (p imagesPage) handleKey(msg tea.KeyMsg) (imagesPage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	switch {
	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.locked = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.EnterDeleteMode):
		return p, func() tea.Msg { return openImagesDeleteMsg{} }, true

	case key.Matches(msg, p.km.DeleteSingle):
		id, ok := p.cursorImageID()
		if !ok {
			return p, nil, true
		}
		if p.isLocked(id) {
			return p, openDialogCmd(dialogInfo, "Images", "this image is in use and cannot be selected."), true
		}
		return p, openConfirmDialogCmd(
			"Images",
			"Delete 1 image?",
			deleteSingleImagesConfirmedMsg{id: id},
			nil,
		), true
	}

	return p, nil, false
}

func (p imagesPage) applyTableLayout() imagesPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.imagesTable.SetWidth(p.width)
	p.imagesTable.SetHeight(tableHeight)

	cols := columnsForImagesNormalWidth(p.width)
	p.imagesTable.SetColumns(cols)
	if len(p.images) > 0 {
		p.imagesTable.SetRows(rowsFromImageSummariesNormal(p.images, cols))
	}
	return p
}

func newImagesTableNormal() table.Model {
	cols := columnsForImagesNormalWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForImagesNormalWidth(total int) []table.Column {
	const (
		idW      = 12
		sizeW    = 10
		createdW = 12
	)

	repoW := 24
	if total > 0 {
		rest := total - (idW + sizeW + createdW) - 6
		if rest > repoW {
			repoW = rest
		}
	}

	return []table.Column{
		{Title: "ID", Width: idW},
		{Title: "REPO:TAG", Width: repoW},
		{Title: "SIZE", Width: sizeW},
		{Title: "CREATED", Width: createdW},
	}
}

func rowsFromImageSummariesNormal(items []engine.ImageSummary, cols []table.Column) []table.Row {
	idW := colWidth(cols, 0, 12)
	repoW := colWidth(cols, 1, 24)
	sizeW := colWidth(cols, 2, 10)
	createdW := colWidth(cols, 3, 12)

	out := make([]table.Row, 0, len(items))
	for _, img := range items {
		// Why trim sha256 prefix:
		// - Docker image IDs are long; trimming improves table readability.
		// - The remaining prefix is typically sufficient for identification in UI.
		displayID := strings.TrimPrefix(img.ID, "sha256:")
		row := table.Row{
			truncText(displayID, idW),
			truncText(img.RepoTags, repoW),
			truncText(img.Size, sizeW),
			truncText(img.CreatedAt, createdW),
		}
		out = append(out, row)
	}
	return out
}

func (p imagesPage) isLocked(id string) bool {
	_, ok := p.locked[id]
	return ok
}

func (p imagesPage) cursorImageID() (string, bool) {
	if len(p.images) == 0 {
		return "", false
	}
	i := p.imagesTable.Cursor()
	if i < 0 || i >= len(p.images) {
		return "", false
	}
	return p.images[i].ID, true
}

func (p imagesPage) setIdle() imagesPage {
	p.loading = false
	p.deleting = false
	return p
}
