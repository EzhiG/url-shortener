package repository

import (
	"sync"

	"github.com/EzhiG/url-shortener/internal/shortener"
)

type StorageRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage interface {
	Save(id, url string) error
	SaveMany(records map[string]string) error
	Get(id string) (string, bool)
	Check() error
	Close() error
}
type MapStorage struct {
	mu      sync.Mutex
	origIdx map[string]string
	data    map[string]string
}

func (s *MapStorage) Close() error {
	return nil
}

func (s *MapStorage) Check() error {
	return nil
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:    make(map[string]string),
		origIdx: make(map[string]string),
	}
}

func (s *MapStorage) Save(id, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.origIdx[val]; ok {
		return shortener.NewURLConflictError(id)
	}

	if _, ok := s.data[id]; ok {
		return shortener.ErrIDCollision
	}

	s.set(id, val)
	return nil
}

func (s *MapStorage) SaveMany(records map[string]string) error {
	for id, url := range records {
		if err := s.Save(id, url); err != nil {
			return err
		}
	}

	return nil
}

func (s *MapStorage) removeMany(ids []string) {
	for _, id := range ids {
		s.remove(id)
	}
}

func (s *MapStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.data[id]
	return val, ok
}

func (s *MapStorage) set(id, val string) {
	s.data[id] = val
	s.origIdx[val] = id
}

func (s *MapStorage) remove(id string) {
	value := s.data[id]
	delete(s.data, id)
	delete(s.origIdx, value)
}
