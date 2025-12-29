package engine

import "context"

type Engine interface {
	ListContainers(ctx context.Context) ([]ContainerSummary, error)
}
