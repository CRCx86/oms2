package repository

import (
	"go.uber.org/fx"

	"oms2/internal/pkg/repository/robot"
	"oms2/internal/pkg/repository/root"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(root.NewRepository),
		fx.Provide(robot.NewRepository),
	)
}
