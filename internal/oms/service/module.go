package service

import (
	"go.uber.org/fx"
	"oms2/internal/pkg/service/health"
	"oms2/internal/pkg/service/log"
	robot2 "oms2/internal/pkg/service/robot"

	"oms2/internal/oms"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(log.NewService),
		fx.Provide(health.NewService),
		fx.Provide(robot2.NewAction),
		fx.Provide(robot2.NewService),

		fx.Invoke(func(lc fx.Lifecycle, cfg *oms.Config, service *robot2.Service) {
			lc.Append(fx.Hook{
				OnStart: service.Start,
				OnStop:  service.Stop,
			})
		}),
	)
}
