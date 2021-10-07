package apiserver

import (
	"go.uber.org/fx"

	"oms2/internal/oms/apiserver/controllers/health"
)

func Module() fx.Option {
	return fx.Options(

		fx.Provide(health.NewController),

		fx.Provide(NewAPIServer),
		fx.Invoke(
			func(lf fx.Lifecycle, server *APIServer) {
				lf.Append(fx.Hook{
					OnStart: server.Start,
					OnStop:  server.Stop,
				})
			},
		),
	)
}
