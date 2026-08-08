package config

import "flag"

type Config struct {
	Address string
	BaseURL string
}

func New() *Config {
	cfg := new(Config)

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "http service address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "http service base url")
	flag.Parse()

	return cfg
}
