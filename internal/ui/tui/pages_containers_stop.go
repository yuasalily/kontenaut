package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type stopSelectedContainersConfirmedMsg struct {
	ids []string
}

// containersStopPage renders Containers stop mode as a separate page.
//
// Why:
// - Stop mode has different UI behaviors (selection/execute) and minimal shared state.
// - Keeping it as separate page reduces branching in the normal Containers page.
type containersStopPage struct {
	containerUC *usecase.ContainerUsecase

	loading  bool
	stopping bool

	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model

	selected     map[string]struct{}
	nonStoppable map[string]struct{}
	busy         map[string]struct{}

	gkm globalKeyMap
	km  containersKeyMap

	sp spinner.Model
}

// compile-time interface check
var _ Page = containersStopPage{}

func newContainersStopPage(gkm globalKeyMap, containerUC *usecase.ContainerUsecase) Page {
	return containersStopPage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTableStop(),
		selected:        map[string]struct{}{},
		nonStoppable:    map[string]struct{}{},
		busy:            map[string]struct{}{},
		gkm:             gkm,
		km:              newContainersKeyMap(),
		sp:              newLoadingSpinner(),
	}
}

func (p containersStopPage) Init() tea.Cmd {
	return tea.Batch(
		listContainersCmd(p.containerUC),
		p.sp.Tick,
	)
}

func (p containersStopPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		p.sp, cmd = p.sp.Update(msg)
		if p.loading {
			return p, cmd
		}
		return p, nil

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
		p.nonStoppable = nonStoppableContainerIDs(p.containers)
		p.containersTable.SetRows(rowsFromContainerSummariesStop(
			p.containers,
			p.containersTable.Columns(),
			p.selected,
			p.nonStoppable,
			p.busy,
		))
		return p, nil

	case containersLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Containers", msg.err.Error())

	case stopSelectedContainersConfirmedMsg:
		if len(msg.ids) == 0 {
			return p, nil
		}
		p.stopping = true
		p.busy = toIDSet(msg.ids)
		return p, tea.Sequence(
			setGlobalBusyCmd(true),
			stopContainersCmd(p.containerUC, msg.ids),
		)

	case containersStoppedMsg:
		p.stopping = false
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.nonStoppable = map[string]struct{}{}

		var dlt tea.Cmd
		if msg.failed == 0 {
			dlt = openDialogCmd(dialogInfo, "Containers", fmt.Sprintf("Stopped %d container(s)", msg.stopped))
		} else {
			body := fmt.Sprintf("Stopped %d container(s). Failed %d container(s).", msg.stopped, msg.failed)
			if msg.firstErr != nil {
				body = fmt.Sprintf("%s\n\n%s", body, msg.firstErr.Error())
			}
			dlt = openDialogCmd(dialogError, "Containers", body)
		}
		// Stay in stop page after operation.
		return p, tea.Sequence(
			setGlobalBusyCmd(false),
			listContainersCmd(p.containerUC),
			dlt,
		)
	}

	var cmd tea.Cmd
	p.containersTable, cmd = p.containersTable.Update(msg)
	return p, cmd
}

func (p containersStopPage) View() string {
	if p.loading {
		return fmt.Sprintf("%s Loading...\n", p.sp.View())
	}
	if p.stopping {
		return "Stopping..."
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Containers [STOP MODE]") + "\n")
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

func (p containersStopPage) applyTableLayout() containersStopPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.containersTable.SetWidth(p.width)
	p.containersTable.SetHeight(tableHeight)

	cols := columnsForContainersStopWidth(p.width)
	p.containersTable.SetColumns(cols)
	if len(p.containers) > 0 {
		p.containersTable.SetRows(rowsFromContainerSummariesStop(p.containers, cols, p.selected, p.nonStoppable, p.busy))
	}
	return p
}

func (p containersStopPage) handleKey(msg tea.KeyMsg) (containersStopPage, tea.Cmd, bool) {
	if p.loading || p.stopping {
		return p, nil, false
	}

	switch {
	case msg.Type == tea.KeyEnter:
		// no-op: stop mode does not use Enter.
		//
		// Why:
		// - Execute is bound explicitly (e.g. "x") to reduce accidental execution.
		// - Explicitly swallowing Enter prevents the table component from reacting to it and
		// keeps destructive operations behind a deliberate key press.
		return p, nil, true

	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.nonStoppable = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		c, ok := p.cursorContainer()
		if !ok {
			return p, nil, true
		}
		if _, ns := p.nonStoppable[c.ID]; ns {
			// Running containers are not selectable/stoppable.
			return p, nil, true
		}

		// toggle selection
		if _, ok := p.selected[c.ID]; ok {
			delete(p.selected, c.ID)
		} else {
			p.selected[c.ID] = struct{}{}
		}
		p.containersTable.SetRows(rowsFromContainerSummariesStop(p.containers, p.containersTable.Columns(), p.selected, p.nonStoppable, p.busy))
		return p, nil, true

	case key.Matches(msg, p.km.Execute):
		ids := make([]string, 0, len(p.selected))
		for id := range p.selected {
			if _, ns := p.nonStoppable[id]; ns {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			// Spec: do nothing when none selected.
			return p, nil, true
		}
		body := fmt.Sprintf("Stop %d container(s)?", len(ids))
		return p, openConfirmDialogCmd(
			"Containers",
			body,
			stopSelectedContainersConfirmedMsg{ids: ids},
			nil,
		), true

	case key.Matches(msg, p.km.Exit):
		// Exit stop mode -> back to normal Containers page.
		return p, func() tea.Msg { return navigateMsg{to: pageContainers} }, true
	}

	return p, nil, false
}

func (p containersStopPage) cursorContainer() (engine.ContainerSummary, bool) {
	if len(p.containers) == 0 {
		return engine.ContainerSummary{}, false
	}
	i := p.containersTable.Cursor()
	if i < 0 || i >= len(p.containers) {
		return engine.ContainerSummary{}, false
	}
	return p.containers[i], true
}

func (p containersStopPage) setIdle() containersStopPage {
	p.loading = false
	p.stopping = false
	return p
}

func newContainersTableStop() table.Model {
	cols := columnsForContainersStopWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForContainersStopWidth(total int) []table.Column {
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

func rowsFromContainerSummariesStop(
	items []engine.ContainerSummary,
	cols []table.Column,
	selected map[string]struct{},
	nonStoppable map[string]struct{},
	busy map[string]struct{},
) []table.Row {
	selW := colWidth(cols, 0, 4)
	idW := colWidth(cols, 1, 12)
	imageW := colWidth(cols, 2, 20)
	statusW := colWidth(cols, 3, 18)
	nameW := colWidth(cols, 4, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		sel := selMark(c.ID, selected, nonStoppable, busy)

		out = append(out, table.Row{
			truncText(sel, selW),
			truncText(c.ID, idW),
			truncText(c.Image, imageW),
			truncText(c.Status, statusW),
			truncText(c.Name, nameW),
		})
	}
	return out
}
