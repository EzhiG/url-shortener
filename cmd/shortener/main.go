package main

import (
	"log"
	"net/http"

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
		log.Fatal(err)
	}
	defer sugar.Sync()
	cfg := config.New()
	storage := repository.NewMapStorage()
	service := shortener.New(storage)
	h := handler.New(service, cfg.BaseURL)
	mw := logger.NewMiddleware(sugar)

	router := chi.NewRouter()
	router.Use(mw)
	router.Post("/", h.PlainPostShortenUrl)
	router.Post("/api/shorten", h.ApiPostShortenUrl)
	router.Get("/{id}", h.GetShortenUrl)

	err = http.ListenAndServe(cfg.Address, router)

	if err != nil {
		log.Fatal(err)
	}
}
