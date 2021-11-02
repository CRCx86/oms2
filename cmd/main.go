package main

import (
	"os"

	"go.uber.org/fx"

	"oms2/internal/oms"
	"oms2/internal/oms/app"
	"oms2/internal/pkg/logger"
	"oms2/internal/pkg/tracing"
)

var (
	version   string
	buildDate string
	commit    string
)

func main() {

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		oms.Usage()
		return
	}

	conf, err := oms.NewConfig()
	if err != nil {
		panic(err)
	}

	conf.Version = version
	conf.BuildDate = buildDate
	conf.Commit = commit

	zapLogger, err := logger.New(app.Name, *conf)
	if err != nil {
		panic(err)
	}
	tracing.New(zapLogger)

	defer app.Recover(zapLogger)
	fx.New(
		app.Provide(conf, zapLogger),
	).Run()

}
