package root

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"oms2/internal/pkg/storage/postgres"
	"oms2/internal/pkg/util"
)

var (
	ErrBadModel         = errors.New("bad model")
	ErrValidationFailed = errors.New("validation failed")
)

const (
	Id     = "id"
	Number = "number"
	Name   = "name"
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

func (r Repository) Create(ctx context.Context, _sql string, args ...interface{}) (uint, error) {

	conn, err := r.storage.Conn(ctx)
	if err != nil {
		return 0, err
	}

	var result uint
	err = conn.QueryRow(ctx, _sql, args...).Scan(&result)
	if err != nil {
		return 0, err
	}

	return result, err

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
