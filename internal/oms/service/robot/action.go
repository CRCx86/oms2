package robot

import (
	"context"

	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"

	"oms2/internal/oms"
	actionR "oms2/internal/pkg/repository/action"
)

type Action struct {
	zl      *zap.Logger
	cfg     *oms.Config
	actionR *actionR.Repository
}

func NewAction(cfg *oms.Config, actionR *actionR.Repository, zl *zap.Logger) *Action {
	return &Action{
		zl:      zl,
		cfg:     cfg,
		actionR: actionR,
	}
}

func (a *Action) FirstInit(ctx interface{}, data interface{}) error {

	c := ctx.(context.Context)

	_sql, args, err := squirrel.
		StatementBuilder.
		Select("*").
		From("lots").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}
	list, err := a.actionR.RootRepository.Get(c, _sql, args...)
	if err != nil {
		return err
	}

	a.zl.Sugar().Info("FirstInit", data, list)
	return nil
}

func (a *Action) SecondInit(ctx interface{}, data interface{}) error {

	a.zl.Sugar().Info("SecondInit", data)
	return nil
}
