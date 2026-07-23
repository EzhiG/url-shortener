package main

import (
	"net/http"

	"github.com/EzhiG/url-shortener/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", handler.PostShortenUrl)
	mux.HandleFunc("GET /{id}", handler.GetShortenUrl)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
