package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"oms2/internal/oms"
	"oms2/internal/pkg/config"
)

func New(appName string, conf oms.Config) (*zap.Logger, error) {
	zapConf := zap.Config{
		Level:       zapLevel(conf.Logger.Level),
		Development: conf.Logger.Debug,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zapEncoderConfig(),
		OutputPaths:      conf.Logger.GetOutput(),
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapConf.Build()
	if err != nil {
		return nil, err
	}

	logger = logger.With(
		zap.Any("application", struct {
			Name      string `json:"name"`
			Version   string `json:"version"`
			BuildDate string `json:"buildDate"`
			Commit    string `json:"commit"`
		}{
			Name:      appName,
			Version:   conf.Version,
			BuildDate: conf.BuildDate,
			Commit:    conf.Commit,
		}),
	)
	return logger, nil
}

func zapLevel(level string) (l zap.AtomicLevel) {
	switch level {
	case config.DebugLevel:
		l = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case config.ErrorLevel:
		return zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case config.InfoLevel:
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case config.WarnLevel:
		return zap.NewAtomicLevelAt(zapcore.WarnLevel)
	default:
		panic("unknown log level")
	}
	return l
}

func zapEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "@log_name",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stackTrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.NanosDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
