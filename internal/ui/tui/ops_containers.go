package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

// Shared Containers operations (Cmd + Msg types).
//
// Why shared:
// - Containers normal mode and delete mode need the same operations:
//   list containers nad delete containers.
// - Keep IO inside Cmd (not Update) and keep pages focused on UI state.

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

type containersDeletedMsg struct {
	deleted  int
	failed   int
	firstErr error
}

func deleteContainersCmd(containerUC *usecase.ContainerUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		deleted := 0
		failed := 0
		var firstErr error

		for _, id := range ids {
			if err := containerUC.Delete(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			deleted++
		}
		return containersDeletedMsg{
			deleted:  deleted,
			failed:   failed,
			firstErr: firstErr,
		}
	}
}

type containersStoppedMsg struct {
	stopped  int
	failed   int
	firstErr error
}

// stopContainersCmd stops containers sequentially.
func stopContainersCmd(containerUC *usecase.ContainerUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		stopped := 0
		failed := 0
		var firstErr error

		for _, id := range ids {
			if err := containerUC.Stop(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			stopped++
		}

		return containersStoppedMsg{
			stopped:  stopped,
			failed:   failed,
			firstErr: firstErr,
		}
	}
}

type containersRestartedMsg struct {
	restarted int
	failed    int
	firstErr  error
}

// restartContainersCmd restarts containers sequentially.
func restartContainersCmd(containerUC *usecase.ContainerUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		restarted := 0
		failed := 0
		var firstErr error

		for _, id := range ids {
			if err := containerUC.Restart(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			restarted++
		}

		return containersRestartedMsg{
			restarted: restarted,
			failed:    failed,
			firstErr:  firstErr,
		}
	}
}

type containersStartedMsg struct {
	started  int
	failed   int
	firstErr error
}

// startContainersCmd starts containers sequentially.
//
// Why sequential + counts:
// - Keep the initial behavior consistent with deleteContainersCmd.
// - Provide a simple, UI-friendly summary (started/failed + first error).
// - Avoid overwhelming the daemon with consurrent requests (initial policy).
func startContainersCmd(containerUC *usecase.ContainerUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		started := 0
		failed := 0
		var firstErr error

		for _, id := range ids {
			if err := containerUC.Start(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			started++
		}

		return containersStartedMsg{
			started:  started,
			failed:   failed,
			firstErr: firstErr,
		}
	}
}
