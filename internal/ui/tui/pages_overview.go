package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type overviewPage struct {
	daemonUC *usecase.DaemonUsecase

	loading bool
	info    *engine.DaemonInfo

	width  int
	height int

	gkm globalKeyMap
	km  overviewKeyMap
}

// compile-time interface check
var _ Page = overviewPage{}

func newOverviewPage(gkm globalKeyMap, daemonUC *usecase.DaemonUsecase) Page {
	return overviewPage{daemonUC: daemonUC, loading: true, gkm: gkm, km: newOverviewKeyMap()}
}

type daemonInfoLoadedMsg engine.DaemonInfo
type daemonInfoLoadFailedMsg struct{ err error }

func daemonInfoCmd(daemonUC *usecase.DaemonUsecase) tea.Cmd {
	return func() tea.Msg {
		info, err := daemonUC.Info(context.Background())
		if err != nil {
			return daemonInfoLoadFailedMsg{err: err}
		}
		return daemonInfoLoadedMsg(info)
	}
}

func (p overviewPage) Init() tea.Cmd { return daemonInfoCmd(p.daemonUC) }

func (p overviewPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil

	case tea.KeyMsg:
		if key.Matches(msg, p.km.Refresh) {
			// reload daemon info
			p.loading = true
			p.info = nil
			return p, p.Init()
		}

	case daemonInfoLoadedMsg:
		p.loading = false
		x := engine.DaemonInfo(msg)
		p.info = &x
		return p, nil

	case daemonInfoLoadFailedMsg:
		p.loading = false
		p.info = nil
		return p, openDialogCmd(dialogError, "Overview", msg.err.Error())
	}
	return p, nil
}

func (p overviewPage) View() string {
	var b strings.Builder
	b.WriteString("Overview\n\n")

	if p.loading {
		b.WriteString("Loading...\n")
		return b.String()
	}

	if p.info == nil {
		b.WriteString("Failed to connect to the Docker daemon.\n")
		b.WriteString("Please make sure the daemon is running.\n")
		footer := renderHelpBlock(p.width, p.km.Refresh, p.gkm.Quit)
		if footer != "" {
			b.WriteString(footer + "\n")
		}
		return b.String()
	}

	b.WriteString("Docker daemon: OK\n\n")
	b.WriteString(fmt.Sprintf("Version: %s\n", p.info.ServerVersion))
	b.WriteString(fmt.Sprintf("OS: %s\n\n", p.info.OperatingSystem))

	footer := renderHelpBlock(p.width, p.gkm.NavOverview, p.gkm.NavImages, p.gkm.NavContainers, p.km.Refresh, p.gkm.Quit)
	if footer != "" {
		b.WriteString(fmt.Sprintf("%s\n", footer))
	}
	return b.String()
}
