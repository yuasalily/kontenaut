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

type imagesMode int

const (
	imagesModeNormal imagesMode = iota
	imagesModeDelete
)

// imagesPage renders the Images list and destructive actions (delete).
//
// Why:
// - Docker prevents deleting images in use; we compute "locked" to prevent noisy errors.
// - UI concerns (selection, confirmation) live here; usecases perform operations.
type imagesPage struct {
	imageUC *usecase.ImageUsecase

	// ctrl is the controller for the current mode.
	ctrl imagesModeController

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
var _ Page = imagesPage{}

func newImagesPage(gkm globalKeyMap, imageUC *usecase.ImageUsecase) Page {
	return imagesPage{
		imageUC:     imageUC,
		ctrl:        newImagesNormalController(),
		loading:     true,
		imagesTable: newImagesTableNormal(),
		selected:    map[string]struct{}{},
		locked:      map[string]struct{}{},
		busy:        map[string]struct{}{},
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

func (p imagesPage) currentController() imagesModeController {
	return p.ctrl
}

func (p imagesPage) titleForMode() string {
	return p.currentController().Title(p)
}

func (p imagesPage) columnsForMode(width int) []table.Column {
	return p.currentController().Columns(width)
}

func (p imagesPage) footerKeysForMode() []key.Binding {
	return p.currentController().FooterKeys(p)
}

func (p imagesPage) rowsForMode() []table.Row {
	return p.currentController().Rows(p)
}

func (p imagesPage) handleKeyForMode(msg tea.KeyMsg) (imagesPage, tea.Cmd, bool) {
	return p.currentController().HandleKey(p, msg)
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
		p.imagesTable.SetRows(p.rowsForMode())
		return p, nil

	case imagesLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case lockedImagesLoadedMsg:
		p.locked = map[string]struct{}(msg)
		if len(p.images) > 0 {
			p.imagesTable.SetRows(p.rowsForMode())
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
		p.busy = toIDSet(ids)
		return p, deleteImagesCmd(p.imageUC, ids)

	case imagesDeletedMsg:
		p.deleting = false
		p.loading = true
		p = p.resetTransientState(true)

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
	b.WriteString(p.titleForMode() + "\n")

	b.WriteString(p.imagesTable.View())
	footer := renderHelpBlock(p.width, p.footerKeysForMode()...)
	if footer != "" {
		b.WriteString("\n" + footer + "\n")
	}
	return b.String()
}

func (p imagesPage) handleKey(msg tea.KeyMsg) (imagesPage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	return p.handleKeyForMode(msg)
}

func (p imagesPage) applyTableLayout() imagesPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.imagesTable.SetWidth(p.width)
	p.imagesTable.SetHeight(tableHeight)

	// Always apply mode-specific columns/rows.
	// Why:
	// - Even when there are no items, mode changes should be reflected immediately (e.g. SEL column).
	p.imagesTable.SetColumns(p.columnsForMode(p.width))
	p.imagesTable.SetRows(p.rowsForMode())
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

func newImagesTableDelete() table.Model {
	cols := columnsForImagesDeleteWidth(0)
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

func rowsFromImageSummariesNormal(items []engine.ImageSummary, cols []table.Column) []table.Row {
	getW := func(i int, fallback int) int {
		if i < 0 || i >= len(cols) {
			return fallback
		}
		return cols[i].Width
	}

	idW := getW(0, 12)
	repoW := getW(1, 24)
	sizeW := getW(2, 10)
	createdW := getW(3, 12)

	out := make([]table.Row, 0, len(items))
	for _, img := range items {
		// Why trim sha256 prefix:
		// - Docker image IDs are long; trimming improves table readability.
		// - The remaining prefix is typically sufficient for identification in UI.
		displayID := strings.TrimPrefix(img.ID, "sha256:")
		row := table.Row{
			truncImage(displayID, idW),
			truncImage(img.RepoTags, repoW),
			truncImage(img.Size, sizeW),
			truncImage(img.CreatedAt, createdW),
		}
		out = append(out, row)
	}
	return out
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

// resetTransientState clears in-memory transient state for Images page.
func (p imagesPage) resetTransientState(clearLocked bool) imagesPage {
	p.selected = map[string]struct{}{}
	p.pendingDeleteIDs = nil
	p.busy = map[string]struct{}{}
	if clearLocked {
		p.locked = map[string]struct{}{}
	}
	return p
}

func (p imagesPage) setIdle() imagesPage {
	p.loading = false
	p.deleting = false
	return p
}

func newImagesController(mode imagesMode) imagesModeController {
	switch mode {
	case imagesModeDelete:
		return newImagesDeleteController()
	default:
		return newImagesNormalController()
	}
}

func (p imagesPage) switchMode(to imagesMode) imagesPage {
	if p.ctrl != nil && p.ctrl.ID() == to {
		return p
	}

	oldCursor := p.imagesTable.Cursor()

	// Clear mode-related transient state.
	p = p.resetTransientState(false)

	p.ctrl = newImagesController(to)
	p.imagesTable = p.ctrl.NewTable()
	p = p.applyTableLayout()

	// Restore cursor within bounds.
	hi := 0
	if len(p.images) > 0 {
		hi = len(p.images) - 1
	}
	p.imagesTable.SetCursor(max(0, min(oldCursor, hi)))
	return p
}

func toIDSet(ids []string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}
