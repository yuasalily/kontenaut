package usecase

import (
	"bufio"
	"context"
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

type ContainerUsecase struct {
	eng engine.Engine
}

func NewContainerUsecase(eng engine.Engine) *ContainerUsecase {
	return &ContainerUsecase{eng: eng}
}

func (u *ContainerUsecase) List(ctx context.Context) ([]engine.ContainerSummary, error) {
	return u.eng.ListContainers(ctx)
}

func (u *ContainerUsecase) Delete(ctx context.Context, containerID string) error {
	return u.eng.RemoveContainer(ctx, containerID, false)
}

// Logs returns a snapshot of container logs (non-follow).
//
// NOTE:
// This method is currently not used by the TUI logs page, which relies on follow-based streaming instead.
// It is intentionally kept as a snapshot API for potential future use (e.g. export, copy, fallback, or tests).
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
