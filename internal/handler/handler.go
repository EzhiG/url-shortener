package handler

import (
	"context"
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
	ShortenURL(original, userID string) (string, error)
	ShortenManyURLs(original []string, userID string) (map[string]string, error)
	ExpandURL(id string) (string, error)
	GetURLsByUserID(userID string) ([]model.URLRecord, error)
	Ping() error
}

type AuthService interface {
	WithUserID(ctx context.Context, userID string) context.Context
	UserIDFromContext(ctx context.Context) string
	IsTokenExpiredOnlyError(err error) bool
	GenerateUserID() (string, error)
	BuildToken(userID string) (string, error)
	ParseToken(tokenString string) (string, error)
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	shortener ShortenerService
	auth      AuthService
	baseURL   string
	logger    *zap.SugaredLogger
}

func New(auth AuthService, shortenerService ShortenerService, baseURL string, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		auth:      auth,
		shortener: shortenerService,
		baseURL:   baseURL,
		logger:    logger,
	}
}

func (h *Handler) shortenWithBaseURL(originURL, userID string) (string, error) {
	id, err := h.shortener.ShortenURL(originURL, userID)

	var conflictError *shortener.URLConflictError
	if errors.As(err, &conflictError) {
		id = conflictError.Items[0].ShortURL
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

func (h *Handler) APIGetUserURLs(w http.ResponseWriter, r *http.Request) {
	token := GetAuthCookie(r)
	if token == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := h.auth.ParseToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	urls, err := h.shortener.GetURLsByUserID(userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make(model.URLsResponse, len(urls))
	for i, urlItem := range urls {
		shortenedURL, err := url.JoinPath(h.baseURL, urlItem.ShortURL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp[i] = model.URLResponseItem{ShortURL: shortenedURL, OriginalURL: urlItem.OriginalURL}
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&resp)

	if err != nil {
		h.logger.Error(err.Error())
	}
}

func (h *Handler) PlainPostShortenURL(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := h.auth.UserIDFromContext(r.Context())
	shortenedURL, err := h.shortenWithBaseURL(string(data), userID)

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

func (h *Handler) APIPostShortenURL(w http.ResponseWriter, r *http.Request) {
	var req model.Request
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := h.auth.UserIDFromContext(r.Context())
	shortenedURL, err := h.shortenWithBaseURL(req.URL, userID)

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

func (h *Handler) APIPostBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	var req model.BatchRequest
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	urls := make([]string, 0, len(req))
	for _, item := range req {
		urls = append(urls, item.OriginalURL)
	}
	userID := h.auth.UserIDFromContext(r.Context())
	records, err := h.shortener.ShortenManyURLs(urls, userID)

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
