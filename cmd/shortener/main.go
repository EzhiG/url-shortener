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

	database, err := db.NewPostgres(cfg.DatabaseDSN)
	if err != nil {
		sugar.Fatal(err)
	}
	defer database.Close()

	storage, err := repository.NewFileStorage(cfg.FileStoragePath)
	if err != nil {
		sugar.Fatal(err)
	}
	defer storage.CloseFile()
	service := shortener.New(storage)
	h := handler.New(service, cfg.BaseURL, database, sugar)
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
