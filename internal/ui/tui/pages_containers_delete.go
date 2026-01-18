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

type deleteSelectedContainersConfirmedMsg struct {
	ids []string
}

// containersDeletePage renders Containers delete mode as a separate page.
//
// Why:
// - Delete mode has different UI behaviors (selection/execute) and minimal shared state.
// - Keeping it as separate page reduces branching in the normal Containers page.
type containersDeletePage struct {
	containerUC *usecase.ContainerUsecase

	loading  bool
	deleting bool

	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model

	selected map[string]struct{}
	locked   map[string]struct{}
	busy     map[string]struct{}

	gkm globalKeyMap
	km  containersKeyMap
}

// compile-time interface check
var _ Page = containersDeletePage{}

func newContainersDeletePage(gkm globalKeyMap, containerUC *usecase.ContainerUsecase) Page {
	return containersDeletePage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTableDelete(),
		selected:        map[string]struct{}{},
		locked:          map[string]struct{}{},
		busy:            map[string]struct{}{},
		gkm:             gkm,
		km:              newContainersKeyMap(),
	}
}

func (p containersDeletePage) Init() tea.Cmd {
	return listContainersCmd(p.containerUC)
}

func (p containersDeletePage) Update(msg tea.Msg) (Page, tea.Cmd) {
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

	case containersLoadedMsg:
		p = p.setIdle()
		p.containers = []engine.ContainerSummary(msg)
		p.locked = lockedContainerIDs(p.containers)
		p.containersTable.SetRows(rowsFromContainerSummariesDelete(
			p.containers,
			p.containersTable.Columns(),
			p.selected,
			p.locked,
			p.busy,
		))
		return p, nil

	case containersLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Containers", msg.err.Error())

	case deleteSelectedContainersConfirmedMsg:
		if len(msg.ids) == 0 {
			return p, nil
		}
		p.deleting = true
		p.busy = toIDSet(msg.ids)
		return p, deleteContainersCmd(p.containerUC, msg.ids)

	case containersDeletedMsg:
		p.deleting = false
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.locked = map[string]struct{}{}

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
		// Stay in delete page after operation.
		return p, tea.Sequence(listContainersCmd(p.containerUC), dlt)
	}

	var cmd tea.Cmd
	p.containersTable, cmd = p.containersTable.Update(msg)
	return p, cmd
}

func (p containersDeletePage) View() string {
	if p.loading {
		return "Loading..."
	}
	if p.deleting {
		return "Deleting..."
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Containers [DELETE MODE]") + "\n")
	b.WriteString(p.containersTable.View())

	footer := renderHelpBlock(
		p.width,
		p.containersTable.KeyMap.LineUp,
		p.containersTable.KeyMap.LineDown,
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

func (p containersDeletePage) applyTableLayout() containersDeletePage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.containersTable.SetWidth(p.width)
	p.containersTable.SetHeight(tableHeight)

	cols := columnsForContainersDeleteWidth(p.width)
	p.containersTable.SetColumns(cols)
	if len(p.containers) > 0 {
		p.containersTable.SetRows(rowsFromContainerSummariesDelete(p.containers, cols, p.selected, p.locked, p.busy))
	}
	return p
}

func (p containersDeletePage) handleKey(msg tea.KeyMsg) (containersDeletePage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	switch {
	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.locked = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		c, ok := p.cursorContainer()
		if !ok {
			return p, nil, true
		}
		if _, locked := p.locked[c.ID]; locked {
			// Running containers are not selectable/deletable.
			return p, nil, true
		}

		// toggle selection
		if _, ok := p.selected[c.ID]; ok {
			delete(p.selected, c.ID)
		} else {
			p.selected[c.ID] = struct{}{}
		}
		p.containersTable.SetRows(rowsFromContainerSummariesDelete(p.containers, p.containersTable.Columns(), p.selected, p.locked, p.busy))
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
		body := fmt.Sprintf("Delete %d container(s)?", len(ids))
		return p, openConfirmDialogCmd(
			"Containers",
			body,
			deleteSelectedContainersConfirmedMsg{ids: ids},
			nil,
		), true

	case key.Matches(msg, p.km.Exit):
		// Exit delete mode -> back to normal Containers page.
		return p, func() tea.Msg { return navigateMsg{to: pageContainers} }, true
	}

	return p, nil, false
}

func (p containersDeletePage) cursorContainer() (engine.ContainerSummary, bool) {
	if len(p.containers) == 0 {
		return engine.ContainerSummary{}, false
	}
	i := p.containersTable.Cursor()
	if i < 0 || i >= len(p.containers) {
		return engine.ContainerSummary{}, false
	}
	return p.containers[i], true
}

func (p containersDeletePage) setIdle() containersDeletePage {
	p.loading = false
	p.deleting = false
	return p
}

func newContainersTableDelete() table.Model {
	cols := columnsForContainersDeleteWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForContainersDeleteWidth(total int) []table.Column {
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

func rowsFromContainerSummariesDelete(
	items []engine.ContainerSummary,
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
	imageW := getW(2, 20)
	statusW := getW(3, 18)
	nameW := getW(4, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		sel := "[ ]"
		if _, ok := busy[c.ID]; ok {
			sel = "[*]"
		} else if _, ok := locked[c.ID]; ok {
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
