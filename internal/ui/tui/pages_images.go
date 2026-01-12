package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

const confirmDeleteImages ConfirmID = "images:delete"

// imagesModeID identifies Images UI modes.
// What: Normal / Delete
// Why: The spec defines behavior, keys and SEL column visibility by mode.
type imagesModeID int

const (
	imagesModeNormal imagesModeID = iota
	imagesModeDelete
)

// imagesCtx holds dependencies and environment for the Images page.
// Why: Keep modes pure-ish (no direct usecase calls) and make wiring explicit.
type imagesCtx struct {
	uc *usecase.ImageUsecase

	gkm globalKeyMap
	km  imagesKeyMap

	width  int
	height int
}

// imageState is the shared state for Images page and its modes.
//
// Why:
// - Keep all mutable state in one place (router-owned).
// - Modes read/update state, router performs side effects via actions.
type imagesState struct {
	items []engine.ImageSummary

	selected map[string]struct{}
	locked   map[string]struct{}
	busy     map[string]struct{}

	pendingDeleteIDs []string

	// op aggregates results for an in-flight delete session.
	op imagesDeleteOpState

	table table.Model
	mode  imagesMode
}

// imagesDeleteOpState tracks a single delete execution session.
//
// Why:
// - We want row-level busy feedback (per-item completion) without freezing the whole page.
// - Bubble Tea commands return a single Msg, so we stream per-item completion via a channel.
type imagesDeleteOpState struct {
	inFlight bool
	cancel   context.CancelFunc
	ch       <-chan imagesDeleteEvent

	total    int
	done     int
	ok       int
	skipped  int
	failed   int
	firstErr error
}

// imagesDeleteEvent is emitted per deleted images (success or failure).
type imagesDeleteEvent struct {
	id  string
	err error
}

type imagesDeleteStartedMsg struct {
	cancel  context.CancelFunc
	ch      <-chan imagesDeleteEvent
	total   int
	skipped int
}

type imagesDeleteStartFailedMsg struct{ err error }

type imagesDeleteEventReceivedMsg struct {
	ev imagesDeleteEvent
	ok bool
}

func startDeleteImagesCmd(imageUC *usecase.ImageUsecase, ids []string, skipped int) tea.Cmd {
	return func() tea.Msg {
		if imageUC == nil {
			return imagesDeleteStartFailedMsg{err: fmt.Errorf("image usecase is nil")}
		}
		if len(ids) == 0 {
			// Spec: do nothing when none selected.
			return imagesDeleteStartFailedMsg{err: fmt.Errorf("no ids to delete")}
		}

		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan imagesDeleteEvent, 64)

		go func() {
			defer close(ch)
			for _, id := range ids {
				// Best-effort: stop quickly if the page navigated away.
				select {
				case <-ctx.Done():
					return
				default:
				}

				err := imageUC.Delete(context.Background(), id)
				ev := imagesDeleteEvent{id: id, err: err}
				select {
				case ch <- ev:
				case <-ctx.Done():
					return
				}
			}
		}()

		return imagesDeleteStartedMsg{
			cancel:  cancel,
			ch:      ch,
			total:   len(ids),
			skipped: skipped,
		}
	}
}

func waitNextDeleteEventCmd(ch <-chan imagesDeleteEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return imagesDeleteStartFailedMsg{err: fmt.Errorf("delete event channel is nil")}
		}
		ev, ok := <-ch
		return imagesDeleteEventReceivedMsg{ev: ev, ok: ok}
	}
}

// imagesPage is a mode router for Images.
//
// Why mode router:
// - The spec is explicit about mode-based behavior (SEL column, enter=execute, key meanings).
// - Containers will gain more modes; this structure scales by adding submodels.

type imagesPage struct {
	ctx imagesCtx
	st  imagesState

	// loading is used only before the first list load.
	// Why: keep the UI responsive during refresh and while opts are running.
	loading bool
}

// compile-time interface check
var _ Page = imagesPage{}
var _ PageCloser = imagesPage{}

func newImagesPage(gkm globalKeyMap, imageUC *usecase.ImageUsecase) Page {
	p := imagesPage{
		ctx: imagesCtx{
			uc:  imageUC,
			gkm: gkm,
			km:  newImagesKeyMap(),
		},
		st: imagesState{
			selected: map[string]struct{}{},
			locked:   map[string]struct{}{},
			busy:     map[string]struct{}{},
			table: table.New(
				table.WithColumns(nil),
				table.WithRows(nil),
				table.WithFocused(true),
			),
			mode: newImagesNormalMode(),
		},
		loading: true,
	}
	return p
}

func (p imagesPage) Init() tea.Cmd {
	return tea.Batch(
		listImagesCmd(p.ctx.uc),
		listLockedImagesCmd(p.ctx.uc),
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

func (p imagesPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.ctx.width, p.ctx.height = msg.Width, msg.Height
		p.applyTableLayout()
		return p, nil

	case imagesLoadedMsg:
		p.loading = false
		p.st.items = []engine.ImageSummary(msg)
		p.rebuildTable()
		return p, nil

	case imagesLoadFailedMsg:
		p.loading = false
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case lockedImagesLoadedMsg:
		p.st.locked = map[string]struct{}(msg)
		p.rebuildTable()
		return p, nil

	case lockedImagesLoadFailedMsg:
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case confirmDialogResolvedMsg:
		if msg.id != confirmDeleteImages {
			return p, nil
		}

		ids := p.st.pendingDeleteIDs
		p.st.pendingDeleteIDs = nil

		if !msg.ok {
			return p, nil
		}

		// Revalidate at execution time:
		// - lock state may change between selection and confirm resolution.
		ids, skipped := p.deletableIDs(ids)
		if len(ids) == 0 {
			// Spec: do nothing when none selected / nothing deletable.
			return p, nil
		}

		// Start async delete:
		// - mark all target rows busy immediately ([*])
		// - clear selection per spec
		p.setBusy(ids)
		p.clearSelection()
		p.rebuildTable()
		return p, startDeleteImagesCmd(p.ctx.uc, ids, skipped)

	case imagesDeleteStartedMsg:
		p.st.op = imagesDeleteOpState{
			inFlight: true,
			cancel:   msg.cancel,
			ch:       msg.ch,
			total:    msg.total,
			skipped:  msg.skipped,
		}
		return p, waitNextDeleteEventCmd(p.st.op.ch)

	case imagesDeleteStartFailedMsg:
		// If start fails, clear busy to avoid stuck UI.
		p.clearBusy()
		p.rebuildTable()
		return p, openDialogCmd(dialogError, "Images", msg.err.Error())

	case imagesDeleteEventReceivedMsg:
		if !p.st.op.inFlight {
			return p, nil
		}
		if !msg.ok {
			// Channel closed -> operation finished.
			cmd := p.finishDeleteOp()
			return p, cmd
		}

		// One item finished: clear row-level busy and update counters.
		p.st.op.done++
		if msg.ev.err != nil {
			p.st.op.failed++
			if p.st.op.firstErr == nil {
				p.st.op.firstErr = msg.ev.err
			}
		} else {
			p.st.op.ok++
		}
		p.unsetBusy(msg.ev.id)
		p.rebuildTable()
		return p, waitNextDeleteEventCmd(p.st.op.ch)
	}

	// Delegate to mode for key interpretation and mode-specific actions.
	next, act, handled := p.st.mode.Update(p.ctx, &p.st, msg)
	if handled {
		return p, p.applyImagesAction(next, act)
	}

	// Let table handle cursor navigation etc.
	var cmd tea.Cmd
	p.st.table, cmd = p.st.table.Update(msg)
	return p, cmd
}

func (p imagesPage) View() string {
	if p.loading {
		return "Loading..."
	}

	var b strings.Builder
	b.WriteString(p.st.mode.Title() + "\n")
	b.WriteString(p.st.table.View())

	footer := renderHelpBlock(p.ctx.width, p.st.mode.FooterKeys(p.ctx, &p.st)...)
	if footer != "" {
		b.WriteString("\n" + footer + "\n")
	}
	return b.String()
}

func (p *imagesPage) applyTableLayout() {
	if p.ctx.width <= 0 || p.ctx.height <= 0 {
		return
	}
	tableHeight := max(p.ctx.height-tableNonBodyRows, 1)
	p.st.table.SetWidth(p.ctx.width)
	p.st.table.SetHeight(tableHeight)
	p.rebuildTable()
}

func (p *imagesPage) rebuildTable() {
	cols := p.st.mode.Columns(p.ctx.width)
	p.st.table.SetColumns(cols)
	p.st.table.SetRows(p.st.mode.Rows(&p.st))
}

func (p imagesPage) applyImagesAction(next imagesMode, act imagesAction) tea.Cmd {
	// Mode transition is owned by the router to keep it consistent.
	if next != nil && next.ID() != p.st.mode.ID() {
		// Spec/decision: selection is not preserved across mode changes.
		p.clearSelection()
		p.st.mode = next
		p.rebuildTable()
	}

	switch a := act.(type) {
	case actNone:
		return nil

	case actRefresh:
		// Decision: refresh clears selection (mode is kept).
		p.clearSelection()
		p.rebuildTable()
		return tea.Batch(listImagesCmd(p.ctx.uc), listLockedImagesCmd(p.ctx.uc))

	case actEnterDeleteMode:
		// Transition already handled above.
		return nil
	case actExitMode:
		// Transition already handled above.
		return nil

	case actToggleSelect:
		p.toggleSelected(a.id)
		p.rebuildTable()
		return nil

	case actExecuteDelete:
		ids := p.sortedSelectedIDs()
		if len(ids) == 0 {
			// Spec: do nothing when none selected
			return nil
		}
		// Confirm is required by spec (delete mode).
		return p.openDeleteConfirm(ids)

	case actRequestDelete:
		// Normal 'd' also requires confirm (decision).
		if len(a.ids) == 0 {
			return nil
		}
		return p.openDeleteConfirm(a.ids)

	default:
		return nil
	}
}

func (p imagesPage) openDeleteConfirm(ids []string) tea.Cmd {
	// Revalidate for better UX in normal mode:
	// - Without SEL column, users can't see lock state easily.
	ids, _ = p.deletableIDs(ids)
	if len(ids) == 0 {
		return openDialogCmd(dialogInfo, "Images", "No deletable images selected")
	}

	// Store pending ids so confirm resolution can start the operation.
	p.st.pendingDeleteIDs = ids
	body := fmt.Sprintf("Delete %d image(s)?", len(ids))
	return openConfirmDialogCmd(confirmDeleteImages, "Images", body)
}

func (p imagesPage) deletableIDs(in []string) (ids []string, skipped int) {
	for _, id := range in {
		if id == "" {
			continue
		}
		if p.isBusy(id) || p.isLocked(id) {
			skipped++
			continue
		}
		ids = append(ids, id)
	}
	return ids, skipped
}

func (p *imagesPage) finishDeleteOp() tea.Cmd {
	// Capture summary before resetting.
	ok := p.st.op.ok
	failed := p.st.op.failed
	skipped := p.st.op.skipped
	firstErr := p.st.op.firstErr

	// Reset op state and clear busy in case some IDs were not observed (safety)
	p.st.op = imagesDeleteOpState{}
	p.clearBusy()
	p.rebuildTable()

	// Spec: refresh after execution and show aggregated results.
	refresh := tea.Batch(listImagesCmd(p.ctx.uc), listLockedImagesCmd(p.ctx.uc))

	body := fmt.Sprintf("Success: %d\nSkipped: %d\nFailed: %d", ok, skipped, failed)
	if failed > 0 {
		if firstErr != nil {
			body = fmt.Sprintf("%s\n\n%s", body, firstErr.Error())
		}
		return tea.Sequence(refresh, openDialogCmd(dialogError, "Images", body))
	}
	return tea.Sequence(refresh, openDialogCmd(dialogInfo, "Images", body))
}

func (p imagesPage) Close() tea.Cmd {
	// Stop in-flight operation promptly on navigation.
	cancel := p.st.op.cancel
	return func() tea.Msg {
		if cancel != nil {
			cancel()
		}
		return nil
	}
}

// --- state helpers (centralized mutations; router owns state) ---

func (p *imagesPage) clearSelection() {
	p.st.selected = map[string]struct{}{}
	p.st.pendingDeleteIDs = nil
}

func (p imagesPage) isLocked(id string) bool {
	_, ok := p.st.locked[id]
	return ok
}

func (p imagesPage) isBusy(id string) bool {
	_, ok := p.st.busy[id]
	return ok
}

func (p *imagesPage) setBusy(ids []string) {
	p.st.busy = toIDSet(ids)
}

func (p *imagesPage) unsetBusy(id string) {
	delete(p.st.busy, id)
}

func (p *imagesPage) clearBusy() {
	p.st.busy = map[string]struct{}{}
}

func (p *imagesPage) toggleSelected(id string) {
	if _, ok := p.st.selected[id]; ok {
		delete(p.st.selected, id)
		return
	}
	p.st.selected[id] = struct{}{}
}

func (p imagesPage) sortedSelectedIDs() []string {
	out := make([]string, 0, len(p.st.selected))
	for id := range p.st.selected {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (st *imagesState) cursorImageID() (string, bool) {
	if len(st.items) == 0 {
		return "", false
	}
	i := st.table.Cursor()
	if i < 0 || i >= len(st.items) {
		return "", false
	}
	return st.items[i].ID, true
}

// 要移動候補
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
