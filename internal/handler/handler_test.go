package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/EzhiG/url-shortener/internal/repository"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

const testBaseURL = "http://localhost:8080"
const exampleURL = "https://example.com"

func newTestHandler(storage *repository.MapStorage) *Handler {
	svc := shortener.New(storage)
	logger := zap.NewNop().Sugar()
	return New(svc, testBaseURL, logger)
}

type postWant struct {
	body        string
	contentType string
	status      int
	wantSaved   bool
}

func TestPlainPostShortenURL(t *testing.T) {
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

			h.PlainPostShortenURL(w, r)

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

func TestApiPostShortenURL(t *testing.T) {
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

			h.ApiPostShortenURL(w, r)

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

func TestGetShortenURL(t *testing.T) {
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
			id:        "unknownID",
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

			h.GetShortenURL(w, request)

			result := w.Result()
			defer result.Body.Close()

			require.Equal(t, tt.want.status, result.StatusCode)
			assert.Equal(t, tt.want.location, result.Header.Get("Location"))
		})
	}
}

func TestGzipMiddleware(t *testing.T) {
	requestBody := `{"url":"` + exampleURL + `"}`
	responseBody := `{"result":"` + testBaseURL + `/xyz"}`
	createServer := func(responseContentType string) *httptest.Server {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, requestBody, string(body))

			w.Header().Set("Content-Type", responseContentType)
			w.WriteHeader(http.StatusCreated)
			_, err = w.Write([]byte(responseBody))
			require.NoError(t, err)
		})

		srv := httptest.NewServer(GzipMiddleware(handler))
		return srv
	}

	tests := []struct {
		name                string
		responseContentType string
		gzipRequest         bool
		acceptGzip          bool
		wantGzipResponse    bool
	}{
		{
			name:                "plain request, plain response",
			responseContentType: "application/json",
			gzipRequest:         false,
			acceptGzip:          false,
			wantGzipResponse:    false,
		},
		{
			name:                "gzip request body is decompressed",
			responseContentType: "application/json",
			gzipRequest:         true,
			acceptGzip:          false,
			wantGzipResponse:    false,
		},
		{
			name:                "json content type gets compressed response",
			responseContentType: "application/json",
			gzipRequest:         false,
			acceptGzip:          true,
			wantGzipResponse:    true,
		},
		{
			name:                "html content type gets compressed response",
			responseContentType: "text/html",
			gzipRequest:         false,
			acceptGzip:          true,
			wantGzipResponse:    true,
		},
		{
			name:                "both gzip request and gzip response",
			responseContentType: "application/json",
			gzipRequest:         true,
			acceptGzip:          true,
			wantGzipResponse:    true,
		},
		{
			name:                "unsupported content type isn't compressed",
			responseContentType: "text/plain",
			gzipRequest:         false,
			acceptGzip:          true,
			wantGzipResponse:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := createServer(tt.responseContentType)
			defer srv.Close()

			var body io.Reader
			if tt.gzipRequest {
				buf := bytes.NewBuffer(nil)
				zb := gzip.NewWriter(buf)
				_, err := zb.Write([]byte(requestBody))
				require.NoError(t, err)
				require.NoError(t, zb.Close())
				body = buf
			} else {
				body = strings.NewReader(requestBody)
			}

			r := httptest.NewRequest(http.MethodPost, srv.URL, body)
			r.RequestURI = ""

			if tt.gzipRequest {
				r.Header.Set("Content-Encoding", "gzip")
			}
			if tt.acceptGzip {
				r.Header.Set("Accept-Encoding", "gzip")
			}

			resp, err := http.DefaultClient.Do(r)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode)
			require.Equal(t, tt.responseContentType, resp.Header.Get("Content-Type"))

			if tt.wantGzipResponse {
				require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
			} else {
				require.Empty(t, resp.Header.Get("Content-Encoding"))
			}

			reader := resp.Body
			if tt.wantGzipResponse {
				zr, err := gzip.NewReader(resp.Body)
				require.NoError(t, err)
				reader = zr
			}

			b, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.JSONEq(t, responseBody, string(b))
		})
	}
}
