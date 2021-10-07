package root

import (
	"context"
	"github.com/Masterminds/squirrel"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	postgres2 "oms2/internal/pkg/storage/postgres"
)

func TestRepository_List(t *testing.T) {
	// t.Skip("Пример работы со слоем БД")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	zl := zap.L()
	p := postgres2.NewPostgres(postgres2.Config{
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "oms",
		LogLevel:   "error",
	}, zl)
	err := p.Start(ctx)
	require.NoError(t, err)
	defer p.Stop(ctx)

	repo := NewRepository(p, zl)

	sbSQL := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	q := sbSQL.Select().From("lots l")
	q = q.Columns(
		"l.id",
		"l.name",
	)
	_sql, args, err := q.ToSql()
	require.NoError(t, err)

	list, err := repo.List(ctx, _sql, args)
	require.NoError(t, err)

	require.Len(t, list, 2)
}
