package app

import "github.com/agidelle/TODO_web_v2/internal/api"

type App struct {
	cfg      *Config
	handlers *api.TaskHandler
}

type Config struct {
	Port     int    `mapstructure:"TODO_PORT"`
	DBdriver string `mapstructure:"TODO_DRIVER"`
	DBPath   string `mapstructure:"TODO_DBFILE"`
	Password string `mapstructure:"TODO_PASSWORD"`
	JWTKey   string `mapstructure:"TODO_JWTSECRET"`
}
