package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type containersPage struct {
	containerUC *usecase.ContainerUsecase

	loading    bool
	containers []engine.ContainerSummary

	width  int
	height int

	containersTable table.Model
}

// compile-time interface check
var _ Page = containersPage{}

func newContainersPage(containerUC *usecase.ContainerUsecase) Page {
	return containersPage{
		containerUC:     containerUC,
		loading:         true,
		containersTable: newContainersTable(),
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

	case containersLoadedMsg:
		p.loading = false
		p.containers = []engine.ContainerSummary(msg)
		p.containersTable.SetRows(rowsFromContainerSummaries(p.containers, p.containersTable.Columns()))

	case containersLoadFailedMsg:
		p.loading = false
		return p, showDialogCmd(dialogError, "Containers", msg.err.Error())
	}

	var cmd tea.Cmd
	p.containersTable, cmd = p.containersTable.Update(msg)
	return p, cmd
}

func (p containersPage) View() string {
	if p.loading {
		return "Loading..."
	}

	var b strings.Builder
	b.WriteString("Containers\n")
	b.WriteString(p.containersTable.View())

	b.WriteString("\n(q to quit)\n")
	return b.String()
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
		p.containersTable.SetRows(rowsFromContainerSummaries(p.containers, cols))
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

func rowsFromContainerSummaries(items []engine.ContainerSummary, cols []table.Column) []table.Row {
	getW := func(i int, fallback int) int {
		if i < 0 || i >= len(cols) {
			return fallback
		}
		return cols[i].Width
	}

	idW := getW(0, 12)
	imageW := getW(1, 20)
	statusW := getW(2, 18)
	nameW := getW(3, 20)

	out := make([]table.Row, 0, len(items))
	for _, c := range items {
		out = append(out, table.Row{
			truncContainer(c.ID, idW),
			truncContainer(c.Image, imageW),
			truncContainer(c.Status, statusW),
			truncContainer(c.Name, nameW),
		})
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
