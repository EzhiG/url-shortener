package repository

import (
	"errors"
	"sync"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

type Storage interface {
	Save(id, url, userID string) error
	SaveMany(records map[string]string, userID string) error
	Get(id string) (model.URLRecord, bool)
	GetByUserID(userID string) ([]model.URLRecord, error)
	Check() error
	Close() error
}

type MapStorage struct {
	mu          sync.Mutex
	userOrigIdx map[string]string
	data        map[string]model.URLRecord
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:        make(map[string]model.URLRecord),
		userOrigIdx: make(map[string]string),
	}
}
func getIdxKey(url, userID string) string {
	return userID + "_" + url
}

func (s *MapStorage) Close() error {
	return nil
}

func (s *MapStorage) Check() error {
	return nil
}

func (s *MapStorage) checkConflictUnlocked(id, val, userID string) error {
	if existingID, ok := s.userOrigIdx[getIdxKey(val, userID)]; ok {
		return shortener.NewURLConflictError([]shortener.URLConflictItem{{ShortURL: existingID, OriginalURL: val}})
	}

	if _, ok := s.data[id]; ok {
		return shortener.ErrIDCollision
	}

	return nil
}

func (s *MapStorage) checkManyConflictsUnlocked(records map[string]string, userID string) error {
	var conflicts []shortener.URLConflictItem

	for id, url := range records {
		err := s.checkConflictUnlocked(id, url, userID)
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

func (s *MapStorage) setUnlocked(id, val, userID string) {
	s.data[id] = model.URLRecord{UserID: userID, OriginalURL: val, ShortURL: id}
	s.userOrigIdx[getIdxKey(val, userID)] = id
}

func (s *MapStorage) Save(id, val, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkConflictUnlocked(id, val, userID); err != nil {
		return err
	}

	s.setUnlocked(id, val, userID)
	return nil
}

func (s *MapStorage) SaveMany(records map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conflictOriginals := make(map[string]bool, len(records))
	hasConflicts := false

	err := s.checkManyConflictsUnlocked(records, userID)
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

		s.setUnlocked(id, url, userID)
	}

	if hasConflicts {
		return conflictErr
	}

	return nil
}

func (s *MapStorage) Get(id string) (model.URLRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.data[id]
	return record, ok
}

func (s *MapStorage) GetByUserID(userID string) ([]model.URLRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]model.URLRecord, 0)

	for _, record := range s.data {
		if record.UserID == userID {
			records = append(records, record)
		}
	}

	return records, nil
}

func (s *MapStorage) set(id, val, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setUnlocked(id, val, userID)
}
