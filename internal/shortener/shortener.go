package shortener

import (
	"errors"
	"net/url"
)

const maxAttempts = 5

var (
	ErrIdCollision        = errors.New("id collision detected")
	ErrIdGenerationFailed = errors.New("id generation failed")
	ErrInvalidUrl         = errors.New("invalid url")
	ErrUrlNotFound        = errors.New("url not found")
)

type UrlStorage interface {
	Save(id, url string) error
	Get(id string) (string, bool)
}

type Service struct {
	storage UrlStorage
}

func New(storage UrlStorage) *Service {
	return &Service{storage: storage}
}

func (s *Service) ShortenUrl(str string) (string, error) {
	parsed, err := url.Parse(str)

	if (err != nil) || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrInvalidUrl
	}

	parsedUrl := parsed.String()

	for range maxAttempts {
		id := generateId()
		err := s.storage.Save(id, parsedUrl)

		if errors.Is(err, ErrIdCollision) {
			continue
		}

		return id, err
	}

	return "", ErrIdGenerationFailed
}

func (s *Service) ExpandUrl(id string) (string, error) {
	val, ok := s.storage.Get(id)

	if !ok {
		return "", ErrUrlNotFound
	}

	return val, nil
}
