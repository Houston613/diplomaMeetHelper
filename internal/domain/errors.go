package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidUserID       = errors.New("user id cannot be empty")
	ErrDatabaseUnavailable = errors.New("database temporarily unavailable")

	ErrMeetingNotFound   = errors.New("meeting not found")
	ErrAccessDenied      = errors.New("access denied: resource belongs to another user")
	ErrJobNotFound       = errors.New("job not found")
	ErrJobNotCompleted   = errors.New("meeting processing is not completed")
	ErrJobNotRetryable   = errors.New("job is not in failed status and cannot be retried")
	ErrFileNotFound      = errors.New("file not found")
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrQueueFull         = errors.New("worker pool queue is full")
	ErrInvalidMeetingID  = errors.New("invalid meeting id")

	ErrEmptySearchQuery = errors.New("search query cannot be empty")
	ErrEmptyQuestion    = errors.New("question cannot be empty")
)
