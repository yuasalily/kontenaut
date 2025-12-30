package usecase

import (
	"context"

	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

type ImageUsecase struct {
	eng engine.Engine
}

func NewImageUsecase(eng engine.Engine) *ImageUsecase {
	return &ImageUsecase{eng: eng}
}

func (u *ImageUsecase) List(ctx context.Context) ([]engine.ImageSummary, error) {
	return u.eng.ListImages(ctx)
}
