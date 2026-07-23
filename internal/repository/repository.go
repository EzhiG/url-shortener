package repository

type Storage interface {
	Save(id, url string)
	Get(id string) (string, bool)
}

type MapStorage struct {
	data map[string]string
}

func NewMapStorage() *MapStorage {
	return &MapStorage{data: make(map[string]string)}
}

func (s *MapStorage) Save(id, url string) {
	s.data[id] = url
}

func (s *MapStorage) Get(id string) (string, bool) {
	url, ok := s.data[id]
	return url, ok
}
