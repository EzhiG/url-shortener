package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

const testBaseURL = "http://localhost:8080"

func newTestHandler(storage *repository.MapStorage) *Handler {
	svc := shortener.NewService(storage)
	return NewHandler(svc, testBaseURL)
}

type postWant struct {
	body        string
	contentType string
	status      int
	wantSaved   bool
}

func TestPostShortenUrl(t *testing.T) {
	tests := []struct {
		name string
		body string
		want postWant
	}{
		{
			name: "valid url",
			body: "https://example.com",
			want: postWant{contentType: "text/plain", status: http.StatusCreated, wantSaved: true},
		},
		{
			name: "invalid url",
			body: "ftp://example.com",
			want: postWant{contentType: "", status: http.StatusBadRequest, wantSaved: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMapStorage()
			h := newTestHandler(storage)

			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.PostShortenUrl(w, r)

			result := w.Result()
			defer result.Body.Close()

			require.Equal(t, tt.want.status, result.StatusCode)
			require.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			if !tt.want.wantSaved {
				return
			}

			id := strings.TrimPrefix(string(body), testBaseURL+"/")
			saved, ok := storage.Get(id)
			require.True(t, ok)
			assert.Equal(t, tt.body, saved)
		})
	}
}

type getWant struct {
	status   int
	location string
}

func TestGetShortenUrl(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		storedURL string
		want      getWant
	}{
		{
			name:      "existing id",
			id:        "abc123",
			storedURL: "https://example.com",
			want:      getWant{status: http.StatusTemporaryRedirect, location: "https://example.com"},
		},
		{
			name:      "unknown id",
			id:        "unknownId",
			storedURL: "",
			want:      getWant{status: http.StatusBadRequest, location: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMapStorage()

			if tt.storedURL != "" {
				storage.Save(tt.id, tt.storedURL)
			}

			h := newTestHandler(storage)
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.GetShortenUrl(w, request)

			result := w.Result()
			defer result.Body.Close()

			require.Equal(t, tt.want.status, result.StatusCode)
			assert.Equal(t, tt.want.location, result.Header.Get("Location"))
		})
	}
}
