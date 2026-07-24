package main

import (
	"net/http"

	"github.com/EzhiG/url-shortener/internal/config"
	"github.com/EzhiG/url-shortener/internal/handler"
	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/shortener"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewConfig()
	storage := repository.NewMapStorage()
	service := shortener.NewService(storage)
	h := handler.NewHandler(service, cfg.BaseURL)

	router := chi.NewRouter()
	router.Post("/", h.PostShortenUrl)
	router.Get("/{id}", h.GetShortenUrl)

	err := http.ListenAndServe(cfg.Address, router)

	if err != nil {
		panic(err)
	}
}
