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
	"net/http"

	"github.com/gofiber/fiber/v2"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	opuc "github.com/canakyuz/keystone/internal/usecase/operation"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/tracing"
)

// Service is what the handler needs from the operation usecase. The interface sits on the
// consumer side.
type Service interface {
	ProvisionTenant(ctx context.Context, in opuc.ProvisionInput) (*oprepo.ProvisionResult, error)
	Get(ctx context.Context, id, subject string) (*domain.Operation, error)
}

// Handler serves the operation and tenant provisioning endpoints.
type Handler struct {
	svc Service
	log *logger.Logger
}

// New creates the handler.
func New(svc Service, log *logger.Logger) *Handler {
	return &Handler{svc: svc, log: log}
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

	result, err := h.svc.ProvisionTenant(c.UserContext(), opuc.ProvisionInput{
		Name:      req.Name,
		Slug:      req.Slug,
		Email:     req.Email,
		Plan:      req.Plan,
		Subject:   subjectID(c),
		RequestID: requestID(c),
		// Captured here rather than further in: this is the last point where the request's
		// own span is still current.
		TraceContext:   tracing.Marshal(c.UserContext()),
		IdempotencyKey: c.Get("Idempotency-Key"),
		Body:           c.Body(),
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
	op, err := h.svc.Get(c.UserContext(), c.Params("id"), subjectID(c))

	var invalid *opuc.ValidationError
	switch {
	case errors.As(err, &invalid):
		return problem(c, http.StatusBadRequest, invalid.Code, invalid.Message)
	case errors.Is(err, domain.ErrNotFound):
		return problem(c, http.StatusNotFound, "operation_not_found", "operation not found")
	case err != nil:
		return h.internal(c, err)
	}

	return c.JSON(toResponse(op))
}

// mapCreateError turns domain errors into HTTP statuses.
func (h *Handler) mapCreateError(c *fiber.Ctx, err error) error {
	var invalid *opuc.ValidationError
	switch {
	case errors.As(err, &invalid):
		return problem(c, http.StatusBadRequest, invalid.Code, invalid.Message)

	case errors.Is(err, opuc.ErrNotPlatformOperator):
		return problem(c, http.StatusForbidden, "platform_permission_required", "requires a platform permission")

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

// requestID returns the correlation id the logging middleware assigned to this request.
//
// It is stored on the operation so the worker, which runs minutes later in another
// process, can put it in its own log lines. Without it the two halves of a provisioning
// have to be matched by timestamp.
func requestID(c *fiber.Ctx) string {
	id, _ := c.Locals("correlation_id").(string)

	return id
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
