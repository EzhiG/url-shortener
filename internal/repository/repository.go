package repository

import (
	"errors"
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

func (s *MapStorage) checkConflictUnlocked(id, val string) error {
	if existingID, ok := s.origIdx[val]; ok {
		return shortener.NewURLConflictError([]shortener.URLConflictItem{{ShortURL: existingID, OriginalURL: val}})
	}

	if _, ok := s.data[id]; ok {
		return shortener.ErrIDCollision
	}

	return nil
}

func (s *MapStorage) checkManyConflictsUnlocked(records map[string]string) error {
	var conflicts []shortener.URLConflictItem

	for id, url := range records {
		err := s.checkConflictUnlocked(id, url)
		if err != nil {
			var conflictErr *shortener.URLConflictError
			if errors.As(err, &conflictErr) {
				conflicts = append(conflicts, conflictErr.Items...)
				continue
			}

			return err
		}
	}

	if len(conflicts) > 0 {
		return shortener.NewURLConflictError(conflicts)
	}

	return nil
}

func (s *MapStorage) setUnlocked(id, val string) {
	s.data[id] = val
	s.origIdx[val] = id
}

func (s *MapStorage) Save(id, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkConflictUnlocked(id, val); err != nil {
		return err
	}

	s.setUnlocked(id, val)
	return nil
}

func (s *MapStorage) SaveMany(records map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conflictOriginals := make(map[string]bool, len(records))
	hasConflicts := false

	err := s.checkManyConflictsUnlocked(records)
	var conflictErr *shortener.URLConflictError
	if err != nil {
		if errors.As(err, &conflictErr) {
			for _, conflictErrItem := range conflictErr.Items {
				hasConflicts = true
				conflictOriginals[conflictErrItem.OriginalURL] = true
			}
		} else {
			return err
		}
	}

	for id, url := range records {
		if conflictOriginals[url] {
			continue
		}

		s.setUnlocked(id, url)
	}

	if hasConflicts {
		return conflictErr
	}

	return nil
}

func (s *MapStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.data[id]
	return val, ok
}

func (s *MapStorage) set(id, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setUnlocked(id, val)
}
