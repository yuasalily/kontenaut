package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

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

	// full: overwrite oldest and advance start
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

// logsPage shows container logs with tail+follow.
//
// Why:
// - Logs can be high frequency; rebuilding viewport content for every line is costly.
// - We buffer lines in a fixed-size ring and rebuild periodically to keep UI responsive.
// - When user scrolls up (pinned=false), we keep the viewport "frozen" even if old lines are evicted.
type logsPage struct {
	containerUC *usecase.ContainerUsecase

	containerID   string
	containerName string

	loading bool
	ring    logRing
	dirty   bool // content needs rebuild

	// pendingYOffsetDelta accumulates YOffset adjustments while pinned=false.
	// Why (freeze semantics):
	// - When pinned=false, the user is reading older logs.
	// - The ring buffer may evict from the head as new lines arrive.
	// - Without compensation, the visible content would "jump" upward.
	// - We estimate how many wrapped display-lines were removed and offset Y accordingly.
	// Applied on the next rebuild tick to avoid doing expensive wrapping per event.
	pendingYOffsetDelta int

	width  int
	height int

	vp viewport.Model

	// follow state
	cancel context.CancelFunc
	ch     <-chan usecase.LogEvent
	pinned bool // if true, auto-follow (GotoBottom) on new lines

	gkm globalKeyMap
	km  logsKeyMap
}

// compile-time interface check
var _ Page = logsPage{}
var _ PageCloser = logsPage{}

func newLogsPage(gkm globalKeyMap, containerUC *usecase.ContainerUsecase, containerID, containerName string) Page {
	return logsPage{
		containerUC:   containerUC,
		containerID:   containerID,
		containerName: containerName,
		loading:       true,
		ring:          newLogRing(logsMaxLines),
		vp:            viewport.New(0, 0),
		pinned:        true,
		gkm:           gkm,
		km:            newLogsKeyMap(),
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
	// Why tick:
	// - Log lines may arrive faster than the terminal can redraw.
	// - Rebuilding wrapped content is relatively expensive (ANSI-aware hardwrap).
	// - Tick coalesces multiple log events into fewer SetContent calls.
	return tea.Tick(logsRebuildInterval, func(time.Time) tea.Msg {
		return logsTickMsg{}
	})
}

func (p logsPage) Init() tea.Cmd {
	return tea.Batch(startFollowLogsCmd(p.containerUC, p.containerID, logsDefaultTail), logsTickCmd())
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
		if key.Matches(msg, p.km.Back) {
			return p, func() tea.Msg { return navigateMsg{to: pageContainers} }
		}
		if key.Matches(msg, p.km.Follow) {
			p.pinned = true
			if !p.loading {
				p.vp.GotoBottom()
			}
			return p, nil
		}

		if !p.loading {
			// If user scrolls up, stop auto-follow until they jump to bottom.
			if key.Matches(msg, p.km.ScrollUp) {
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
			// Channel closed -> follow ended normally
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
		// Freeze semantics:
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
	footer := renderHelpBlock(p.width, p.km.ScrollUp, p.km.Follow, p.km.Back, p.gkm.Quit)
	if footer == "" {
		return ""
	}
	return footer
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
	bodyH := max(p.height-logsNonBodyRows, 1)
	p.vp.Width = p.width
	p.vp.Height = bodyH
	return p
}

func wrapLogLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return nil
	}

	// Note: width is terminal cell width; ansi.Hardwrap is ANSI-aware and wide-char aware.
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
		prevY = max(p.vp.YOffset+p.pendingYOffsetDelta, 0)
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
	// Capture at call time
	cancel := p.cancel
	return func() tea.Msg {
		if cancel != nil {
			cancel()
		}
		return nil
	}
}
