package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

// New returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func New(l *zap.Logger) {
	l = l.Named("tracing")
	cfg, err := config.FromEnv()
	if err != nil {
		l.Warn("Сan not init Jaeger (parse ENV error)",
			zap.Error(err))
		opentracing.SetGlobalTracer(opentracing.NoopTracer{})
		return
	}

	tracer, _, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		l.Warn("Сan not init Jaeger", zap.Error(err))
		opentracing.SetGlobalTracer(opentracing.NoopTracer{})
		return
	}

	opentracing.SetGlobalTracer(tracer)

	l.Info("Tracer successfully initiated")
}
