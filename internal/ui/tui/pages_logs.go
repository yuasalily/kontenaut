package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moby/moby/api/pkg/stdcopy"
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

	// follow state
	cancel context.CancelFunc
	rc     io.ReadCloser
	sc     *bufio.Scanner
	pinned bool // if true, auto-follow (GotoBottom) on new lines
}

// compile-time interface check
var _ Page = logsPage{}
var _ PageCloser = logsPage{}

func newLogsPage(containerUC *usecase.ContainerUsecase, containerID, containerName string) Page {
	return logsPage{
		containerUC:   containerUC,
		containerID:   containerID,
		containerName: containerName,
		loading:       true,
		lines:         nil,
		vp:            viewport.New(0, 0),
		pinned:        true,
	}
}

type logsFollowStartedMsg struct {
	cancel context.CancelFunc
	rc     io.ReadCloser
}

type logsFollowFailedMsg struct{ err error }
type logsLineMsg struct{ line string }
type logsFollowStoppedMsg struct{ err error }

func startFollowLogsCmd(containerUC *usecase.ContainerUsecase, containerID string, tail int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		rc, err := containerUC.LogsFollow(ctx, containerID, tail)
		if err != nil {
			cancel()
			return logsFollowFailedMsg{err: err}
		}
		return logsFollowStartedMsg{cancel: cancel, rc: rc}
	}
}

func readNextLogLineCmd(sc *bufio.Scanner) tea.Cmd {
	return func() tea.Msg {
		if sc == nil {
			return logsFollowStoppedMsg{err: fmt.Errorf("log scanner is nil")}
		}
		if sc.Scan() {
			return logsLineMsg{line: sc.Text()}
		}
		return logsFollowStoppedMsg{err: sc.Err()}
	}
}

func (p logsPage) Init() tea.Cmd {
	return startFollowLogsCmd(p.containerUC, p.containerID, 200)
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
		case "f":
			p.pinned = true
			if !p.loading {
				p.vp.GotoBottom()
			}
			return p, nil
		}
		if !p.loading {
			// If user scrolls up, stop auto-follow until they jump to bottom.
			switch msg.String() {
			case "up", "k", "pgup":
				p.pinned = false
			}
			var cmd tea.Cmd
			p.vp, cmd = p.vp.Update(msg)
			return p, cmd
		}

	case logsFollowStartedMsg:
		p.loading = false
		p.cancel = msg.cancel
		p.rc = msg.rc

		// Docker logs may be multiplexed; demux into a plain text stream for scanning.
		pr, pw := io.Pipe()
		go func() {
			defer func() { _ = pw.Close() }()
			_, err := stdcopy.StdCopy(pw, pw, msg.rc)
			if err != nil {
				_ = pw.CloseWithError(err)
			}
		}()

		sc := bufio.NewScanner(pr)
		// allow reasonably long log lines
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		p.sc = sc

		return p, readNextLogLineCmd(p.sc)

	case logsFollowFailedMsg:
		p.loading = false
		p.lines = nil
		return p, openDialogCmd(dialogError, "Logs", msg.err.Error())

	case logsLineMsg:
		const maxLines = 5000
		p.lines = append(p.lines, msg.line)
		if len(p.lines) > maxLines {
			over := len(p.lines) - maxLines
			p.lines = p.lines[over:]
		}

		p.refreshViewportContent(true, p.pinned)
		return p, readNextLogLineCmd(p.sc)

	case logsFollowStoppedMsg:
		// follow ends when container stops, page closes, or an error happend
		if msg.err != nil {
			return p, openDialogCmd(dialogError, "Logs", msg.err.Error())
		}

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
	return "(↑/↓: scroll, f: follow, esc/b: back, q:quit)"
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

func (p logsPage) Close() tea.Cmd {
	// capture at call time
	cancel := p.cancel
	rc := p.rc
	return func() tea.Msg {
		if cancel != nil {
			cancel()
		}
		if rc != nil {
			_ = rc.Close()
		}
		return nil
	}
}
