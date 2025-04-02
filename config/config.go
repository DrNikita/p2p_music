package config

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MusicPath string `envconfig:"path"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("music", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
