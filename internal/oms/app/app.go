package app

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/oms/apiserver"
	"oms2/internal/oms/repository"
	"oms2/internal/oms/service"
	"oms2/internal/oms/storage/postgres"
)

const Name = "oms"

type fxLogger struct {
	logger *zap.SugaredLogger
}

func (fxl fxLogger) Printf(format string, v ...interface{}) {
	fxl.logger.Infof(format, v...)
}

func Provide(conf *oms.Config, zl *zap.Logger) fx.Option {

	return fx.Options(
		fx.StartTimeout(conf.StartTimeout),
		fx.StopTimeout(conf.StopTimeout),

		fx.Logger(
			fxLogger{logger: zl.Named(Name).Sugar()},
		),

		fx.Provide(
			func() *zap.Logger {
				return zl
			},
		),

		fx.Provide(
			func() *oms.Config {
				return conf
			}),

		postgres.Module(),
		repository.Module(),
		service.Module(),
		apiserver.Module(),

		fx.Invoke(
			func(cfg *oms.Config, logger *zap.Logger) {
				logger.Info("Order Management System has started...")
			},
		),
	)
}

func Recover(zl *zap.Logger) {
	if err := recover(); err != nil {
		zl.Fatal("app recover error", zap.Any("recoveryError", err))
	}
}
