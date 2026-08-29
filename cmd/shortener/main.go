package main

import (
	"net/http"

	"github.com/EzhiG/url-shortener/internal/config"
	"github.com/EzhiG/url-shortener/internal/config/db"
	"github.com/EzhiG/url-shortener/internal/handler"
	"github.com/EzhiG/url-shortener/internal/logger"
	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/shortener"
	"github.com/go-chi/chi/v5"
)

func main() {
	sugar, err := logger.New()
	if err != nil {
		sugar.Fatal(err)
	}
	defer sugar.Sync()
	cfg := config.New()

	storage, err := createAppStorage(cfg)
	if err != nil {
		sugar.Fatal(err)
	}
	defer storage.Close()

	service := shortener.New(storage)
	h := handler.New(service, cfg.BaseURL, sugar)
	mw := logger.NewMiddleware(sugar)

	router := chi.NewRouter()
	router.Use(mw, handler.MaxBytesMiddleware, handler.GzipMiddleware)
	router.Post("/", h.PlainPostShortenURL)
	router.Post("/api/shorten", h.ApiPostShortenURL)
	router.Get("/{id}", h.GetShortenURL)
	router.Get("/ping", h.Ping)

	err = http.ListenAndServe(cfg.Address, router)

	if err != nil {
		sugar.Fatal(err)
	}
}

func createAppStorage(cfg *config.Config) (repository.Storage, error) {
	if cfg.DatabaseDSN != "" {
		database, err := db.NewPostgres(cfg.DatabaseDSN)
		if err != nil {
			return nil, err
		}
		return repository.NewDBStorage(database), nil
	}
	if cfg.FileStoragePath != "" {
		storage, err := repository.NewFileStorage(cfg.FileStoragePath)
		return storage, err
	}

	return repository.NewMapStorage(), nil
}
