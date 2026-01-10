package usecase

import (
	"bufio"
	"context"
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

// ContainerUsecase provides container operations used by the UI.
//
// It is intentionally thin: it keeps UI logic (selection, dialogs) out of infra,
// while keeping engine-specific details out of the UI.
type ContainerUsecase struct {
	eng engine.Engine
}


// NewContainerUsecase constructs a ContainerUsecase.
func NewContainerUsecase(eng engine.Engine) *ContainerUsecase {
	return &ContainerUsecase{eng: eng}
}

// List returns all containers (including stopped ones).
func (u *ContainerUsecase) List(ctx context.Context) ([]engine.ContainerSummary, error) {
	return u.eng.ListContainers(ctx)
}

// Delete removes a container without forcing.
//
// Why:
// - The TUI treats running containers as "locked" and does not offer force delete.
func (u *ContainerUsecase) Delete(ctx context.Context, containerID string) error {
	return u.eng.RemoveContainer(ctx, containerID, false)
}

// Logs returns a snapshot of container logs (non-follow).
//
// NOTE:
// This method is currently not used by the TUI logs page, which relies on follow-based streaming instead.
// It is intentionally kept as a snapshot API for potential future use (e.g. export, copy, fallback, or tests).
//
// Why snapshot exists:
// - Some UX features want a bounded, consistent log view (copy/export).
// Streaming is handled separately to keep this method simple and predictable.
func (u *ContainerUsecase) Logs(ctx context.Context, containerID string, tail int) ([]string, error) {
	return u.eng.ContainerLogs(ctx, containerID, tail)
}

// LogEvent is a UI-agnostic event for streaming container logs.
//
// - Line: a single log line (without trailing newline).
// - Done: true when the steam ends normally (EOF / ctx cancel)
// - Err: non-nil when the stream ends due to an error.
type LogEvent struct {
	Line string
	Done bool
	Err  error
}

// FollowLogs streams container logs as line events.
//
// This method intentionally hides io.ReaderCloser from callers.
// Cancellation is controlled by ctx; the internal reader will be closed on exit.
//
// Why channel-based API:
// - Bubble Tea updates are message-driven; a channel maps cleanly to Cmd -> Msg flow.
// - The UI should not manage io.ReadCloser lifetime directly.
// - We can normalize Docker's multiplexed stream and deliver plain lines to the UI.
func (u *ContainerUsecase) FollowLogs(ctx context.Context, containerID string, tail int) (<-chan LogEvent, error) {
	rc, err := u.eng.ContainerLogsFollow(ctx, containerID, tail)
	if err != nil {
		return nil, err
	}

	ch := make(chan LogEvent, 100)

	go func() {
		defer close(ch)
		defer func() { _ = rc.Close() }()

		send := func(ev LogEvent) bool {
			select {
			case ch <- ev:
				return true
			case <-ctx.Done():
				return false
			}
		}

		// Docker logs may be multiplexed; demux into a plain text stream for scanning.
		//
		// Why io.Pipe:
		// - stdcopy.StdCopy consumes the raw stream and writes a plain text stream.
		// - bufio.Scanner wants an io.Reader; Pipe bridges these without buffering everything in memory.
		pr, pw := io.Pipe()
		defer func() { _ = pr.Close() }()
		go func() {
			defer func() { _ = pw.Close() }()
			_, err := stdcopy.StdCopy(pw, pw, rc)
			if err != nil {
				_ = pw.CloseWithError(err)
			}
		}()

		sc := bufio.NewScanner(pr)
		// allow reasonably long log lines
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for sc.Scan() {
			if !send(LogEvent{Line: sc.Text()}) {
				return
			}
		}

		// Stream ended.
		if ctx.Err() != nil {
			_ = send(LogEvent{Done: true})
			return
		}
		if serr := sc.Err(); serr != nil {
			_ = send(LogEvent{Err: serr})
			return
		}
		_ = send(LogEvent{Done: true})
	}()

	return ch, nil
}
