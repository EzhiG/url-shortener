package main

import (
	"net/http"

	"github.com/EzhiG/url-shortener/internal/auth"
	"github.com/EzhiG/url-shortener/internal/config"
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

	shortenerService := shortener.New(storage)
	authService := auth.New("SuperSecretKey") // TODO: move to ENV
	h := handler.New(authService, shortenerService, cfg.BaseURL, sugar)
	loggerMw := logger.NewMiddleware(sugar)
	authMw := handler.NewAuthMiddleware(authService)

	router := chi.NewRouter()
	router.Use(loggerMw, handler.MaxBytesMiddleware, handler.GzipMiddleware, authMw)
	router.Post("/", h.PlainPostShortenURL)
	router.Post("/api/shorten", h.APIPostShortenURL)
	router.Post("/api/shorten/batch", h.APIPostBatchShortenURL)
	router.Get("/api/user/urls", h.APIGetUserURLs)
	router.Get("/{id}", h.GetShortenURL)
	router.Get("/ping", h.Ping)

	err = http.ListenAndServe(cfg.Address, router)

	if err != nil {
		sugar.Fatal(err)
	}
}

func createAppStorage(cfg *config.Config) (repository.Storage, error) {
	if cfg.DatabaseDSN != "" {
		return repository.NewDBStorage(cfg.DatabaseDSN)
	}
	if cfg.FileStoragePath != "" {
		storage, err := repository.NewFileStorage(cfg.FileStoragePath)
		return storage, err
	}

	return repository.NewMapStorage(), nil
}
