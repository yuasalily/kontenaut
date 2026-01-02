package engine

import "context"

type Engine interface {
	DaemonInfo(ctx context.Context) (DaemonInfo, error)
	ListImages(ctx context.Context) ([]ImageSummary, error)
	ListContainers(ctx context.Context) ([]ContainerSummary, error)
	RemoveImage(ctx context.Context, imageID string) error
	ContainerLogs(ctx context.Context, containerID string, tail int) ([]string, error)
}
