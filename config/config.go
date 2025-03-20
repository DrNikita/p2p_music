package config

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MusicPath string `envconfig:"path"`
}

func MustConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("music", &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
