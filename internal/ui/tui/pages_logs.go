package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

const logsMaxLines = 5000
const logsRebuildInterval = 100 * time.Millisecond

// logRing is a fixed-size ring buffer specialized for log lines.
// It keeps the last N lines in chronological order (oldest -> newest)
type logRing struct {
	buf   []string
	start int // index of the oldest element
	size  int // number of valid elements (<= len(buf))
}

func newLogRing(capacity int) logRing {
	if capacity < 0 {
		capacity = 0
	}
	return logRing{buf: make([]string, capacity)}
}

func (r *logRing) Cap() int { return len(r.buf) }
func (r *logRing) Len() int { return r.size }

// Push adds a line to the ring.
// If the ring is full, it overwrites the oldest line and returns it as evicted=true.
func (r *logRing) Push(line string) (evictedLine string, evicted bool) {
	if len(r.buf) == 0 {
		return "", false
	}

	if r.size < len(r.buf) {
		// append at logical end
		idx := (r.start + r.size) % len(r.buf)
		r.buf[idx] = line
		r.size++
		return "", false
	}

	// full :overwrite oldest and advance start
	oldest := r.buf[r.start]
	r.buf[r.start] = line
	r.start = (r.start + 1) % len(r.buf)
	return oldest, true
}

// Slice returns a newly allocated slice of lines in chronological order.
func (r *logRing) Slice() []string {
	if r.size == 0 {
		return nil
	}
	out := make([]string, r.size)
	for i := 0; i < r.size; i++ {
		idx := (r.start + i) % len(r.buf)
		out[i] = r.buf[idx]
	}
	return out
}

type logsPage struct {
	containerUC *usecase.ContainerUsecase

	containerID   string
	containerName string

	loading bool
	ring    logRing
	dirty   bool // content needs rebuild

	// pendingYOffsetDelta accumulates YOffset adjustments while pinned=false.
	// It compensates for ring eviction so the viewport keeps showing the same log content (freeze semantics).
	// Applied on the next rebuild tick.
	pendingYOffsetDelta int

	width  int
	height int

	vp viewport.Model

	// follow state
	cancel context.CancelFunc
	ch     <-chan usecase.LogEvent
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
		ring:          newLogRing(logsMaxLines),
		vp:            viewport.New(0, 0),
		pinned:        true,
	}
}

type logsFollowStartedMsg struct {
	cancel context.CancelFunc
	ch     <-chan usecase.LogEvent
}

type logsFollowFailedMsg struct{ err error }
type logsEventReceivedMsg struct {
	ev usecase.LogEvent
	ok bool
}

type logsTickMsg struct{}

func startFollowLogsCmd(containerUC *usecase.ContainerUsecase, containerID string, tail int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := containerUC.FollowLogs(ctx, containerID, tail)
		if err != nil {
			cancel()
			return logsFollowFailedMsg{err: err}
		}
		return logsFollowStartedMsg{cancel: cancel, ch: ch}
	}
}

func waitNextLogEventCmd(ch <-chan usecase.LogEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return logsFollowFailedMsg{err: fmt.Errorf("log channel is nil")}
		}
		ev, ok := <-ch
		return logsEventReceivedMsg{ev: ev, ok: ok}
	}
}

func logsTickCmd() tea.Cmd {
	return tea.Tick(logsRebuildInterval, func(time.Time) tea.Msg {
		return logsTickMsg{}
	})
}

func (p logsPage) Init() tea.Cmd {
	return tea.Batch(startFollowLogsCmd(p.containerUC, p.containerID, 200), logsTickCmd())
}

func (p logsPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		p = p.applyViewportLayout()
		if !p.loading {
			// Resize is rare; rebuild immediately so wrapping reflects the new width
			p.pendingYOffsetDelta = 0
			p.rebuildViewportContent(p.pinned)
		}
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
		p.ch = msg.ch
		return p, waitNextLogEventCmd(p.ch)

	case logsFollowFailedMsg:
		p.loading = false
		return p, openDialogCmd(dialogError, "Logs", msg.err.Error())

	case logsEventReceivedMsg:
		if !msg.ok {
			// channel closed -> follow ended normally
			return p, nil
		}
		if msg.ev.Err != nil {
			return p, openDialogCmd(dialogError, "Logs", msg.ev.Err.Error())
		}
		if msg.ev.Done {
			// follow ends when container stops, page closes, or ctx is canceled
			return p, nil
		}
		evictedLine, evicted := p.ring.Push(msg.ev.Line)
		// Freeze smentics:
		// When pinned=false, keep showing the same content even if the ring evicts from the top.
		// Compensate YOffset by the number of wrapped display-lines removed from the head.
		if evicted && !p.pinned {
			pending := wrappedLineCount(evictedLine, p.vp.Width)
			p.pendingYOffsetDelta -= pending
		}

		p.dirty = true
		return p, waitNextLogEventCmd(p.ch)

	case logsTickMsg:
		if !p.loading && p.dirty {
			p.rebuildViewportContent(p.pinned)
			p.dirty = false
		}
		return p, logsTickCmd()
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
	if p.ring.Len() == 0 {
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

func wrapLogLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return nil
	}

	// NOTE: width is terminal cell width; ansi.Hardwrap is ANSI-aware and wide-char aware.
	if width <= 0 {
		return lines
	}
	if width < 1 {
		width = 1
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		// Preserve leading spaces (e.g. stack traces, indented logs)
		wrapped := ansi.Hardwrap(line, width, true)
		// Hardwrap may return a block containing '\n'.
		parts := strings.Split(wrapped, "\n")
		out = append(out, parts...)
	}
	return out
}

// wrappedLineCount returns how many viewport "display lines" a single log line occupies after wrapping.
// It mirrors wrapLogLines for a single line, but returns count only.
func wrappedLineCount(line string, width int) int {
	// If viewport width is not ready, fall back to 1.
	if width <= 0 {
		return 1
	}
	if width < 1 {
		width = 1
	}
	wrapped := ansi.Hardwrap(line, width, true)
	// Hardwrap may contain '\n'. Count segments.
	if wrapped == "" {
		return 1
	}
	return strings.Count(wrapped, "\n") + 1
}

func (p *logsPage) rebuildViewportContent(gotoBottom bool) {
	if p.loading || p.vp.Width <= 0 || p.vp.Height <= 0 {
		return
	}

	keepOffset := !gotoBottom
	var prevY int
	if keepOffset {
		prevY = max(p.vp.YOffset + p.pendingYOffsetDelta, 0)
	}

	// delta is applied (or discarded) on rebuild.
	p.pendingYOffsetDelta = 0

	lines := p.ring.Slice()
	viewLines := wrapLogLines(lines, p.vp.Width)
	content := strings.Join(viewLines, "\n")
	p.vp.SetContent(content)

	if gotoBottom {
		p.vp.GotoBottom()
		return
	}

	p.vp.SetYOffset(prevY)
}

func (p logsPage) Close() tea.Cmd {
	// capture at call time
	cancel := p.cancel
	return func() tea.Msg {
		if cancel != nil {
			cancel()
		}
		return nil
	}
}
