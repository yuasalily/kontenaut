package usecase

import (
	"context"
	"io"

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

// Logs returns a snapshot of container logs (non-follow).
//
// NOTE:
// This method is currently not used by the TUI logs page, which relies on follow-based streaming instead.
// It is intentionally kept as a snapshot API for potential future use (e.g. export, copy, fallback, or tests).
func (u *ContainerUsecase) Logs(ctx context.Context, containerID string, tail int) ([]string, error) {
	return u.eng.ContainerLogs(ctx, containerID, tail)
}

func (u *ContainerUsecase) LogsFollow(ctx context.Context, containerID string, tail int) (io.ReadCloser, error) {
	return u.eng.ContainerLogsFollow(ctx, containerID, tail)
}
