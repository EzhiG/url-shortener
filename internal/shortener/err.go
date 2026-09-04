package shortener

import (
	"errors"
	"fmt"
)

var (
	ErrIDCollision        = errors.New("id collision detected")
	ErrIDGenerationFailed = errors.New("id generation failed")
	ErrInvalidURL         = errors.New("invalid url")
	ErrURLNotFound        = errors.New("url not found")
)

type URLConflictError struct {
	ID string
}

func (e *URLConflictError) Error() string {
	return fmt.Sprintf("Original URL already exists: %s", e.ID)
}

func NewURLConflictError(id string) *URLConflictError {
	return &URLConflictError{ID: id}
}
