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

const confirmDeleteContainers ConfirmID = "containers:delete"

type containersPage struct {
	containerUC *usecase.ContainerUsecase

	loading    bool
	deleting   bool
	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model

	selected map[string]struct{}
	locked   map[string]struct{}

	pendingDeleteIDs []string

	km containersKeyMap
}

// compile-time interface check
var _ Page = containersPage{}

func newContainersPage(containerUC *usecase.ContainerUsecase) Page {
	return containersPage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTable(),
		selected:        map[string]struct{}{},
		locked:          map[string]struct{}{},
		km:              newContainersKeyMap(),
	}
}

type containersDeletedMsg struct {
	deleted  int
	failed   int
	firstErr error
}

func deleteContainersCmd(containerUC *usecase.ContainerUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		deleted := 0
		failed := 0
		var firstErr error
		for _, id := range ids {
			if err := containerUC.Delete(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			deleted++
		}
		return containersDeletedMsg{
			deleted:  deleted,
			failed:   failed,
			firstErr: firstErr,
		}
	}
}

func (p containersPage) Init() tea.Cmd {
	return listContainersCmd(p.containerUC)
}

type containersLoadedMsg []engine.ContainerSummary
type containersLoadFailedMsg struct{ err error }

func listContainersCmd(containerUC *usecase.ContainerUsecase) tea.Cmd {
	return func() tea.Msg {
		items, err := containerUC.List(context.Background())
		if err != nil {
			return containersLoadFailedMsg{err: err}
		}
		return containersLoadedMsg(items)
	}
}

func (p containersPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		p = p.applyTableLayout()
		return p, nil

	case tea.KeyMsg:
		if msg.String() == "r" && !p.loading && !p.deleting {
			p.loading = true
			p.selected = map[string]struct{}{}
			p.pendingDeleteIDs = nil
			return p, p.Init()
		}

	case containersLoadedMsg:
		p.loading = false
		p.deleting = false
		p.containers = []engine.ContainerSummary(msg)
		p.locked = lockedContainerIDs(p.containers)
		p.containersTable.SetRows(rowsFromContainerSummaries(p.containers, p.containersTable.Columns(), p.selected, p.locked))
		return p, nil

	case containersLoadFailedMsg:
		p.loading = false
		p.deleting = false
		return p, openDialogCmd(dialogError, "Containers", msg.err.Error())

	case confirmDialogResolvedMsg:
		if msg.id != confirmDeleteContainers {
			return p, nil
		}
		ids := p.pendingDeleteIDs
		p.pendingDeleteIDs = nil
		if !msg.ok || len(ids) == 0 {
			return p, nil
		}
		p.deleting = true
		return p, deleteContainersCmd(p.containerUC, ids)

	case containersDeletedMsg:
		p.deleting = false
		p.loading = true
		p.selected = map[string]struct{}{}

		var dlt tea.Cmd
		if msg.failed == 0 {
			dlt = openDialogCmd(dialogInfo, "Containers", fmt.Sprintf("Deleted %d container(s)", msg.deleted))
		} else {
			body := fmt.Sprintf("Deleted %d container(s). Failed %d container(s).", msg.deleted, msg.failed)
			if msg.firstErr != nil {
				body = fmt.Sprintf("%s\n\n%s", body, msg.firstErr.Error())
			}
			dlt = openDialogCmd(dialogError, "Containers", body)
		}
		return p, tea.Sequence(listContainersCmd(p.containerUC), dlt)
	}

	if km, ok := msg.(tea.KeyMsg); ok && !p.loading && !p.deleting {
		switch km.String() {
		case " ", "space":
			id, ok := p.cursorContainerID()
			if !ok {
				return p, nil
			}
			if p.isLocked(id) {
				return p, nil
			}
			p.toggleSelected(id)
			p.containersTable.SetRows(rowsFromContainerSummaries(p.containers, p.containersTable.Columns(), p.selected, p.locked))
			return p, nil

		case "d":
			ids := p.selectedDeletableIDs()
			if len(ids) == 0 {
				return p, openDialogCmd(dialogInfo, "Containers", "No containers selected")
			}
			p.pendingDeleteIDs = ids
			body := fmt.Sprintf("Delete %d container(s)?", len(ids))
			return p, openConfirmDialogCmd(confirmDeleteContainers, "Containers", body)

		case "enter":
			c, ok := p.cursorContainer()
			if !ok {
				return p, nil
			}
			name := c.Name
			if name == "" {
				name = "Unnamed"
			}
			return p, func() tea.Msg {
				return openLogsMsg{id: c.ID, name: name}
			}
		}
	}

	var cmd tea.Cmd
	p.containersTable, cmd = p.containersTable.Update(msg)
	return p, cmd
}

func (p containersPage) View() string {
	if p.loading {
		return "Loading..."
	}
	if p.deleting {
		return "Deleting..."
	}

	var b strings.Builder
	b.WriteString("Containers\n")
	b.WriteString(p.containersTable.View())

	b.WriteString("\n(space: select, d: delete, enter: logs, r: refresh, q to quit)\n")
	return b.String()
}

func (p containersPage) handleKey(msg tea.KeyMsg) (containersPage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	switch {
	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.selected = map[string]struct{}{}
		p.pendingDeleteIDs = nil
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		id, ok := p.cursorContainerID()
		if !ok {
			return p, nil, true
		}
		if p.isLocked(id) {
			return p, nil, true
		}
		p.toggleSelected(id)
		p.containersTable.SetRows(rowsFromContainerSummaries(p.containers, p.containersTable.Columns(), p.selected, p.locked))
		return p, nil, true

	case key.Matches(msg, p.km.Delete):
		ids := p.selectedDeletableIDs()
		if len(ids) == 0 {
			return p, openDialogCmd(dialogInfo, "Containers", "No containers selected"), true
		}
		p.pendingDeleteIDs = ids
		body := fmt.Sprintf("Delete %d container(s)?", len(ids))
		return p, openConfirmDialogCmd(confirmDeleteContainers, "Containers", body), true

	case key.Matches(msg, p.km.Logs):
		c, ok := p.cursorContainer()
		if !ok {
			return p, nil, true
		}
		name := c.Name
		if name == "" {
			name = "Unnamed"
		}
		return p, func() tea.Msg {
			return openLogsMsg{id: c.ID, name: name}
		}, true
	}

	return p, nil, false
}

func (p containersPage) applyTableLayout() containersPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-4, 1)

	p.containersTable.SetWidth(p.width)
	p.containersTable.SetHeight(tableHeight)

	cols := columnsForContainersWidth(p.width)
	p.containersTable.SetColumns(cols)
	if len(p.containers) > 0 {
		p.containersTable.SetRows(rowsFromContainerSummaries(p.containers, cols, p.selected, p.locked))
	}
	return p
}

func newContainersTable() table.Model {
	cols := columnsForContainersWidth(0)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
	return t
}

func columnsForContainersWidth(total int) []table.Column {
	const (
		selW    = 4
		idW     = 12
		imageW  = 20
		statusW = 18
	)

	nameW := 20
	if total > 0 {
		rest := total - (selW + idW + imageW + statusW) - 8
		if rest > nameW {
			nameW = rest
		}
	}

	return []table.Column{
		{Title: "SEL", Width: selW},
		{Title: "ID", Width: idW},
		{Title: "IMAGE", Width: imageW},
		{Title: "STATUS", Width: statusW},
		{Title: "NAME", Width: nameW},
	}
}

func rowsFromContainerSummaries(items []engine.ContainerSummary, cols []table.Column, selected map[string]struct{}, locked map[string]struct{}) []table.Row {
	getW := func(i int, fallback int) int {
		if i < 0 || i >= len(cols) {
			return fallback
		}
		return cols[i].Width
	}

	selW := getW(0, 4)
	idW := getW(1, 12)
	imageW := getW(2, 20)
	statusW := getW(3, 18)
	nameW := getW(4, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		sel := "[ ]"
		if _, ok := locked[c.ID]; ok {
			sel = "[#]"
		} else if _, ok := selected[c.ID]; ok {
			sel = "[x]"
		}
		out = append(out, table.Row{
			truncContainer(sel, selW),
			truncContainer(c.ID, idW),
			truncContainer(c.Image, imageW),
			truncContainer(c.Status, statusW),
			truncContainer(c.Name, nameW),
		})
	}
	return out
}

func (p containersPage) cursorContainer() (engine.ContainerSummary, bool) {
	if len(p.containers) == 0 {
		return engine.ContainerSummary{}, false
	}
	i := p.containersTable.Cursor()
	if i < 0 || i >= len(p.containers) {
		return engine.ContainerSummary{}, false
	}
	return p.containers[i], true
}

func (p containersPage) cursorContainerID() (string, bool) {
	c, ok := p.cursorContainer()
	if !ok {
		return "", false
	}
	return c.ID, true
}

func lockedContainerIDs(items []engine.ContainerSummary) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, c := range items {
		if isContainerRunning(c.State) {
			out[c.ID] = struct{}{}
		}
	}
	return out
}

func isContainerRunning(state string) bool {
	// Docker "State" is a machine-readable string like:
	// "running", "exited", ...
	// We treat "running" as locked (not deletable without -f)
	return state == "running"
}

func (p containersPage) isLocked(id string) bool {
	_, ok := p.locked[id]
	return ok
}

func (p *containersPage) toggleSelected(id string) {
	if _, ok := p.selected[id]; ok {
		delete(p.selected, id)
		return
	}
	p.selected[id] = struct{}{}
}

func (p containersPage) selectedDeletableIDs() []string {
	out := make([]string, 0, len(p.selected))
	for id := range p.selected {
		if _, ok := p.locked[id]; ok {
			continue
		}
		out = append(out, id)
	}
	return out
}

func truncContainer(s string, w int) string {
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
