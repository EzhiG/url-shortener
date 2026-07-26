package handler

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

type ShortenerService interface {
	ShortenUrl(str string) (string, error)
	ExpandUrl(id string) (string, error)
}

type Handler struct {
	service ShortenerService
	baseURL string
}

func NewHandler(service ShortenerService, baseURL string) *Handler {
	return &Handler{service: service, baseURL: baseURL}
}

func (h *Handler) GetShortenUrl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	expanded, err := h.service.ExpandUrl(id)

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

	id, err := h.service.ShortenUrl(string(data))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortenedUrl, err := url.JoinPath(h.baseURL + "/" + id)

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
