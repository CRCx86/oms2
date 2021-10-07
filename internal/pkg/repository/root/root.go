package root

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"oms2/internal/pkg/util"

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

func (r *Repository) List(ctx context.Context, _sql string, args ...interface{}) (interface{}, error) {

	conn, err := r.storage.Conn(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, _sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects, err := util.ParseRowQuery(rows)
	if err != nil {
		return nil, err
	}

	return objects, nil
}
