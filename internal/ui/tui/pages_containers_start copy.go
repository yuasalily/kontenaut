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

type restartSelectedContainersConfirmedMsg struct {
	ids []string
}

// containersRestartPage renders Containers restart mode as a separate page.
//
// Why:
// - Restart mode has different UI behaviors (selection/execute) and minimal shared state.
// - Keeping it as separate page reduces branching in the normal Containers page.
type containersRestartPage struct {
	containerUC *usecase.ContainerUsecase

	loading  bool
	restarting bool

	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model

	selected     map[string]struct{}
	nonRestartable map[string]struct{}
	busy         map[string]struct{}

	gkm globalKeyMap
	km  containersKeyMap

	sp spinner.Model
}

// compile-time interface check
var _ Page = containersRestartPage{}

func newContainersRestartPage(gkm globalKeyMap, containerUC *usecase.ContainerUsecase) Page {
	return containersRestartPage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTableRestart(),
		selected:        map[string]struct{}{},
		nonRestartable:    map[string]struct{}{},
		busy:            map[string]struct{}{},
		gkm:             gkm,
		km:              newContainersKeyMap(),
		sp:              newLoadingSpinner(),
	}
}

func (p containersRestartPage) Init() tea.Cmd {
	return tea.Batch(
		listContainersCmd(p.containerUC),
		p.sp.Tick,
	)
}

func (p containersRestartPage) Update(msg tea.Msg) (Page, tea.Cmd) {
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
		p.nonRestartable = nonRestartableContainerIDs(p.containers)
		p.containersTable.SetRows(rowsFromContainerSummariesRestart(
			p.containers,
			p.containersTable.Columns(),
			p.selected,
			p.nonRestartable,
			p.busy,
		))
		return p, nil

	case containersLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Containers", msg.err.Error())

	case restartSelectedContainersConfirmedMsg:
		if len(msg.ids) == 0 {
			return p, nil
		}
		p.restarting = true
		p.busy = toIDSet(msg.ids)
		return p, tea.Sequence(
			setGlobalBusyCmd(true),
			restartContainersCmd(p.containerUC, msg.ids),
		)

	case containersRestartedMsg:
		p.restarting = false
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.nonRestartable = map[string]struct{}{}

		var dlt tea.Cmd
		if msg.failed == 0 {
			dlt = openDialogCmd(dialogInfo, "Containers", fmt.Sprintf("Restarted %d container(s)", msg.restarted))
		} else {
			body := fmt.Sprintf("Restarted %d container(s). Failed %d container(s).", msg.restarted, msg.failed)
			if msg.firstErr != nil {
				body = fmt.Sprintf("%s\n\n%s", body, msg.firstErr.Error())
			}
			dlt = openDialogCmd(dialogError, "Containers", body)
		}
		// Stay in restart page after operation.
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

func (p containersRestartPage) View() string {
	if p.loading {
		return fmt.Sprintf("%s Loading...\n", p.sp.View())
	}
	if p.restarting {
		return "Restarting..."
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Containers [RESTART MODE]") + "\n")
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

func (p containersRestartPage) applyTableLayout() containersRestartPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.containersTable.SetWidth(p.width)
	p.containersTable.SetHeight(tableHeight)

	cols := columnsForContainersRestartWidth(p.width)
	p.containersTable.SetColumns(cols)
	if len(p.containers) > 0 {
		p.containersTable.SetRows(rowsFromContainerSummariesRestart(p.containers, cols, p.selected, p.nonRestartable, p.busy))
	}
	return p
}

func (p containersRestartPage) handleKey(msg tea.KeyMsg) (containersRestartPage, tea.Cmd, bool) {
	if p.loading || p.restarting {
		return p, nil, false
	}

	switch {
	case msg.Type == tea.KeyEnter:
		// no-op: restart mode does not use Enter.
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
		p.nonRestartable = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		c, ok := p.cursorContainer()
		if !ok {
			return p, nil, true
		}
		if _, ns := p.nonRestartable[c.ID]; ns {
			// Running containers are not selectable/restartable.
			return p, nil, true
		}

		// toggle selection
		if _, ok := p.selected[c.ID]; ok {
			delete(p.selected, c.ID)
		} else {
			p.selected[c.ID] = struct{}{}
		}
		p.containersTable.SetRows(rowsFromContainerSummariesRestart(p.containers, p.containersTable.Columns(), p.selected, p.nonRestartable, p.busy))
		return p, nil, true

	case key.Matches(msg, p.km.Execute):
		ids := make([]string, 0, len(p.selected))
		for id := range p.selected {
			if _, ns := p.nonRestartable[id]; ns {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			// Spec: do nothing when none selected.
			return p, nil, true
		}
		body := fmt.Sprintf("Restart %d container(s)?", len(ids))
		return p, openConfirmDialogCmd(
			"Containers",
			body,
			restartSelectedContainersConfirmedMsg{ids: ids},
			nil,
		), true

	case key.Matches(msg, p.km.Exit):
		// Exit restart mode -> back to normal Containers page.
		return p, func() tea.Msg { return navigateMsg{to: pageContainers} }, true
	}

	return p, nil, false
}

func (p containersRestartPage) cursorContainer() (engine.ContainerSummary, bool) {
	if len(p.containers) == 0 {
		return engine.ContainerSummary{}, false
	}
	i := p.containersTable.Cursor()
	if i < 0 || i >= len(p.containers) {
		return engine.ContainerSummary{}, false
	}
	return p.containers[i], true
}

func (p containersRestartPage) setIdle() containersRestartPage {
	p.loading = false
	p.restarting = false
	return p
}

func newContainersTableRestart() table.Model {
	cols := columnsForContainersRestartWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForContainersRestartWidth(total int) []table.Column {
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

func rowsFromContainerSummariesRestart(
	items []engine.ContainerSummary,
	cols []table.Column,
	selected map[string]struct{},
	nonRestartable map[string]struct{},
	busy map[string]struct{},
) []table.Row {
	selW := colWidth(cols, 0, 4)
	idW := colWidth(cols, 1, 12)
	imageW := colWidth(cols, 2, 20)
	statusW := colWidth(cols, 3, 18)
	nameW := colWidth(cols, 4, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		sel := selMark(c.ID, selected, nonRestartable, busy)

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
