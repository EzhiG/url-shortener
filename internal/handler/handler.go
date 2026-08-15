package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

type ShortenerService interface {
	ShortenUrl(str string) (string, error)
	ExpandUrl(id string) (string, error)
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	shortener ShortenerService
	baseURL   string
}

func New(shortenerService ShortenerService, baseURL string) *Handler {
	return &Handler{shortener: shortenerService, baseURL: baseURL}
}

func (h *Handler) shortenWithBaseUrl(originUrl string) (string, error) {
	id, err := h.shortener.ShortenUrl(originUrl)

	if err != nil {
		return "", err
	}

	shortenedUrl, err := url.JoinPath(h.baseURL, id)
	return shortenedUrl, err
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

func (h *Handler) PlainPostShortenUrl(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortenedUrl, err := h.shortenWithBaseUrl(string(data))

	if errors.Is(err, shortener.ErrIdGenerationFailed) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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

func (h *Handler) ApiPostShortenUrl(w http.ResponseWriter, r *http.Request) {
	var req model.Request
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortenedUrl, err := h.shortenWithBaseUrl(req.Url)

	if errors.Is(err, shortener.ErrIdGenerationFailed) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := model.Response{Result: shortenedUrl}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(&resp)

	if err != nil {
		log.Println(err)
	}
}
