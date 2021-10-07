package config

type Storage struct {
	Host         string
	Port         int
	DBName       []string
	User         string
	Password     string
	MaxOpenConns int
	MaxIdleConns int
}
