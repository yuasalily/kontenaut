package engine

import "context"

type Engine interface {
	ListContaienrs(ctx context.Context) ([]ContainerSummary, error)
}