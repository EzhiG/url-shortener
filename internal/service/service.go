package service

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

func ShortenUrl(str string) (string, error) {
	parsed, err := url.Parse(str)

	if (err != nil) || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("invalid url")
	}

	id := generateId()
	repository.Save(id, parsed.String())

	return id, nil
}

func ExpandUrl(id string) string {
	return repository.Get(id)
}
