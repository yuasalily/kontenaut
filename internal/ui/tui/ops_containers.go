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

type containerStartedMsg struct {
	id   string
	name string
	err  error
}

func startContainerCmd(containerUC *usecase.ContainerUsecase, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := containerUC.Start(context.Background(), id)
		return containerStartedMsg{id: id, name: name, err: err}
	}
}
