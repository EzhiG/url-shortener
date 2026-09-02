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
	SaveMany(records map[string]string) error
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
	parsedURL, err := s.parse(str)
	if err != nil {
		return "", err
	}

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

func (s *Service) generateURLMap(originals []string) (map[string]string, error) {
	var records = make(map[string]string, len(originals))

	for _, originalURL := range originals {
		parsedURL, err := s.parse(originalURL)
		if err != nil {
			return nil, err
		}

		records[generateID()] = parsedURL
	}

	return records, nil
}

func (s *Service) ShortenManyURLs(originals []string) (map[string]string, error) {
	for range maxAttempts {
		records, err := s.generateURLMap(originals)

		if err != nil {
			return nil, err
		}

		err = s.storage.SaveMany(records)

		if errors.Is(err, ErrIDCollision) {
			continue
		}

		return s.swap(records), err
	}

	return nil, ErrIDGenerationFailed
}

func (s *Service) swap(records map[string]string) map[string]string {
	swapped := make(map[string]string, len(records))

	for key, value := range records {
		swapped[value] = key
	}

	return swapped
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

func (s *Service) parse(original string) (string, error) {
	parsed, err := url.Parse(original)
	if (err != nil) || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrInvalidURL
	}

	return parsed.String(), nil
}
