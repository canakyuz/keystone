package blog

import "time"

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusArchived  PostStatus = "archived"
)

type Post struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Title       string         `json:"title"`
	Slug        string         `json:"slug"`
	Content     string         `json:"content"`
	Excerpt     string         `json:"excerpt"`
	Status      PostStatus     `json:"status"`
	Featured    bool           `json:"featured"`
	ViewCount   int64          `json:"view_count"`
	Image       string         `json:"image,omitempty"`
	Tags        []string       `json:"tags"`
	CategoryID  string         `json:"category_id,omitempty"`
	PublishedAt *time.Time     `json:"published_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by"`
}

func (p *Post) Publish() {
	now := time.Now()
	p.Status = PostStatusPublished
	p.PublishedAt = &now
	p.UpdatedAt = now
}

func (p *Post) Archive() {
	p.Status = PostStatusArchived
	p.UpdatedAt = time.Now()
}

func (p *Post) IncrementViewCount() {
	p.ViewCount++
}
