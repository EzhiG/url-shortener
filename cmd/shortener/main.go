package main

import (
	"net/http"

	"github.com/EzhiG/url-shortener/internal/handler"
	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

func main() {
	storage := repository.NewMapStorage()
	serv := shortener.NewService(storage)
	h := handler.NewHandler(serv)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.PostShortenUrl)
	mux.HandleFunc("GET /{id}", h.GetShortenUrl)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
