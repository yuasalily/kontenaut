package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	vp viewport.Model
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
		vp:            viewport.New(0, 0),
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
		p = p.applyViewportLayout()
		p.refreshViewportContent(true, false)
		return p, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			return p, func() tea.Msg { return navigateMsg{to: pageContainers} }
		}
		if !p.loading {
			var cmd tea.Cmd
			p.vp, cmd = p.vp.Update(msg)
			return p, cmd
		}

	case logsLoadedMsg:
		p.loading = false
		p.lines = msg.lines
		p.refreshViewportContent(false, true)
		return p, nil

	case logsLoadFailedMsg:
		p.loading = false
		p.lines = nil
		return p, openDialogCmd(dialogError, "Logs", msg.err.Error())
	}

	return p, nil
}

func (p logsPage) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		p.headerView(),
		"",
		p.bodyView(),
		"",
		p.footerView(),
	)
}

func (p logsPage) headerView() string {
	if p.containerName == "" {
		return "Logs"
	}
	return fmt.Sprintf("Logs: %s", p.containerName)
}

func (p logsPage) footerView() string {
	return "(↑/↓: scroll, esc/b: back, q:quit)"
}

func (p logsPage) bodyView() string {
	if p.loading {
		return "Loading..."
	}
	if len(p.lines) == 0 {
		return "<no logs>"
	}
	return p.vp.View()
}

func (p logsPage) applyViewportLayout() logsPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	bodyH := max(p.height-4, 1)
	p.vp.Width = p.width
	p.vp.Height = bodyH
	return p
}

func (p *logsPage) refreshViewportContent(keepOffset bool, gotoBottom bool) {
	if p.loading || p.vp.Width <= 0 || p.vp.Height <= 0 {
		return
	}

	content := strings.Join(p.lines, "\n")
	p.vp.SetContent(content)

	if gotoBottom {
		p.vp.GotoBottom()
		return
	}

	var prevY int
	if keepOffset {
		prevY = p.vp.YOffset
	}

	if keepOffset {
		p.vp.SetYOffset(prevY)
	}
}
