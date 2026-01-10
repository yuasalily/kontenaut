package usecase

import (
	"context"

	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

// DaemonUsecase provides daemon-level operations used by the UI.
type DaemonUsecase struct {
	eng engine.Engine
}

// NewDaemonUsecase constructs a DaemonUsecase.
func NewDaemonUsecase(eng engine.Engine) *DaemonUsecase {
	return &DaemonUsecase{eng: eng}
}

// Info returns basic daemon metadata for the Overview page.
func (u *DaemonUsecase) Info(ctx context.Context) (engine.DaemonInfo, error) {
	return u.eng.DaemonInfo(ctx)
}
