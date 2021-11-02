package config

type Elastic struct {
	Url                 string `envconfig:"url" default:"http://localhost:9700"`
	Login               string `envconfig:"login"`
	Password            string `envconfig:"password"`
	Sniff               bool   `envconfig:"sniff" default:"false"`
	HealthCheckInterval int    `envconfig:"health_check_interval" default:"10"`
	SkipIndexCreation   bool   `envconfig:"skip_index_creation"`
}
