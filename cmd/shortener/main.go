package main

import (
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
)

var store = make(map[string]string)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const idLength = 8

func generateId() string {
	var b strings.Builder
	for range idLength {
		b.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return b.String()
}

func expandUrl(id string) string {
	return store[id]
}

func shortenUrl(str string) (string, error) {
	url, err := url.Parse(str)

	if (err != nil) || (url.Scheme != "http" && url.Scheme != "https") {
		return "", errors.New("invalid url")
	}

	id := generateId()
	store[id] = url.String()

	return id, nil
}

func getShortenUrl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	expanded := expandUrl(id)

	if expanded == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", expanded)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func postShortenUrl(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := shortenUrl(string(data))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://localhost:8080/" + id))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", postShortenUrl)
	mux.HandleFunc("GET /{id}", getShortenUrl)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
