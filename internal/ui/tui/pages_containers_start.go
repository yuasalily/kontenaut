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

type startSelectedContainersConfirmedMsg struct {
	ids []string
}

// containersStartPage renders Containers start mode as a separate page.
//
// Why:
// - Start mode has different UI behaviors (selection/execute) and minimal shared state.
// - Keeping it as separate page reduces branching in the normal Containers page.
type containersStartPage struct {
	containerUC *usecase.ContainerUsecase

	loading  bool
	starting bool

	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model

	selected     map[string]struct{}
	nonStartable map[string]struct{}
	busy         map[string]struct{}

	gkm globalKeyMap
	km  containersKeyMap

	sp spinner.Model
}

// compile-time interface check
var _ Page = containersStartPage{}

func newContainersStartPage(gkm globalKeyMap, containerUC *usecase.ContainerUsecase) Page {
	return containersStartPage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTableStart(),
		selected:        map[string]struct{}{},
		nonStartable:    map[string]struct{}{},
		busy:            map[string]struct{}{},
		gkm:             gkm,
		km:              newContainersKeyMap(),
		sp:              newLoadingSpinner(),
	}
}

func (p containersStartPage) Init() tea.Cmd {
	return tea.Batch(
		listContainersCmd(p.containerUC),
		p.sp.Tick,
	)
}

func (p containersStartPage) Update(msg tea.Msg) (Page, tea.Cmd) {
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
		p.nonStartable = nonStartableContainerIDs(p.containers)
		p.containersTable.SetRows(rowsFromContainerSummariesStart(
			p.containers,
			p.containersTable.Columns(),
			p.selected,
			p.nonStartable,
			p.busy,
		))
		return p, nil

	case containersLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Containers", msg.err.Error())

	case startSelectedContainersConfirmedMsg:
		if len(msg.ids) == 0 {
			return p, nil
		}
		p.starting = true
		p.busy = toIDSet(msg.ids)
		return p, tea.Sequence(
			setGlobalBusyCmd(true),
			startContainersCmd(p.containerUC, msg.ids),
		)

	case containersStartedMsg:
		p.starting = false
		p.loading = true
		p.selected = map[string]struct{}{}
		p.busy = map[string]struct{}{}
		p.nonStartable = map[string]struct{}{}

		var dlt tea.Cmd
		if msg.failed == 0 {
			dlt = openDialogCmd(dialogInfo, "Containers", fmt.Sprintf("Started %d container(s)", msg.started))
		} else {
			body := fmt.Sprintf("Started %d container(s). Failed %d container(s).", msg.started, msg.failed)
			if msg.firstErr != nil {
				body = fmt.Sprintf("%s\n\n%s", body, msg.firstErr.Error())
			}
			dlt = openDialogCmd(dialogError, "Containers", body)
		}
		// Stay in start page after operation.
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

func (p containersStartPage) View() string {
	if p.loading {
		return fmt.Sprintf("%s Loading...\n", p.sp.View())
	}
	if p.starting {
		return "Starting..."
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Containers [START MODE]") + "\n")
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

func (p containersStartPage) applyTableLayout() containersStartPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.containersTable.SetWidth(p.width)
	p.containersTable.SetHeight(tableHeight)

	cols := columnsForContainersStartWidth(p.width)
	p.containersTable.SetColumns(cols)
	if len(p.containers) > 0 {
		p.containersTable.SetRows(rowsFromContainerSummariesStart(p.containers, cols, p.selected, p.nonStartable, p.busy))
	}
	return p
}

func (p containersStartPage) handleKey(msg tea.KeyMsg) (containersStartPage, tea.Cmd, bool) {
	if p.loading || p.starting {
		return p, nil, false
	}

	switch {
	case msg.Type == tea.KeyEnter:
		// no-op: start mode does not use Enter.
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
		p.nonStartable = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.Select):
		c, ok := p.cursorContainer()
		if !ok {
			return p, nil, true
		}
		if _, ns := p.nonStartable[c.ID]; ns {
			// Running containers are not selectable/startable.
			return p, nil, true
		}

		// toggle selection
		if _, ok := p.selected[c.ID]; ok {
			delete(p.selected, c.ID)
		} else {
			p.selected[c.ID] = struct{}{}
		}
		p.containersTable.SetRows(rowsFromContainerSummariesStart(p.containers, p.containersTable.Columns(), p.selected, p.nonStartable, p.busy))
		return p, nil, true

	case key.Matches(msg, p.km.Execute):
		ids := make([]string, 0, len(p.selected))
		for id := range p.selected {
			if _, ns := p.nonStartable[id]; ns {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			// Spec: do nothing when none selected.
			return p, nil, true
		}
		body := fmt.Sprintf("Start %d container(s)?", len(ids))
		return p, openConfirmDialogCmd(
			"Containers",
			body,
			startSelectedContainersConfirmedMsg{ids: ids},
			nil,
		), true

	case key.Matches(msg, p.km.Exit):
		// Exit start mode -> back to normal Containers page.
		return p, func() tea.Msg { return navigateMsg{to: pageContainers} }, true
	}

	return p, nil, false
}

func (p containersStartPage) cursorContainer() (engine.ContainerSummary, bool) {
	if len(p.containers) == 0 {
		return engine.ContainerSummary{}, false
	}
	i := p.containersTable.Cursor()
	if i < 0 || i >= len(p.containers) {
		return engine.ContainerSummary{}, false
	}
	return p.containers[i], true
}

func (p containersStartPage) setIdle() containersStartPage {
	p.loading = false
	p.starting = false
	return p
}

func newContainersTableStart() table.Model {
	cols := columnsForContainersStartWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForContainersStartWidth(total int) []table.Column {
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

func rowsFromContainerSummariesStart(
	items []engine.ContainerSummary,
	cols []table.Column,
	selected map[string]struct{},
	nonStartable map[string]struct{},
	busy map[string]struct{},
) []table.Row {
	selW := colWidth(cols, 0, 4)
	idW := colWidth(cols, 1, 12)
	imageW := colWidth(cols, 2, 20)
	statusW := colWidth(cols, 3, 18)
	nameW := colWidth(cols, 4, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		sel := selMark(c.ID, selected, nonStartable, busy)

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
