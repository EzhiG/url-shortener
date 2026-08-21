package config

import (
	"flag"
	"os"
)

type Config struct {
	Address string
	BaseURL string
}

func New() *Config {
	cfg := new(Config)

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "http service address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "http service base url")
	flag.Parse()

	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.Address = envAddr
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	return cfg
}
