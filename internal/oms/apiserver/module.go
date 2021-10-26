package apiserver

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
	"oms2/internal/oms"

	"oms2/internal/oms/apiserver/controllers/health"
)

type ApiServer struct {
	fx.In
	Cfg *oms.Config
	Zl  *zap.Logger

	Health *health.Controller
}

func Module() fx.Option {
	return fx.Options(

		fx.Provide(health.NewController),

		fx.Provide(func(a ApiServer) *APIServer {
			return NewAPIServer(&a.Cfg.APIServer, a.Cfg, a.Zl).
				AddController(a.Health)
		}),

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
