package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/engine"
	"github.com/yuasalily/kontenaut/internal/engine/docker"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type model struct {
	uc *usecase.ContainerUsecase

	loading bool
	items   []engine.ContainerSummary
	err     error

	width  int
	height int

	table table.Model
}

// compile-time interface check
var _ tea.Model = model{}

func (m model) Init() tea.Cmd {
	return listContainersCmd(m.uc)
}

type containerMsg []engine.ContainerSummary
type errMsg struct{ err error }

func listContainersCmd(uc *usecase.ContainerUsecase) tea.Cmd {
	return func() tea.Msg {
		items, err := uc.List(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return containerMsg(items)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyTableLayout()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case containerMsg:
		m.loading = false
		m.items = []engine.ContainerSummary(msg)
		m.table.SetRows(rowsFromSummaries(m.items, m.table.Columns()))
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.loading {
		return "Loading..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\n(q to quit)\n", m.err)
	}

	var b strings.Builder
	b.WriteString("Containers\n")
	b.WriteString(m.table.View())

	b.WriteString("\n(q to quit)\n")
	return b.String()
}

func trunc(s string, w int) string {
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

func newTable() table.Model {
	cols := columnsForWidth(0)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
	return t
}

func (m *model) applyTableLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	tableHeight := m.height - 4
	if tableHeight < 1 {
		tableHeight = 1
	}

	m.table.SetWidth(m.width)
	m.table.SetHeight(tableHeight)

	cols := columnsForWidth(m.width)
	m.table.SetColumns(cols)
	if len(m.items) > 0 {
		m.table.SetRows(rowsFromSummaries(m.items, cols))
	}
}

func columnsForWidth(total int) []table.Column {
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

func rowsFromSummaries(items []engine.ContainerSummary, cols []table.Column) []table.Row {
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
		id := c.ID
		if len(id) > idW {
			id = id[:idW]
		}
		out = append(out, table.Row{
			trunc(id, idW),
			trunc(c.Image, imageW),
			trunc(c.Status, statusW),
			trunc(c.Name, nameW),
		})
	}
	return out
}

func main() {
	eng, err := docker.New()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = eng.Close() }()
	uc := usecase.NewContainerUsecase(eng)
	m := model{
		uc:      uc,
		loading: true,
		table:   newTable(),
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
