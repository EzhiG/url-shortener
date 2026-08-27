package config

import (
	"flag"
	"os"
)

type Config struct {
	Address         string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
}

func New() *Config {
	cfg := new(Config)

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "http service address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "http service base url")
	flag.StringVar(&cfg.FileStoragePath, "f", "storage.json", "storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database dsn")
	flag.Parse()

	if envAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.Address = envAddr
	}

	if envBaseURL, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.BaseURL = envBaseURL
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = envFileStoragePath
	}

	if envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	return cfg
}
