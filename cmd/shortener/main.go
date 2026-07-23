package main

import (
	"net/http"

	"github.com/EzhiG/url-shortener/internal/handler"
	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/service"
)

func main() {
	storage := repository.NewMapStorage()
	serv := service.NewService(storage)
	h := handler.NewHandler(serv)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.PostShortenUrl)
	mux.HandleFunc("GET /{id}", h.GetShortenUrl)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
