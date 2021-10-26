package v7

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"oms2/internal/oms"
	v7 "oms2/internal/pkg/storage/elastic/v7"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(func(cfg *oms.Config, zl *zap.Logger) *v7.Elastic {
			return v7.NewElastic(cfg.V7Elastic, zl)
		}),

		fx.Invoke(func(lc fx.Lifecycle, elastic *v7.Elastic) {
			lc.Append(fx.Hook{
				OnStart: elastic.Start,
				OnStop:  elastic.Stop,
			})
		}),
	)
}
