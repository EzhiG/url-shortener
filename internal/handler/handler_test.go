package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

const testBaseURL = "http://localhost:8080"
const exampleURL = "https://example.com"

func newTestHandler(storage *repository.MapStorage) *Handler {
	svc := shortener.New(storage)
	return New(svc, testBaseURL)
}

type postWant struct {
	body        string
	contentType string
	status      int
	wantSaved   bool
}

func TestPlainPostShortenUrl(t *testing.T) {
	tests := []struct {
		name string
		body string
		want postWant
	}{
		{
			name: "valid url",
			body: exampleURL,
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

			h.PlainPostShortenUrl(w, r)

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

func TestApiPostShortenUrl(t *testing.T) {
	tests := []struct {
		name string
		body string
		want postWant
	}{
		{
			name: "valid url",
			body: `{"url":"` + exampleURL + `"}`,
			want: postWant{contentType: "application/json", status: http.StatusCreated, wantSaved: true},
		},
		{
			name: "invalid url",
			body: `{"url":"ftp://example.com"}`,
			want: postWant{contentType: "", status: http.StatusBadRequest, wantSaved: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMapStorage()
			h := newTestHandler(storage)

			r := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.ApiPostShortenUrl(w, r)

			result := w.Result()
			defer result.Body.Close()

			require.Equal(t, tt.want.status, result.StatusCode)
			require.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			if !tt.want.wantSaved {
				return
			}

			var res model.Response
			require.NoError(t, json.NewDecoder(result.Body).Decode(&res))

			id := strings.TrimPrefix(res.Result, testBaseURL+"/")
			saved, ok := storage.Get(id)
			require.True(t, ok)
			assert.Equal(t, exampleURL, saved)
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
			storedURL: exampleURL,
			want:      getWant{status: http.StatusTemporaryRedirect, location: exampleURL},
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
