package shortener

import (
	"errors"
	"math/rand/v2"
	"net/url"
	"strings"

	"github.com/EzhiG/url-shortener/internal/repository"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const idLength = 8

func generateId() string {
	var b strings.Builder
	for range idLength {
		b.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return b.String()
}

type Service struct {
	storage repository.Storage
}

func NewService(storage repository.Storage) *Service {
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

func (s *Service) ExpandUrl(id string) (string, bool) {
	return s.storage.Get(id)
}
