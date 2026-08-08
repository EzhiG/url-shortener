package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/EzhiG/url-shortener/internal/shortener"
)

type ShortenerService interface {
	ShortenUrl(str string) (string, error)
	ExpandUrl(id string) (string, error)
}

type Handler struct {
	shortener ShortenerService
	baseURL   string
}

func New(shortenerService ShortenerService, baseURL string) *Handler {
	return &Handler{shortener: shortenerService, baseURL: baseURL}
}

func (h *Handler) GetShortenUrl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	expanded, err := h.shortener.ExpandUrl(id)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", expanded)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) PostShortenUrl(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := h.shortener.ShortenUrl(string(data))

	if errors.Is(err, shortener.ErrIdGenerationFailed) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortenedUrl, err := url.JoinPath(h.baseURL, id)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(shortenedUrl))

	if err != nil {
		log.Println(err)
	}
}
