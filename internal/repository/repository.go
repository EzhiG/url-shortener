package repository

import (
	"sync"

	"github.com/EzhiG/url-shortener/internal/shortener"
)

type MapStorage struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMapStorage() *MapStorage {
	return &MapStorage{data: make(map[string]string)}
}

func (s *MapStorage) Save(id, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; ok {
		return shortener.ErrIdCollision
	}

	s.data[id] = val
	return nil
}

func (s *MapStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.data[id]
	return val, ok
}
