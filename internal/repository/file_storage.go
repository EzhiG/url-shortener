package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strconv"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/EzhiG/url-shortener/internal/shortener"
)

type FileStorageRecord struct {
	model.URLRecord
	UUID string `json:"uuid"`
}

type FileStorage struct {
	mapStorage *MapStorage
	file       *os.File
	nextID     int
}

func (s *FileStorage) Close() error {
	return s.file.Close()
}

func (s *FileStorage) Check() error {
	return nil
}

func NewFileStorage(name string) (*FileStorage, error) {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return nil, err
	}
	fileStorage := &FileStorage{file: file, mapStorage: NewMapStorage()}

	if err = fileStorage.restore(); err != nil {
		return nil, err
	}

	return fileStorage, nil
}

func (s *FileStorage) restore() error {
	scanner := bufio.NewScanner(s.file)

	for scanner.Scan() {
		var record FileStorageRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return err
		}

		s.mapStorage.set(record.ShortURL, record.OriginalURL, record.UserID)
		s.nextID++
	}

	return nil
}

func (s *FileStorage) Save(id, url, userID string) error {
	s.mapStorage.mu.Lock()
	defer s.mapStorage.mu.Unlock()

	if err := s.mapStorage.checkConflictUnlocked(id, url, userID); err != nil {
		return err
	}

	s.nextID++
	record := FileStorageRecord{
		UUID:      strconv.Itoa(s.nextID),
		URLRecord: model.URLRecord{OriginalURL: url, UserID: userID, ShortURL: id},
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	_, err = s.file.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	s.mapStorage.setUnlocked(id, url, userID)
	return nil
}

func (s *FileStorage) SaveMany(records map[string]string, userID string) error {
	s.mapStorage.mu.Lock()
	defer s.mapStorage.mu.Unlock()

	conflictOriginals := make(map[string]bool, len(records))
	hasConflicts := false

	err := s.mapStorage.checkManyConflictsUnlocked(records, userID)
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

	var data []byte
	for id, url := range records {
		if conflictOriginals[url] {
			continue
		}

		s.nextID++
		record := FileStorageRecord{
			UUID:      strconv.Itoa(s.nextID),
			URLRecord: model.URLRecord{UserID: userID, OriginalURL: url, ShortURL: id},
		}
		chunk, err := json.Marshal(record)
		if err != nil {
			return err
		}
		data = append(data, chunk...)
		data = append(data, '\n')
	}

	if len(data) > 0 {
		if _, err := s.file.Write(data); err != nil {
			return err
		}
	}

	for id, url := range records {
		if conflictOriginals[url] {
			continue
		}

		s.mapStorage.setUnlocked(id, url, userID)
	}

	if hasConflicts {
		return conflictErr
	}

	return nil
}

func (s *FileStorage) Get(id string) (model.URLRecord, bool) {
	return s.mapStorage.Get(id)
}

func (s *FileStorage) GetByUserID(userID string) ([]model.URLRecord, error) {
	return s.mapStorage.GetByUserID(userID)
}

func (s *FileStorage) CloseFile() error {
	return s.file.Close()
}
