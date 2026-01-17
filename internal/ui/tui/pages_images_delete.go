package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

// imagesDeletePage renders Images delete mode as a separate page.
//
// Why:
// - Delete mode has different UI behaviors (selection/execute) and minimal shared state.
// - Keeping it as a separate page reduces branching in the normal Images page.
type imagesDeletePage struct {
	imageUC *usecase.ImageUsecase

	loading  bool
	deleting bool

	images []engine.ImageSummary

	width  int
	height int

	imagesTable table.Model

	selected map[string]struct{}
	locked   map[string]struct{}
	busy     map[string]struct{}

	pendingDeleteIDs []string

	gkm globalKeyMap
	km  imagesKeyMap
}

// compile-time interface check
var _ Page = imagesDeletePage{}

func newImagesDeletePage(gkm globalKeyMap, imagesUC *usecase.ImageUsecase) Page {
	return imagesDeletePage{
		imageUC:     imagesUC,
		loading:     true,
		imagesTable: newImagesTableDelete(),
		selected:    map[string]struct{}{},
		locked:      map[string]struct{}{},
		busy:        map[string]struct{}{},
		gkm:         gkm,
		km:          newImagesKeyMap(),
	}
}

func (p imagesDeletePage) Init() tea.Cmd {
	return tea.Batch(
		listImagesCmd(p.imageUC),
		listLockedImagesCmd(p.imageUC),
	)
}

func (p imagesDeletePage) Update(msg tea.Msg) (Page, tea.Cmd) {
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
		p.imagesTable.SetRows(rowsFromImageSummariesDelete(
			p.images,
			p.imagesTable.Columns(),
			p.selected,
			p.locked,
			p.busy,
		))
		return p, nil

	case imagesLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case lockedImagesLoadedMsg:
		p.locked = map[string]struct{}(msg)
		if len(p.images) > 0 {
			p.imagesTable.SetRows(rowsFromImageSummariesDelete(
				p.images,
				p.imagesTable.Columns(),
				p.selected,
				p.locked,
				p.busy,
			))
		}
		return p, nil

	case lockedImagesLoadFailedMsg:
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case confirmDialogResolvedMsg:
		if msg.id != confirmDeleteImages {
			return p, nil
		}

		ids := p.pendingDeleteIDs

		if !msg.ok || len(ids) == 0 {
			return p, nil
		}
		p.deleting = true
		p.busy = toIDSet(ids)
		return p, deleteImagesCmd(p.imageUC, ids)

	case imagesDeletedMsg:
		p.deleting = false
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.pendingDeleteIDs = nil
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
		// Stay in delete page after operation.
		return p, tea.Sequence(tea.Batch(listImagesCmd(p.imageUC), listLockedImagesCmd(p.imageUC)), dlt)
	}

	var cmd tea.Cmd
	p.imagesTable, cmd = p.imagesTable.Update(msg)
	return p, cmd
}

func (p imagesDeletePage) View() string {
	if p.loading {
		return "Loading..."
	}
	if p.deleting {
		return "Deleting..."
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Images [DELETE MODE]") + "\n")
	b.WriteString(p.imagesTable.View())

	footer := renderHelpBlock(
		p.width,
		p.imagesTable.KeyMap.LineUp,
		p.imagesTable.KeyMap.LineDown,
		p.km.Select,
		p.km.Execute,
		p.km.Exit,
		p.km.Refresh,
		p.gkm.Quit,
	)
	if footer != "" {
		b.WriteString("\n" + footer + "\n")
	}
	return b.String()
}

func (p imagesDeletePage) applyTableLayout() imagesDeletePage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.imagesTable.SetWidth(p.width)
	p.imagesTable.SetHeight(tableHeight)

	cols := columnsForImagesDeleteWidth(p.width)
	p.imagesTable.SetColumns(cols)
	if len(p.images) > 0 {
		p.imagesTable.SetRows(rowsFromImageSummariesDelete(p.images, cols, p.selected, p.locked, p.busy))
	}
	return p
}

func (p imagesDeletePage) handleKey(msg tea.KeyMsg) (imagesDeletePage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	switch {
	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.pendingDeleteIDs = nil
		p.locked = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		id, ok := p.cursorImageID()
		if !ok {
			return p, nil, true
		}
		if _, locked := p.locked[id]; locked {
			// Locked images are not selectable/deletable.
			return p, nil, true
		}
		if _, busy := p.busy[id]; busy {
			return p, nil, true
		}

		// toggle selection
		if _, ok := p.selected[id]; ok {
			delete(p.selected, id)
		} else {
			p.selected[id] = struct{}{}
		}
		p.imagesTable.SetRows(rowsFromImageSummariesDelete(p.images, p.imagesTable.Columns(), p.selected, p.locked, p.busy))
		return p, nil, true

	case key.Matches(msg, p.km.Execute):
		ids := make([]string, 0, len(p.selected))
		for id := range p.selected {
			if _, ok := p.locked[id]; ok {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			// Spec: do nothing when none selected.
			return p, nil, true
		}
		p.pendingDeleteIDs = ids
		body := fmt.Sprintf("Delete %d image(s)", len(ids))
		return p, openConfirmDialogCmd(confirmDeleteImages, "Images", body), true

	case key.Matches(msg, p.km.Exit):
		// Exit delete mode -> back to normal Images page.
		return p, func() tea.Msg { return navigateMsg{to: pageImages} }, true
	}

	return p, nil, false
}

func (p imagesDeletePage) cursorImageID() (string, bool) {
	if len(p.images) == 0 {
		return "", false
	}
	i := p.imagesTable.Cursor()
	if i < 0 || i >= len(p.images) {
		return "", false
	}
	return p.images[i].ID, true
}

func (p imagesDeletePage) setIdle() imagesDeletePage {
	p.loading = false
	p.deleting = false
	return p
}

func newImagesTableDelete() table.Model {
	cols := columnsForImagesDeleteWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForImagesDeleteWidth(total int) []table.Column {
	const (
		selW     = 4
		idW      = 12
		sizeW    = 10
		createdW = 12
	)

	repoW := 24
	if total > 0 {
		rest := total - (selW + idW + sizeW + createdW) - 8
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

func rowsFromImageSummariesDelete(
	items []engine.ImageSummary,
	cols []table.Column,
	selected map[string]struct{},
	locked map[string]struct{},
	busy map[string]struct{},
) []table.Row {
	getW := func(i int, fallback int) int {
		if i < 0 || i >= len(cols) {
			return fallback
		}
		return cols[i].Width
	}

	selW := getW(0, 4)
	idW := getW(1, 12)
	repoW := getW(2, 24)
	sizeW := getW(3, 10)
	createdW := getW(4, 12)

	out := make([]table.Row, 0, len(items))
	for _, img := range items {
		sel := "[ ]"
		if _, ok := busy[img.ID]; ok {
			sel = "[*]"
		} else if _, ok := locked[img.ID]; ok {
			sel = "[#]"
		} else if _, ok := selected[img.ID]; ok {
			sel = "[x]"
		}

		// Why trim sha256 prefix:
		// - Docker image IDs are long; trimming improves table readability.
		// - The remaining prefix is typically sufficient for identification in UI.
		displayID := strings.TrimPrefix(img.ID, "sha256:")
		row := table.Row{
			truncImage(sel, selW),
			truncImage(displayID, idW),
			truncImage(img.RepoTags, repoW),
			truncImage(img.Size, sizeW),
			truncImage(img.CreatedAt, createdW),
		}
		out = append(out, row)
	}
	return out
}
