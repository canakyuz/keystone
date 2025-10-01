package service

import "errors"

var (
	ErrServiceNotFound      = errors.New("service not found")
	ErrServiceAlreadyExists = errors.New("service with this slug already exists")
	ErrInvalidPrice         = errors.New("invalid price value")
)
