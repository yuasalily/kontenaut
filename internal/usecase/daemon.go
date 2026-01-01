package usecase

import (
	"context"

	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

type DaemonUsecase struct {
	eng engine.Engine
}

func NewDaemonUsecase(eng engine.Engine) *DaemonUsecase {
	return &DaemonUsecase{eng: eng}
}

func (u *DaemonUsecase) Info(ctx context.Context) (engine.DaemonInfo, error) {
	return u.eng.DaemonInfo(ctx)
}