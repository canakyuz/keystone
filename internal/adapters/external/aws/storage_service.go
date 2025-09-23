package aws

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// StorageService implements the storage service using AWS S3
type StorageService struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	region   string
}

// NewStorageService creates a new AWS S3 storage service
func NewStorageService(region, bucket string) (*StorageService, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)

	return &StorageService{
		client:   client,
		uploader: uploader,
		bucket:   bucket,
		region:   region,
	}, nil
}

// NewStorageServiceWithCredentials creates a new AWS S3 storage service with explicit credentials
func NewStorageServiceWithCredentials(region, bucket, accessKeyID, secretAccessKey string) (*StorageService, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)

	return &StorageService{
		client:   client,
		uploader: uploader,
		bucket:   bucket,
		region:   region,
	}, nil
}

// Upload uploads a file and returns the URL
func (s *StorageService) Upload(ctx context.Context, key string, content io.Reader, contentType string) (string, error) {
	// Ensure the key starts with proper tenant isolation
	if !strings.Contains(key, "/") {
		return "", fmt.Errorf("key must include tenant prefix for isolation")
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"uploaded-by": "nexspaces-api",
			"upload-date": time.Now().Format(time.RFC3339),
		},
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	}

	result, err := s.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return result.Location, nil
}

// Download downloads a file
func (s *StorageService) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}

	return result.Body, nil
}

// Delete deletes a file
func (s *StorageService) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// GetURL gets a pre-signed URL for a file
func (s *StorageService) GetURL(ctx context.Context, key string, expiry int) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiry) * time.Second
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// List lists files with a prefix
func (s *StorageService) List(ctx context.Context, prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(s.client, input)

	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list files from S3: %w", err)
		}

		for _, object := range result.Contents {
			if object.Key != nil {
				keys = append(keys, *object.Key)
			}
		}
	}

	return keys, nil
}

// GetFileInfo retrieves metadata about a file
func (s *StorageService) GetFileInfo(ctx context.Context, key string) (*FileInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info from S3: %w", err)
	}

	info := &FileInfo{
		Key:          key,
		Size:         *result.ContentLength,
		ContentType:  aws.ToString(result.ContentType),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         strings.Trim(aws.ToString(result.ETag), "\""),
		Metadata:     result.Metadata,
	}

	return info, nil
}

// CopyFile copies a file from one key to another
func (s *StorageService) CopyFile(ctx context.Context, sourceKey, destKey string) error {
	copySource := fmt.Sprintf("%s/%s", s.bucket, sourceKey)

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(destKey),
		Metadata: map[string]string{
			"copied-by":   "nexspaces-api",
			"copied-date": time.Now().Format(time.RFC3339),
		},
		MetadataDirective:    types.MetadataDirectiveReplace,
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	}

	_, err := s.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to copy file in S3: %w", err)
	}

	return nil
}

// DeleteMultiple deletes multiple files
func (s *StorageService) DeleteMultiple(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// S3 delete operation can handle up to 1000 keys at once
	const maxKeys = 1000
	for i := 0; i < len(keys); i += maxKeys {
		end := i + maxKeys
		if end > len(keys) {
			end = len(keys)
		}

		var objectIdentifiers []types.ObjectIdentifier
		for _, key := range keys[i:end] {
			objectIdentifiers = append(objectIdentifiers, types.ObjectIdentifier{
				Key: aws.String(key),
			})
		}

		input := &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(true),
			},
		}

		_, err := s.client.DeleteObjects(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to delete multiple files from S3: %w", err)
		}
	}

	return nil
}

// CreateTenantFolder creates a folder structure for a tenant
func (s *StorageService) CreateTenantFolder(ctx context.Context, tenantID string) error {
	// Create folder structure for tenant
	folders := []string{
		fmt.Sprintf("tenants/%s/", tenantID),
		fmt.Sprintf("tenants/%s/templates/", tenantID),
		fmt.Sprintf("tenants/%s/uploads/", tenantID),
		fmt.Sprintf("tenants/%s/backups/", tenantID),
		fmt.Sprintf("tenants/%s/exports/", tenantID),
	}

	for _, folder := range folders {
		// Create empty object to represent folder
		input := &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(folder + ".keep"), // .keep file to maintain folder structure
			Body:        strings.NewReader(""),
			ContentType: aws.String("text/plain"),
			Metadata: map[string]string{
				"purpose":      "folder-marker",
				"tenant-id":    tenantID,
				"created-by":   "nexspaces-api",
				"created-date": time.Now().Format(time.RFC3339),
			},
		}

		_, err := s.client.PutObject(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to create tenant folder %s: %w", folder, err)
		}
	}

	return nil
}

// DeleteTenantFolder deletes all files in a tenant's folder
func (s *StorageService) DeleteTenantFolder(ctx context.Context, tenantID string) error {
	prefix := fmt.Sprintf("tenants/%s/", tenantID)

	// List all objects with the tenant prefix
	keys, err := s.List(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list tenant files: %w", err)
	}

	if len(keys) == 0 {
		return nil // No files to delete
	}

	// Delete all tenant files
	err = s.DeleteMultiple(ctx, keys)
	if err != nil {
		return fmt.Errorf("failed to delete tenant files: %w", err)
	}

	return nil
}

// FileInfo represents file metadata
type FileInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	Metadata     map[string]string `json:"metadata"`
}
