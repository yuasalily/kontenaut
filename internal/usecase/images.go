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

func (u *ImageUsecase) Delete(ctx context.Context, imageID string) error {
	return u.eng.RemoveImage(ctx, imageID)
}

func (u *ImageUsecase) LockedImageIDs(ctx context.Context) (map[string]struct{}, error) {
	containers, err := u.eng.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	locked := make(map[string]struct{}, len(containers))
	for _, c := range containers {
		if c.ImageID == "" {
			continue
		}
		locked[c.ImageID] = struct{}{}
	}
	return locked, nil
}
