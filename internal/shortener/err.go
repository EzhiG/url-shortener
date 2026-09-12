package shortener

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrIDCollision        = errors.New("id collision detected")
	ErrIDGenerationFailed = errors.New("id generation failed")
	ErrInvalidURL         = errors.New("invalid url")
	ErrURLNotFound        = errors.New("url not found")
)

type URLConflictItem struct {
	ShortURL    string
	OriginalURL string
}

type URLConflictError struct {
	Items []URLConflictItem
}

func (e *URLConflictError) Error() string {
	urls := make([]string, 0, len(e.Items))
	for _, item := range e.Items {
		urls = append(urls, item.ShortURL+"->"+item.OriginalURL)
	}
	return fmt.Sprintf("Original URLs already exists: %s", strings.Join(urls, ", "))
}

func NewURLConflictError(items []URLConflictItem) *URLConflictError {
	return &URLConflictError{items}
}
