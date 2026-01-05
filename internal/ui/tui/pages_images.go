package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

const confirmDeleteImages ConfirmID = "images:delete"

type imagesPage struct {
	imageUC *usecase.ImageUsecase

	loading  bool
	deleting bool

	images []engine.ImageSummary

	width  int
	height int

	imagesTable table.Model

	selected map[string]struct{}
	locked   map[string]struct{}

	pendingDeleteIDs []string

	km imagesKeyMap
}

// compile-time interface check
var _ Page = imagesPage{}

func newImagesPage(imageUC *usecase.ImageUsecase) Page {
	return imagesPage{
		imageUC:     imageUC,
		loading:     true,
		imagesTable: newImagesTable(),
		selected:    map[string]struct{}{},
		locked:      map[string]struct{}{},
		km:          newImagesKeyMap(),
	}
}

func (p imagesPage) Init() tea.Cmd {
	return tea.Batch(
		listImagesCmd(p.imageUC),
		listLockedImagesCmd(p.imageUC),
	)
}

type imagesLoadedMsg []engine.ImageSummary
type imagesLoadFailedMsg struct{ err error }

func listImagesCmd(imageUC *usecase.ImageUsecase) tea.Cmd {
	return func() tea.Msg {
		items, err := imageUC.List(context.Background())
		if err != nil {
			return imagesLoadFailedMsg{err: err}
		}
		return imagesLoadedMsg(items)
	}
}

type lockedImagesLoadedMsg map[string]struct{}
type lockedImagesLoadFailedMsg struct{ err error }

func listLockedImagesCmd(imageUC *usecase.ImageUsecase) tea.Cmd {
	return func() tea.Msg {
		locked, err := imageUC.LockedImageIDs(context.Background())
		if err != nil {
			return lockedImagesLoadFailedMsg{err: err}
		}
		return lockedImagesLoadedMsg(locked)
	}
}

type imagesDeletedMsg struct {
	deleted  int
	failed   int
	firstErr error
}

func deleteImagesCmd(imagesUC *usecase.ImageUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		deleted := 0
		failed := 0
		var firstErr error
		for _, id := range ids {
			if err := imagesUC.Delete(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			deleted++
		}
		return imagesDeletedMsg{
			deleted:  deleted,
			failed:   failed,
			firstErr: firstErr,
		}
	}
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
		p.loading = false
		p.images = []engine.ImageSummary(msg)
		p.imagesTable.SetRows(rowsFromImageSummaries(p.images, p.imagesTable.Columns(), p.selected, p.locked))
		return p, nil

	case imagesLoadFailedMsg:
		p.loading = false
		p.deleting = false
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case lockedImagesLoadedMsg:
		p.locked = map[string]struct{}(msg)
		if len(p.images) > 0 {
			p.imagesTable.SetRows(rowsFromImageSummaries(p.images, p.imagesTable.Columns(), p.selected, p.locked))
		}
		return p, nil

	case lockedImagesLoadFailedMsg:
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case confirmDialogResolvedMsg:
		if msg.id != confirmDeleteImages {
			return p, nil
		}

		ids := p.pendingDeleteIDs
		p.pendingDeleteIDs = nil

		if !msg.ok || len(ids) == 0 {
			return p, nil
		}
		p.deleting = true
		return p, deleteImagesCmd(p.imageUC, ids)

	case imagesDeletedMsg:
		p.deleting = false
		p.loading = true
		p.selected = map[string]struct{}{}

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
	b.WriteString("\n(space: select, d: delete, r: refresh, q: quit)\n")
	return b.String()
}

func (p imagesPage) handleKey(msg tea.KeyMsg) (imagesPage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	switch {
	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.selected = map[string]struct{}{}
		p.locked = map[string]struct{}{}
		p.pendingDeleteIDs = nil
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		id, ok := p.cursorImageID()
		if !ok {
			return p, nil, true
		}
		if p.isLocked(id) {
			return p, nil, true
		}
		p.toggleSelected(id)
		p.imagesTable.SetRows(rowsFromImageSummaries(p.images, p.imagesTable.Columns(), p.selected, p.locked))
		return p, nil, true

	case key.Matches(msg, p.km.Delete):
		ids := p.selectedDeletableIDs()
		if len(ids) == 0 {
			return p, openDialogCmd(dialogInfo, "Images", "No images selected"), true
		}
		p.pendingDeleteIDs = ids
		body := fmt.Sprintf("Delete %d image(s)?", len(ids))
		return p, openConfirmDialogCmd(confirmDeleteImages, "Images", body), true
	}

	return p, nil, false
}

func (p imagesPage) applyTableLayout() imagesPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-4, 1)

	p.imagesTable.SetWidth(p.width)
	p.imagesTable.SetHeight(tableHeight)

	cols := columnsForImagesWidth(p.width)
	p.imagesTable.SetColumns(cols)
	if len(p.images) > 0 {
		p.imagesTable.SetRows(rowsFromImageSummaries(p.images, cols, p.selected, p.locked))
	}
	return p
}

func newImagesTable() table.Model {
	cols := columnsForImagesWidth(0)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
	return t
}

func columnsForImagesWidth(total int) []table.Column {
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

func rowsFromImageSummaries(items []engine.ImageSummary, cols []table.Column, selected map[string]struct{}, locked map[string]struct{}) []table.Row {
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
		if _, ok := locked[img.ID]; ok {
			sel = "[#]"
		} else if _, ok := selected[img.ID]; ok {
			sel = "[x]"
		}
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

func truncImage(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "..."
}

func (p imagesPage) isLocked(id string) bool {
	_, ok := p.locked[id]
	return ok
}

func (p *imagesPage) toggleSelected(id string) {
	if _, ok := p.selected[id]; ok {
		delete(p.selected, id)
		return
	}
	p.selected[id] = struct{}{}
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

func (p imagesPage) selectedDeletableIDs() []string {
	out := make([]string, 0, len(p.selected))
	for id := range p.selected {
		if _, ok := p.locked[id]; ok {
			continue
		}
		out = append(out, id)
	}
	return out
}
