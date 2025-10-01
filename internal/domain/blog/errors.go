package blog

import "errors"

var (
	ErrPostNotFound     = errors.New("post not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrSlugExists       = errors.New("slug already exists")
)
