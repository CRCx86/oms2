package postgres

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/pkg/storage/postgres"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(func(conf *oms.Config, log *zap.Logger) *postgres.Postgres {
			return postgres.NewPostgres(conf.Postgres, log)
		}),
		fx.Invoke(func(lc fx.Lifecycle, cfg *oms.Config, storage *postgres.Postgres) {
			lc.Append(fx.Hook{
				OnStart: storage.Start,
				OnStop:  storage.Stop,
			})
		}),
	)
}
