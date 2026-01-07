package engine

import (
	"context"
	"io"
)

type Engine interface {
	Close() error
	DaemonInfo(ctx context.Context) (DaemonInfo, error)
	ListImages(ctx context.Context) ([]ImageSummary, error)
	ListContainers(ctx context.Context) ([]ContainerSummary, error)
	RemoveImage(ctx context.Context, imageID string) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error

	// ContainerLogs returns a snapshot of contaienr logs without follow
	//
	// NOTE:
	// Follow-based log streaming is implemented separately
	// This snapshot API is kept intentionally for non-streaming use cases.
	ContainerLogs(ctx context.Context, containerID string, tail int) ([]string, error)

	// ContainerLogsFollow returns a streaming reader for container logs (tail + follow).
	// Caller must Close() it and should cancel the provided ctx to stop streaming.
	ContainerLogsFollow(ctx context.Context, containerID string, tail int) (io.ReadCloser, error)
}
