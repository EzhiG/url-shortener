package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/EzhiG/url-shortener/internal/shortener"
	"go.uber.org/zap"
)

type ShortenerService interface {
	ShortenURL(original string) (string, error)
	ShortenManyURLs(original []string) (map[string]string, error)
	ExpandURL(id string) (string, error)
	Ping() error
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	shortener ShortenerService
	baseURL   string
	logger    *zap.SugaredLogger
}

func New(shortenerService ShortenerService, baseURL string, logger *zap.SugaredLogger) *Handler {
	return &Handler{shortener: shortenerService, baseURL: baseURL, logger: logger}
}

func (h *Handler) shortenWithBaseURL(originURL string) (string, error) {
	id, err := h.shortener.ShortenURL(originURL)

	var conflictError *shortener.URLConflictError
	if errors.As(err, &conflictError) {
		id = conflictError.ID
	} else if err != nil {
		return "", err
	}

	shortenedURL, joinErr := url.JoinPath(h.baseURL, id)
	if joinErr != nil {
		return "", joinErr
	}

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

	var conflictError *shortener.URLConflictError
	if errors.As(err, &conflictError) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusConflict)
	} else if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
	}

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

	var conflictError *shortener.URLConflictError
	if errors.As(err, &conflictError) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
	} else if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
	}

	resp := model.Response{Result: shortenedURL}
	err = json.NewEncoder(w).Encode(&resp)

	if err != nil {
		h.logger.Error(err.Error())
	}
}

func (h *Handler) ApiPostBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	var req model.BatchRequest
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	urls := make([]string, 0, len(req))
	for _, r := range req {
		urls = append(urls, r.OriginalURL)
	}

	records, err := h.shortener.ShortenManyURLs(urls)

	if err != nil {
		if errors.Is(err, shortener.ErrIDGenerationFailed) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := model.BatchResponse{}

	for _, r := range req {
		shortURL, ok := records[r.OriginalURL]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		shortURL, err = url.JoinPath(h.baseURL, shortURL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resp = append(resp, model.BatchResponseItem{CorrelationID: r.CorrelationID, ShortURL: shortURL})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(&resp)

	if err != nil {
		h.logger.Error(err.Error())
	}
}

func (h *Handler) Ping(w http.ResponseWriter, _ *http.Request) {
	if err := h.shortener.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
