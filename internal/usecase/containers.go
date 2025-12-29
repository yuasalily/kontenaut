package usecase

import (
	"context"

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
