package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type logsPage struct {
	containerUC *usecase.ContainerUsecase

	containerID   string
	containerName string

	loading bool
	lines   []string

	width  int
	height int
}

// compile-time interface check
var _ Page = logsPage{}

func newLogsPage(containerUC *usecase.ContainerUsecase, containerID, containerName string) Page {
	return logsPage{
		containerUC:   containerUC,
		containerID:   containerID,
		containerName: containerName,
		loading:       true,
		lines:         nil,
		width:         0,
		height:        0,
	}
}

type logsLoadedMsg struct {
	lines []string
}

type logsLoadFailedMsg struct{ err error }

func loadLogsCmd(containerUC *usecase.ContainerUsecase, containerID string, tail int) tea.Cmd {
	return func() tea.Msg {
		lines, err := containerUC.Logs(context.Background(), containerID, tail)
		if err != nil {
			return logsLoadFailedMsg{err: err}
		}
		return logsLoadedMsg{lines: lines}
	}
}

func (p logsPage) Init() tea.Cmd {
	return loadLogsCmd(p.containerUC, p.containerID, 200)
}

func (p logsPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			return p, func() tea.Msg { return navigateMsg{to: pageContainers} }
		}

	case logsLoadedMsg:
		p.loading = false
		p.lines = msg.lines
		return p, nil

	case logsLoadFailedMsg:
		p.loading = false
		p.lines = nil
		return p, openDialogCmd(dialogError, "Logs", msg.err.Error())
	}

	return p, nil
}

func (p logsPage) View() string {
	title := "Logs"
	if p.containerName != "" {
		title = fmt.Sprintf("Logs: %s", p.containerName)
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	if p.loading {
		b.WriteString("Loading...\n")
		b.WriteString("\n(esc/b : back, q: quit)\n")
		return b.String()
	}

	if len(p.lines) == 0 {
		b.WriteString("<no logs>\n")
		b.WriteString("\n(esc/b : back, q: quit)\n")
		return b.String()
	}

	for _, line := range p.lines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n(esc/b : back, q: quit)\n")
	return b.String()
}
