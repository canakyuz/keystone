package blog

import "github.com/canakyuz/keystone/pkg/clienterr"

var (
	ErrPostNotFound     = clienterr.New("post not found")
	ErrCategoryNotFound = clienterr.New("category not found")
	ErrSlugExists       = clienterr.New("slug already exists")
)
