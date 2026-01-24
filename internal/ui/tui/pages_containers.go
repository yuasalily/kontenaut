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

type deleteSingleContainerConfirmedMsg struct{ id string }

// containersPage renders the Containers list (normal mode).
//
// Why:
// - Keep this page simple: list containers and support single-item actions.
// - Bulk deletion is implemented as a separate delete-mode page (like Images).
type containersPage struct {
	containerUC *usecase.ContainerUsecase

	loading    bool
	deleting   bool
	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model

	locked map[string]struct{}

	gkm globalKeyMap
	km  containersKeyMap
}

// compile-time interface check
var _ Page = containersPage{}

func newContainersPage(gkm globalKeyMap, containerUC *usecase.ContainerUsecase) Page {
	return containersPage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTableNormal(),
		locked:          map[string]struct{}{},
		gkm:             gkm,
		km:              newContainersKeyMap(),
	}
}

func (p containersPage) Init() tea.Cmd {
	return listContainersCmd(p.containerUC)
}

func (p containersPage) Update(msg tea.Msg) (Page, tea.Cmd) {
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
		p.containersTable.SetRows(rowsFromContainerSummariesNormal(p.containers, p.containersTable.Columns()))
		return p, nil

	case containersLoadFailedMsg:
		p = p.setIdle()
		return p, openDialogCmd(dialogError, "Containers", msg.err.Error())

	case deleteSingleContainerConfirmedMsg:
		if msg.id == "" {
			return p, nil
		}
		p.deleting = true
		return p, tea.Sequence(
			setGlobalBusyCmd(true),
			deleteContainersCmd(p.containerUC, []string{msg.id}),
		)

	case containersDeletedMsg:
		p.deleting = false
		p.loading = true
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

	footer := renderHelpBlock(
		p.width,
		p.containersTable.KeyMap.LineUp,
		p.containersTable.KeyMap.LineDown,
		p.km.DeleteSingle,
		p.km.EnterDeleteMode,
		p.km.Logs,
		p.km.Refresh,
		p.gkm.Quit,
	)
	if footer != "" {
		b.WriteString("\n" + footer + "\n")
	}
	return b.String()
}

func (p containersPage) handleKey(msg tea.KeyMsg) (containersPage, tea.Cmd, bool) {
	if p.loading || p.deleting {
		return p, nil, false
	}

	switch {
	case msg.Type == tea.KeyEnter:
		// no-op: Enter is intentionally unused in Containers normal mode.
		//
		// Why:
		// - Enter is reserved for "confirm/execute" semantics in the app (dialogs and destructive actions).
		// - Explicitly swallowing Enter prevents accidental behavior changes from the table component
		//   (some versions may react to Enter) and keeps input handling predictable.
		return p, nil, true

	case key.Matches(msg, p.km.Refresh):
		p.loading = true
		p.locked = map[string]struct{}{}
		return p, p.Init(), true

	case key.Matches(msg, p.km.EnterDeleteMode):
		return p, func() tea.Msg { return openContainersDeleteMsg{} }, true

	case key.Matches(msg, p.km.DeleteSingle):
		c, ok := p.cursorContainer()
		if !ok {
			return p, nil, true
		}
		if p.isLocked(c.ID) {
			return p, openDialogCmd(dialogInfo, "Containers", "this container is running and cannot be deleted."), true
		}
		return p, openConfirmDialogCmd(
			"Containers",
			"Delete 1 container?",
			deleteSingleContainerConfirmedMsg{id: c.ID},
			nil,
		), true

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
	tableHeight := max(p.height-tableNonBodyRows, 1)

	p.containersTable.SetWidth(p.width)
	p.containersTable.SetHeight(tableHeight)

	// Why dynamic columns:
	// - Terminal width varies widely; allocate remaining space to NAME for readability.
	cols := columnsForContainersNormalWidth(p.width)
	p.containersTable.SetColumns(cols)
	if len(p.containers) > 0 {
		p.containersTable.SetRows(rowsFromContainerSummariesNormal(p.containers, cols))
	}
	return p
}

func newContainersTableNormal() table.Model {
	cols := columnsForContainersNormalWidth(0)
	return table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
}

func columnsForContainersNormalWidth(total int) []table.Column {
	const (
		idW     = 12
		imageW  = 20
		statusW = 18
	)

	nameW := 20
	if total > 0 {
		rest := total - (idW + imageW + statusW) - 6
		if rest > nameW {
			nameW = rest
		}
	}

	return []table.Column{
		{Title: "ID", Width: idW},
		{Title: "IMAGE", Width: imageW},
		{Title: "STATUS", Width: statusW},
		{Title: "NAME", Width: nameW},
	}
}

func rowsFromContainerSummariesNormal(items []engine.ContainerSummary, cols []table.Column) []table.Row {
	idW := colWidth(cols, 0, 12)
	imageW := colWidth(cols, 1, 20)
	statusW := colWidth(cols, 2, 18)
	nameW := colWidth(cols, 3, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		out = append(out, table.Row{
			truncText(c.ID, idW),
			truncText(c.Image, imageW),
			truncText(c.Status, statusW),
			truncText(c.Name, nameW),
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

func (p containersPage) isLocked(id string) bool {
	_, ok := p.locked[id]
	return ok
}

func (p containersPage) setIdle() containersPage {
	p.loading = false
	p.deleting = false
	return p
}
