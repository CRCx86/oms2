package root

import (
	"context"
	"go.uber.org/zap"

	"oms2/internal/pkg/storage/postgres"
	"oms2/internal/pkg/util"
)

//var (
//	ErrBadModel         = errors.New("bad model")
//	ErrValidationFailed = errors.New("validation failed")
//)

//const (
//	Id     = "id"
//	Number = "number"
//	Name   = "name"
//)

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

func (r *Repository) CreateOrUpdate(ctx context.Context, _sql string, args ...interface{}) (uint, error) {

	conn, err := r.storage.Conn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err
	}

	var result uint
	err = tx.QueryRow(ctx, _sql, args...).Scan(&result)
	if err != nil {
		err = tx.Rollback(ctx)
		return 0, err
	}
	err = tx.Commit(ctx)

	return result, err

}

func (r *Repository) Get(ctx context.Context, _sql string, args ...interface{}) ([]map[string]interface{}, error) {

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

func (r *Repository) Delete(ctx context.Context, sql string, args ...interface{}) error {

	conn, err := r.storage.Conn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	err = tx.Commit(ctx)

	return err
}
