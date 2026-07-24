package handler

import (
	"io"
	"net/http"

	"github.com/EzhiG/url-shortener/internal/shortener"
)

type Handler struct {
	service *shortener.Service
}

func NewHandler(service *shortener.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetShortenUrl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	expanded, ok := h.service.ExpandUrl(id)

	if !ok {
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

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://localhost:8080/" + id))
}
