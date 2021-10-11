package robot

import (
	"errors"

	"go.uber.org/zap"

	"oms2/internal/pkg/repository/root"
	"oms2/internal/pkg/storage/postgres"
)

var (
	ErrBadModel         = errors.New("bad model")
	ErrValidationFailed = errors.New("validation failed")
)

type Repository struct {
	zl             *zap.Logger
	storage        *postgres.Postgres
	rootRepository *root.Repository
}

func NewRepository(s *postgres.Postgres, root *root.Repository, zl *zap.Logger) *Repository {
	return &Repository{
		zl:             zl,
		storage:        s,
		rootRepository: root,
	}
}

func (r *Repository) Processing() ([]map[string]interface{}, error) {
	return nil, nil
}
