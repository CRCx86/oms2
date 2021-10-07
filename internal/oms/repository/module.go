package repository

import (
	"go.uber.org/fx"

	"oms2/internal/pkg/repository/root"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(root.NewRepository),
	)
}
