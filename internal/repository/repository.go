package repository

import "sync"

type MapStorage struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMapStorage() *MapStorage {
	return &MapStorage{data: make(map[string]string)}
}

func (s *MapStorage) Save(id, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = val
}

func (s *MapStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.data[id]
	return val, ok
}
