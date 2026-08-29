package shortener

import (
	"errors"
	"net/url"
)

const maxAttempts = 5

var (
	ErrIDCollision        = errors.New("id collision detected")
	ErrIDGenerationFailed = errors.New("id generation failed")
	ErrInvalidURL         = errors.New("invalid url")
	ErrURLNotFound        = errors.New("url not found")
)

type URLStorage interface {
	Save(id, url string) error
	Get(id string) (string, bool)
	Check() error
}

type Service struct {
	storage URLStorage
}

func New(storage URLStorage) *Service {
	return &Service{storage: storage}
}

func (s *Service) ShortenURL(str string) (string, error) {
	parsed, err := url.Parse(str)

	if (err != nil) || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrInvalidURL
	}

	parsedURL := parsed.String()

	for range maxAttempts {
		id := generateID()
		err := s.storage.Save(id, parsedURL)

		if errors.Is(err, ErrIDCollision) {
			continue
		}

		return id, err
	}

	return "", ErrIDGenerationFailed
}

func (s *Service) ExpandURL(id string) (string, error) {
	val, ok := s.storage.Get(id)

	if !ok {
		return "", ErrURLNotFound
	}

	return val, nil
}

func (s *Service) Ping() error {
	return s.storage.Check()
}
