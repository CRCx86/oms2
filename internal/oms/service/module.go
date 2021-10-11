package service

import (
	"go.uber.org/fx"

	"oms2/internal/oms"
	"oms2/internal/oms/service/health"
	"oms2/internal/oms/service/robot"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(health.NewService),
		fx.Provide(robot.NewAction),
		fx.Provide(robot.NewService),

		fx.Invoke(func(lc fx.Lifecycle, cfg *oms.Config, service *robot.Service) {
			lc.Append(fx.Hook{
				OnStart: service.Start,
				OnStop:  service.Stop,
			})
		}),
	)
}
