package shortener

import (
	"errors"
	"math/rand/v2"
	"net/url"

	"github.com/EzhiG/url-shortener/internal/errs"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const alphabetLen = len(alphabet)
const idLength = 8
const maxAttempts = 5

type UrlStorage interface {
	Save(id, url string) error
	Get(id string) (string, bool)
}

func generateId() string {
	b := make([]byte, idLength)
	for i := range idLength {
		b[i] = alphabet[rand.IntN(alphabetLen)]
	}

	return string(b)
}

type Service struct {
	storage UrlStorage
}

func NewService(storage UrlStorage) *Service {
	return &Service{storage: storage}
}

func (s *Service) ShortenUrl(str string) (string, error) {
	parsed, err := url.Parse(str)

	if (err != nil) || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errs.ErrInvalidUrl
	}

	parsedUrl := parsed.String()

	for range maxAttempts {
		id := generateId()
		err := s.storage.Save(id, parsedUrl)

		if errors.Is(err, errs.ErrIdCollision) {
			continue
		}

		return id, err
	}

	return "", errs.ErrIdGenerationFailed
}

func (s *Service) ExpandUrl(id string) (string, error) {
	val, ok := s.storage.Get(id)

	if !ok {
		return "", errs.ErrUrlNotFound
	}

	return val, nil
}
