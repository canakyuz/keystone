package website

import "errors"

var (
	// ErrNotFound is returned when a website is not found
	ErrNotFound = errors.New("website not found")

	// ErrSlugExists is returned when a slug already exists
	ErrSlugExists = errors.New("website slug already exists")

	// ErrCannotPublish is returned when a website cannot be published
	ErrCannotPublish = errors.New("website cannot be published: missing homepage or invalid status")

	// ErrAlreadyArchived is returned when trying to archive an already archived website
	ErrAlreadyArchived = errors.New("website is already archived")

	// ErrInvalidStatus is returned when an invalid status is provided
	ErrInvalidStatus = errors.New("invalid website status")

	// ErrUnauthorized is returned when a user doesn't have access to a website
	ErrUnauthorized = errors.New("unauthorized access to website")

	// ErrInvalidInput is returned for validation errors
	ErrInvalidInput = errors.New("invalid input")
)
