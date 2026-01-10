package usecase

import (
	"context"

	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

// ImageUsecase provides image operations used by the UI.
type ImageUsecase struct {
	eng engine.Engine
}

// NewImageUsecase constructs an ImageUsecase.
func NewImageUsecase(eng engine.Engine) *ImageUsecase {
	return &ImageUsecase{eng: eng}
}

// List returns all images.
func (u *ImageUsecase) List(ctx context.Context) ([]engine.ImageSummary, error) {
	return u.eng.ListImages(ctx)
}

// Delete removes an image.
func (u *ImageUsecase) Delete(ctx context.Context, imageID string) error {
	return u.eng.RemoveImage(ctx, imageID)
}


// LockedImageIDs returns image IDs that are currently references by any container.
//
// Why:
// - Docker prevents deleting images that are in use.
// - The UI marks such images as locked and skips selection/deletion to avoid noisy errors.
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
