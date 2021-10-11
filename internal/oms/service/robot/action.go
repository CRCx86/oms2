package robot

import (
	"fmt"
	"go.uber.org/zap"
	"time"

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

func (a *Action) FirstInit(data interface{}) error {
	fmt.Println("FirstInit", data)
	time.Sleep(1 * time.Second)
	return nil
}

func (a *Action) SecondInit(data interface{}) error {
	fmt.Println("SecondInit", data)
	time.Sleep(1 * time.Second)
	return nil
}
