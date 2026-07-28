package errs

import "errors"

var (
	ErrIdCollision        = errors.New("id collision detected")
	ErrIdGenerationFailed = errors.New("id generation failed")
	ErrInvalidUrl         = errors.New("invalid url")
	ErrUrlNotFound        = errors.New("url not found")
)
