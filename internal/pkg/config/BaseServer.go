package config

import "time"

type BaseServer struct {
	Version         string        `envconfig:"version"`
	ReadTimeout     time.Duration `envconfig:"read_timeout" default:"30s"`
	WriteTimeout    time.Duration `envconfig:"write_timeout" default:"30s"`
	ShutdownTimeout time.Duration `envconfig:"shutdown_timeout" default:"60s"`
	CORSOrigin      string        `envconfig:"cors_origin" default:"http://localhost:8765"`
	CORSHeaders     string        `envconfig:"cors_headers" default:"content-type, cache-control"`
	CORSCredentials string        `envconfig:"cors_credentials" default:"true"`
}
