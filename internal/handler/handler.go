package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/EzhiG/url-shortener/internal/shortener"
	"go.uber.org/zap"
)

type ShortenerService interface {
	ShortenURL(str string) (string, error)
	ExpandURL(id string) (string, error)
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	shortener ShortenerService
	baseURL   string
	db        *sql.DB
	logger    *zap.SugaredLogger
}

func New(shortenerService ShortenerService, baseURL string, db *sql.DB, logger *zap.SugaredLogger) *Handler {
	return &Handler{shortener: shortenerService, baseURL: baseURL, db: db, logger: logger}
}

func (h *Handler) shortenWithBaseURL(originURL string) (string, error) {
	id, err := h.shortener.ShortenURL(originURL)

	if err != nil {
		return "", err
	}

	shortenedURL, err := url.JoinPath(h.baseURL, id)
	return shortenedURL, err
}

func (h *Handler) GetShortenURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	expanded, err := h.shortener.ExpandURL(id)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", expanded)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) PlainPostShortenURL(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortenedURL, err := h.shortenWithBaseURL(string(data))

	if errors.Is(err, shortener.ErrIDGenerationFailed) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(shortenedURL))

	if err != nil {
		h.logger.Error(err.Error())
	}
}

func (h *Handler) ApiPostShortenURL(w http.ResponseWriter, r *http.Request) {
	var req model.Request
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortenedURL, err := h.shortenWithBaseURL(req.URL)

	if errors.Is(err, shortener.ErrIDGenerationFailed) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := model.Response{Result: shortenedURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(&resp)

	if err != nil {
		h.logger.Error(err.Error())
	}
}

func (h *Handler) Ping(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
