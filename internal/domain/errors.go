package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidUserID       = errors.New("user id cannot be empty")
	ErrDatabaseUnavailable = errors.New("database unavailable")
)
