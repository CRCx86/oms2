package robot

import (
	"go.uber.org/zap"

	"oms2/internal/oms"
)

type Action struct {
	zl  *zap.Logger
	cfg *oms.Config
}

func NewAction(cfg *oms.Config, zl *zap.Logger) *Action {
	return &Action{
		zl:  zl,
		cfg: cfg,
	}
}
