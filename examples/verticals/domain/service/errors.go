package service

import "github.com/canakyuz/keystone/pkg/clienterr"

var (
	ErrServiceNotFound      = clienterr.New("service not found")
	ErrServiceAlreadyExists = clienterr.New("service with this slug already exists")
	ErrInvalidPrice         = clienterr.New("invalid price value")
)
