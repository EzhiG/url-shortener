package shortener

import (
	"errors"
	"math/rand/v2"
	"net/url"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const idLength = 8

type UrlStorage interface {
	Save(id, url string)
	Get(id string) (string, bool)
}

func generateId() string {
	var b strings.Builder
	for range idLength {
		b.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return b.String()
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
		return "", errors.New("invalid url")
	}

	id := generateId()
	s.storage.Save(id, parsed.String())

	return id, nil
}

func (s *Service) ExpandUrl(id string) (string, error) {
	val, ok := s.storage.Get(id)

	if !ok {
		return "", errors.New("not found")
	}

	return val, nil
}
