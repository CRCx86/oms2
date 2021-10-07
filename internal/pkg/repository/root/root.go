package root

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"oms2/internal/pkg/storage/postgres"
)

var (
	ErrBadModel         = errors.New("bad model")
	ErrValidationFailed = errors.New("validation failed")
)

const (
	id     = "id"
	number = "number"
	name   = "name"
)

type Repository struct {
	zl      *zap.Logger
	storage *postgres.Postgres
}

func NewRepository(s *postgres.Postgres, log *zap.Logger) *Repository {
	return &Repository{
		zl:      log,
		storage: s,
	}
}

func (r *Repository) List(ctx context.Context) (interface{}, error) {

	r.zl.Sugar().Info("step: ", ctx)

	return nil, nil
}
