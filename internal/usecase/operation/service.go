// Package operation holds the rules for accepting tenant provisioning and reading its
// progress, shared by the REST and gRPC surfaces.
//
// The rules used to live in each transport. They drifted: REST trimmed the required fields
// and capped the idempotency key at the column's 255 characters, gRPC did neither, so a
// name of spaces was accepted on one port and refused on the other, and an overlong key
// reached the database and came back as an internal error. Worse, the platform permission
// that tenant creation requires was a REST route middleware, so the gRPC port let any
// member of any tenant create tenants.
package operation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/canakyuz/keystone/internal/authz"
	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
)

// MaxIdempotencyKeyLength is the width of the column that stores the key. Checked here so
// the caller gets a validation error instead of a database error.
const MaxIdempotencyKeyLength = 255

// Codes a ValidationError carries. They are part of the API: REST returns them as the
// problem code, and clients branch on them rather than on the message.
const (
	CodeValidationFailed      = "validation_failed"
	CodeIdempotencyKeyTooLong = "idempotency_key_too_long"
)

// ValidationError is a request the caller has to change before retrying. Its message is
// safe to return to the caller.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(code, format string, args ...any) error {
	return &ValidationError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// ErrNotPlatformOperator refuses tenant creation to a subject without a platform_operators
// row. Creating a tenant acts outside the caller's own tenant, so no tenant role grants it.
var ErrNotPlatformOperator = errors.New("requires a platform permission")

// Store is what the service needs from the operation repository.
type Store interface {
	CreateTenantProvision(ctx context.Context, req oprepo.ProvisionRequest) (*oprepo.ProvisionResult, error)
	GetOperation(ctx context.Context, id, subject string) (*domain.Operation, error)
}

// ProvisionInput is a provisioning request as either transport receives it.
type ProvisionInput struct {
	Name, Slug, Email, Plan string

	// Subject is the authenticated caller. The idempotency scope is bound to it, so one
	// customer's key cannot match another customer's request.
	Subject string

	RequestID    string
	TraceContext string

	IdempotencyKey string

	// Body is what the idempotency fingerprint is taken over. Each transport supplies its
	// own; see ADR-0004 for why the REST one is the raw body.
	Body []byte
}

// Service accepts provisioning work and reports on it.
type Service struct {
	store     Store
	operators authz.PlatformLookup
}

// NewService creates the service.
func NewService(store Store, operators authz.PlatformLookup) *Service {
	return &Service{store: store, operators: operators}
}

// ProvisionTenant validates the request and records the provisioning work. A repeated
// idempotency key returns the existing operation with Replayed set.
func (s *Service) ProvisionTenant(ctx context.Context, in ProvisionInput) (*oprepo.ProvisionResult, error) {
	allowed, err := authz.IsPlatformOperator(ctx, s.operators, in.Subject)
	if err != nil {
		return nil, fmt.Errorf("could not verify the platform permission: %w", err)
	}
	if !allowed {
		return nil, ErrNotPlatformOperator
	}

	key := strings.TrimSpace(in.IdempotencyKey)
	if err := validate(in, key); err != nil {
		return nil, err
	}

	return s.store.CreateTenantProvision(ctx, oprepo.ProvisionRequest{
		Name:           in.Name,
		Slug:           in.Slug,
		Email:          in.Email,
		Plan:           in.Plan,
		CreatedBy:      in.Subject,
		RequestID:      in.RequestID,
		TraceContext:   in.TraceContext,
		Scope:          "subject:" + in.Subject,
		IdempotencyKey: key,
		RequestBody:    in.Body,
	})
}

// Get returns an operation the subject started. Anyone else's reads as not found.
func (s *Service) Get(ctx context.Context, id, subject string) (*domain.Operation, error) {
	if strings.TrimSpace(id) == "" {
		return nil, invalid(CodeValidationFailed, "id is required")
	}

	return s.store.GetOperation(ctx, id, subject)
}

func validate(in ProvisionInput, key string) error {
	switch {
	case strings.TrimSpace(in.Name) == "":
		return invalid(CodeValidationFailed, "name is required")
	case strings.TrimSpace(in.Slug) == "":
		return invalid(CodeValidationFailed, "slug is required")
	case strings.TrimSpace(in.Email) == "":
		return invalid(CodeValidationFailed, "email is required")
	case len(key) > MaxIdempotencyKeyLength:
		return invalid(CodeIdempotencyKeyTooLong, "Idempotency-Key may be at most %d characters", MaxIdempotencyKeyLength)
	}

	return nil
}
