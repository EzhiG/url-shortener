package repository

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"sync"
)

type FileStorage struct {
	mapStorage *MapStorage
	file       *os.File
	fileMu     sync.Mutex
	nextID     int
}

func (s *FileStorage) Close() error {
	return s.file.Close()
}

func (s *FileStorage) Check() error {
	return nil
}

func NewFileStorage(fname string) (*FileStorage, error) {
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

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
		var record StorageRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return err
		}

		s.mapStorage.set(record.ShortURL, record.OriginalURL)
		s.nextID++
	}

	return nil
}

func (s *FileStorage) Save(id, url string) error {
	if err := s.mapStorage.Save(id, url); err != nil {
		return err
	}

	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	s.nextID++
	record := StorageRecord{UUID: strconv.Itoa(s.nextID), ShortURL: id, OriginalURL: url}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	_, err = s.file.Write(append(data, '\n'))

	if err != nil {
		return err
	}

	return nil
}

func (s *FileStorage) Get(id string) (string, bool) {
	return s.mapStorage.Get(id)
}

func (s *FileStorage) CloseFile() error {
	return s.file.Close()
}
