package blog

import (
	"context"
	"time"

	"github.com/canakyuz/keystone/internal/domain/blog"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/google/uuid"
)

type PostService struct {
	repo   blog.PostRepository
	logger logger.Logger
}

func NewPostService(repo blog.PostRepository, logger logger.Logger) *PostService {
	return &PostService{
		repo:   repo,
		logger: logger,
	}
}

func (s *PostService) Create(ctx context.Context, tenantID, userID string, req CreatePostRequest) (*PostResponse, error) {
	var categoryID, content, excerpt, image string
	if req.CategoryID != nil {
		categoryID = *req.CategoryID
	}
	if req.Content != nil {
		content = *req.Content
	}
	if req.Excerpt != nil {
		excerpt = *req.Excerpt
	}
	if req.Image != nil {
		image = *req.Image
	}

	post := &blog.Post{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		CategoryID:  categoryID,
		Title:       req.Title,
		Slug:        req.Slug,
		Content:     content,
		Excerpt:     excerpt,
		Status:      blog.PostStatus(req.Status),
		Featured:    req.Featured,
		ViewCount:   0,
		Image:       image,
		Tags:        req.Tags,
		PublishedAt: req.PublishedAt,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if req.Metadata == nil {
		post.Metadata = make(map[string]interface{})
	}
	if req.Tags == nil {
		post.Tags = []string{}
	}

	if err := s.repo.Create(ctx, post); err != nil {
		s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err.Error()}).Error("Failed to create post")
		return nil, err
	}

	s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "post_id": post.ID}).Info("Post created successfully")
	resp := ToPostResponse(post)
	return &resp, nil
}

func (s *PostService) GetByID(ctx context.Context, id string) (*PostResponse, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := ToPostResponse(post)
	return &resp, nil
}

func (s *PostService) GetBySlug(ctx context.Context, slug string) (*PostResponse, error) {
	post, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	resp := ToPostResponse(post)
	return &resp, nil
}

func (s *PostService) List(ctx context.Context, tenantID string, categoryID, status *string, featured *bool, page, limit int) (*PostListResponse, error) {
	offset := (page - 1) * limit

	filters := blog.PostListFilters{
		CategoryID: categoryID,
		Featured:   featured,
		Limit:      limit,
		Offset:     offset,
	}

	if status != nil {
		st := blog.PostStatus(*status)
		filters.Status = &st
	}

	posts, total, err := s.repo.List(ctx, filters)
	if err != nil {
		s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err.Error()}).Error("Failed to list posts")
		return nil, err
	}

	data := make([]PostResponse, len(posts))
	for i, post := range posts {
		data[i] = ToPostResponse(post)
	}

	return &PostListResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *PostService) Update(ctx context.Context, id, userID string, req UpdatePostRequest) (*PostResponse, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var categoryID, content, excerpt, image string
	if req.CategoryID != nil {
		categoryID = *req.CategoryID
	}
	if req.Content != nil {
		content = *req.Content
	}
	if req.Excerpt != nil {
		excerpt = *req.Excerpt
	}
	if req.Image != nil {
		image = *req.Image
	}

	post.CategoryID = categoryID
	post.Title = req.Title
	post.Slug = req.Slug
	post.Content = content
	post.Excerpt = excerpt
	post.Status = blog.PostStatus(req.Status)
	post.Featured = req.Featured
	post.Image = image
	post.Tags = req.Tags
	post.PublishedAt = req.PublishedAt
	post.Metadata = req.Metadata
	post.UpdatedAt = time.Now()
	post.UpdatedBy = userID

	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.WithFields(logger.Fields{"post_id": id, "error": err.Error()}).Error("Failed to update post")
		return nil, err
	}

	s.logger.WithFields(logger.Fields{"post_id": id}).Info("Post updated successfully")
	resp := ToPostResponse(post)
	return &resp, nil
}

func (s *PostService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{"post_id": id, "error": err.Error()}).Error("Failed to delete post")
		return err
	}

	s.logger.WithFields(logger.Fields{"post_id": id}).Info("Post deleted successfully")
	return nil
}

func (s *PostService) Publish(ctx context.Context, id string) (*PostResponse, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	post.Publish()

	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.WithFields(logger.Fields{"post_id": id, "error": err.Error()}).Error("Failed to publish post")
		return nil, err
	}

	s.logger.WithFields(logger.Fields{"post_id": id}).Info("Post published successfully")
	resp := ToPostResponse(post)
	return &resp, nil
}

func (s *PostService) Archive(ctx context.Context, id string) (*PostResponse, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	post.Archive()

	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.WithFields(logger.Fields{"post_id": id, "error": err.Error()}).Error("Failed to archive post")
		return nil, err
	}

	s.logger.WithFields(logger.Fields{"post_id": id}).Info("Post archived successfully")
	resp := ToPostResponse(post)
	return &resp, nil
}

func (s *PostService) GetByTag(ctx context.Context, tag string, page, limit int) (*PostListResponse, error) {
	offset := (page - 1) * limit

	posts, total, err := s.repo.GetByTag(ctx, tag, limit, offset)
	if err != nil {
		s.logger.WithFields(logger.Fields{"tag": tag, "error": err.Error()}).Error("Failed to get posts by tag")
		return nil, err
	}

	data := make([]PostResponse, len(posts))
	for i, post := range posts {
		data[i] = ToPostResponse(post)
	}

	return &PostListResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *PostService) GetFeatured(ctx context.Context, page, limit int) (*PostListResponse, error) {
	posts, err := s.repo.GetFeatured(ctx, limit)
	if err != nil {
		s.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to get featured posts")
		return nil, err
	}

	data := make([]PostResponse, len(posts))
	for i, post := range posts {
		data[i] = ToPostResponse(post)
	}

	return &PostListResponse{
		Data:  data,
		Total: int64(len(posts)),
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *PostService) IncrementViewCount(ctx context.Context, id string) error {
	return s.repo.IncrementViewCount(ctx, id)
}

func (s *PostService) GetByCategoryID(ctx context.Context, categoryID string, page, limit int) (*PostListResponse, error) {
	offset := (page - 1) * limit

	posts, total, err := s.repo.GetByCategory(ctx, categoryID, limit, offset)
	if err != nil {
		s.logger.WithFields(logger.Fields{"category_id": categoryID, "error": err.Error()}).Error("Failed to get posts by category")
		return nil, err
	}

	data := make([]PostResponse, len(posts))
	for i, post := range posts {
		data[i] = ToPostResponse(post)
	}

	return &PostListResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
