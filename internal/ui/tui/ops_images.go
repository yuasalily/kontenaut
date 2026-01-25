package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

// Shared Images operations (Cmd + Msg types).
//
// Why shared:
// - Images normal mode and delete mode need the same operations:
//   list images, list non-deletable images, and delete images.
// - Keep IO inside Cmd (not Update) and keep pages focused on UI state.

type imagesLoadedMsg []engine.ImageSummary
type imagesLoadFailedMsg struct{ err error }

func listImagesCmd(imageUC *usecase.ImageUsecase) tea.Cmd {
	return func() tea.Msg {
		items, err := imageUC.List(context.Background())
		if err != nil {
			return imagesLoadFailedMsg{err: err}
		}
		return imagesLoadedMsg(items)
	}
}

// nonDeletableImagesLoadedMsg represents "non-deletable image IDs"
// (images referenced by any container).
type nonDeletableImagesLoadedMsg map[string]struct{}
type nonDeletableImagesLoadFailedMsg struct{ err error }

func listNonDeletableImagesCmd(imageUC *usecase.ImageUsecase) tea.Cmd {
	return func() tea.Msg {
		nonDeletable, err := imageUC.NonDeletableImageIDs(context.Background())
		if err != nil {
			return nonDeletableImagesLoadFailedMsg{err: err}
		}
		return nonDeletableImagesLoadedMsg(nonDeletable)
	}
}

type imagesDeletedMsg struct {
	deleted  int
	failed   int
	firstErr error
}

func deleteImagesCmd(imagesUC *usecase.ImageUsecase, ids []string) tea.Cmd {
	return func() tea.Msg {
		deleted := 0
		failed := 0
		var firstErr error
		for _, id := range ids {
			if err := imagesUC.Delete(context.Background(), id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			deleted++
		}
		return imagesDeletedMsg{
			deleted:  deleted,
			failed:   failed,
			firstErr: firstErr,
		}
	}
}
