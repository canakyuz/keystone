// Package operation provides the HTTP face of long-running work.
//
// Provisioning is not synchronous: the request accepts the work and returns an
// operation address. The client polls that address for the status.
//
// Why not synchronous: creating a schema and applying migrations can take seconds.
// Holding the request open leaves the client with no information about the work if it
// times out.
package operation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	"github.com/canakyuz/keystone/pkg/logger"
)

// maxIdempotencyKeyLength caps the key.
// The schema holds 255 characters; the limit is enforced here too so the caller gets
// a legible validation error instead of a database error.
const maxIdempotencyKeyLength = 255

// Store defines the behaviours the handler needs.
// The interface sits on the consumer side: the handler knows only the two methods it
// uses, not the whole repository.
type Store interface {
	CreateTenantProvision(ctx context.Context, req oprepo.ProvisionRequest) (*oprepo.ProvisionResult, error)
	GetOperation(ctx context.Context, id string) (*domain.Operation, error)
}

// Handler serves the operation and tenant provisioning endpoints.
type Handler struct {
	store Store
	log   *logger.Logger
}

// New creates the handler.
func New(store Store, log *logger.Logger) *Handler {
	return &Handler{store: store, log: log}
}

// CreateTenantRequest is a provisioning request.
type CreateTenantRequest struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Email string `json:"email"`
	Plan  string `json:"plan,omitempty"`
}

// OperationResponse is the representation of an operation resource.
type OperationResponse struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// CreateTenant accepts the provisioning work.
//
// The response is 202 Accepted, and the Location header points at the operation
// resource. Returning 201 Created would be wrong: the tenant is not usable yet.
//
// A repeated request also returns 202 and points at the same operation. A client
// retry does not start a second provisioning.
func (h *Handler) CreateTenant(c *fiber.Ctx) error {
	var req CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return problem(c, http.StatusBadRequest, "invalid_body", "could not read request body")
	}

	if msg := validateCreate(req); msg != "" {
		return problem(c, http.StatusBadRequest, "validation_failed", msg)
	}

	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if len(key) > maxIdempotencyKeyLength {
		return problem(c, http.StatusBadRequest, "idempotency_key_too_long",
			fmt.Sprintf("Idempotency-Key en fazla %d karakter olabilir", maxIdempotencyKeyLength))
	}

	result, err := h.store.CreateTenantProvision(c.UserContext(), oprepo.ProvisionRequest{
		Name:      req.Name,
		Slug:      req.Slug,
		Email:     req.Email,
		Plan:      req.Plan,
		CreatedBy: subjectID(c),
		// The scope is bound to the subject making the request. One customer's key must not
		// match another customer's request.
		Scope:          "subject:" + subjectID(c),
		IdempotencyKey: key,
		RequestBody:    c.Body(),
	})
	if err != nil {
		return h.mapCreateError(c, err)
	}

	location := "/api/v1/operations/" + result.Operation.ID
	c.Set("Location", location)

	// A replay is flagged explicitly. The client must be able to see that its request
	// did not start new work.
	if result.Replayed {
		c.Set("Idempotent-Replay", "true")
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"operation": toResponse(result.Operation),
		"tenant_id": result.TenantID,
	})
}

// GetOperation returns the operation status.
func (h *Handler) GetOperation(c *fiber.Ctx) error {
	op, err := h.store.GetOperation(c.UserContext(), c.Params("id"))

	switch {
	case errors.Is(err, domain.ErrNotFound):
		return problem(c, http.StatusNotFound, "operation_not_found", "operation not found")
	case err != nil:
		return h.internal(c, err)
	}

	return c.JSON(toResponse(op))
}

// mapCreateError turns domain errors into HTTP statuses.
func (h *Handler) mapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrIdempotencyConflict):
		// 409: same key, different body. Silently returning the old result would make
		// the client believe a request it never sent had been processed.
		return problem(c, http.StatusConflict, "idempotency_key_reused",
			"this Idempotency-Key was used with a different request body")

	case errors.Is(err, oprepo.ErrSlugTaken):
		return problem(c, http.StatusConflict, "slug_taken", "slug already taken")

	default:
		return h.internal(c, err)
	}
}

// internal logs an unexpected error and returns a generic response.
// No implementation detail leaks to the client.
func (h *Handler) internal(c *fiber.Ctx, err error) error {
	if h.log != nil {
		h.log.WithFields(logger.Fields{
			"path":  c.Path(),
			"error": err.Error(),
		}).Error("request failed")
	}

	return problem(c, http.StatusInternalServerError, "internal_error", "request failed")
}

// validateCreate checks the required fields.
func validateCreate(req CreateTenantRequest) string {
	switch {
	case strings.TrimSpace(req.Name) == "":
		return "name zorunlu"
	case strings.TrimSpace(req.Slug) == "":
		return "slug zorunlu"
	case strings.TrimSpace(req.Email) == "":
		return "email zorunlu"
	}

	return ""
}

// subjectID returns the identity of the subject making the request.
func subjectID(c *fiber.Ctx) string {
	id, _ := c.Locals("user_id").(string)
	return id
}

// toResponse converts the domain object into its API representation.
func toResponse(op *domain.Operation) OperationResponse {
	resp := OperationResponse{
		ID:          op.ID,
		TenantID:    op.TenantID,
		Kind:        string(op.Kind),
		Status:      string(op.Status),
		ErrorCode:   op.ErrorCode,
		ErrorDetail: op.ErrorMessage,
		CreatedAt:   op.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	if op.CompletedAt != nil {
		resp.CompletedAt = op.CompletedAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	return resp
}

// problem returns a consistent error body.
// The machine-readable code keeps clients from branching on the message text.
func problem(c *fiber.Ctx, status int, code, detail string) error {
	return c.Status(status).JSON(fiber.Map{
		"type":   "https://keystone.dev/problems/" + code,
		"code":   code,
		"status": status,
		"detail": detail,
	})
}
