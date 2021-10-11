package oms

import (
	"encoding/json"
	"time"

	"github.com/kelseyhightower/envconfig"

	"oms2/internal/pkg/config"
	"oms2/internal/pkg/storage/postgres"
)

const CoreEnvironmentPrefix = "OMS"

const EnvDev = "dev"

type Config struct {
	Env            string           `envconfig:"env"`
	Debug          bool             `envconfig:"debug"`
	ProfilerEnable bool             `envconfig:"pprof"`
	StartTimeout   time.Duration    `envconfig:"start_timeout" default:"20s"`
	StopTimeout    time.Duration    `envconfig:"stop_timeout" default:"60s"`
	APIServer      config.APIServer `envconfig:"apiserver"`
	Storage        config.Storage   `envconfig:"storage"`
	Postgres       postgres.Config  `envconfig:"postgres"`
	Logger         config.Logger    `envconfig:"logger"`
	Version        string
	BuildDate      string
	Commit         string
}

func NewConfig() (*Config, error) {

	cfg := &Config{}

	if err := envconfig.Process(CoreEnvironmentPrefix, cfg); err != nil {
		return nil, err
	}

	if cfg.Debug {
		cfg.Logger.Level = "debug"
		cfg.Logger.Debug = true
	}

	return cfg, nil
}

const (
	KeyMeta = "meta"

	KeyRequest  = "requestData"
	KeyResponse = "responseData"
)

type Envelope struct {
	Meta json.RawMessage `json:"meta"`
}

type Request struct {
	Envelope
	Data json.RawMessage `json:"data" binding:"required"`
}

type ResponseSuccess struct {
	Success int `json:"success"`
	Envelope
	Data json.RawMessage `json:"data" binding:"required"`
}

type RError struct {
	Message    json.RawMessage `json:"message" binding:"required"`
	StackTrace []string        `json:"stackTrace" binding:"required"`
}

type ResponseError struct {
	Success int `json:"success"`
	Envelope
	Error RError `json:"error" binding:"required"`
}
