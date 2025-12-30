package engine

import "context"

type Engine interface {
	ListImages(ctx context.Context) ([]ImageSummary, error)
	ListContainers(ctx context.Context) ([]ContainerSummary, error)
}
