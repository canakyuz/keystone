// Package webhook lets a tenant's administrators say where the platform should notify them.
package webhook

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/canakyuz/keystone/internal/middleware"
	webhookuc "github.com/canakyuz/keystone/internal/usecase/webhook"
)

// Service is what the handler needs from the webhook usecase. The interface sits here, on
// the consumer side.
type Service interface {
	List(ctx context.Context, tenantID string) ([]webhookuc.Endpoint, error)
	Create(ctx context.Context, tenantID, rawURL string) (*webhookuc.Endpoint, string, error)
	SetActive(ctx context.Context, tenantID, id string, active bool) error
}

// Handler serves the webhook endpoint routes.
type Handler struct {
	svc Service
}

// NewHandler creates the handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Deliveries counts an endpoint's events by state.
type Deliveries struct {
	Delivered int `json:"delivered"`
	Pending   int `json:"pending"`
	Dead      int `json:"dead"`
}

// EndpointResponse is a destination as the API returns it. It never carries the secret.
type EndpointResponse struct {
	ID         string     `json:"id"`
	URL        string     `json:"url"`
	Active     bool       `json:"active"`
	SecretHint string     `json:"secret_hint"`
	CreatedAt  time.Time  `json:"created_at"`
	Deliveries Deliveries `json:"deliveries"`
}

// CreatedResponse is the one response that carries the secret. It is not stored anywhere a
// reader can fetch it again, so the client has to keep it now.
type CreatedResponse struct {
	EndpointResponse
	Secret string `json:"secret"`
}

// List returns the tenant's destinations.
// GET /api/v1/webhook-endpoints
func (h *Handler) List(c *fiber.Ctx) error {
	endpoints, err := h.svc.List(c.UserContext(), middleware.GetTenantID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not list the endpoints"})
	}

	out := make([]EndpointResponse, 0, len(endpoints))
	for _, e := range endpoints {
		out = append(out, toResponse(e))
	}

	return c.JSON(fiber.Map{"data": out})
}

// Create registers a destination and returns its signing secret, once.
// POST /api/v1/webhook-endpoints
func (h *Handler) Create(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	created, secret, err := h.svc.Create(c.UserContext(), middleware.GetTenantID(c), req.URL)

	var invalid *webhookuc.InvalidURLError
	switch {
	case errors.As(err, &invalid):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": invalid.Reason})
	case errors.Is(err, webhookuc.ErrLimitReached):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "this tenant already has the maximum number of endpoints; turn one off or reuse it",
		})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not register the endpoint"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": CreatedResponse{
		EndpointResponse: toResponse(*created),
		Secret:           secret,
	}})
}

// SetActive turns an endpoint on or off.
// PATCH /api/v1/webhook-endpoints/:id
func (h *Handler) SetActive(c *fiber.Ctx) error {
	id := c.Params("id")
	if uuid.Validate(id) != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "webhook endpoint not found"})
	}

	var req struct {
		Active *bool `json:"active"`
	}
	if err := c.BodyParser(&req); err != nil || req.Active == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "active is required"})
	}

	err := h.svc.SetActive(c.UserContext(), middleware.GetTenantID(c), id, *req.Active)

	switch {
	case errors.Is(err, webhookuc.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "webhook endpoint not found"})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not change the endpoint"})
	}

	return c.JSON(fiber.Map{"data": fiber.Map{"id": id, "active": *req.Active}})
}

func toResponse(e webhookuc.Endpoint) EndpointResponse {
	return EndpointResponse{
		ID:         e.ID,
		URL:        e.URL,
		Active:     e.Active,
		SecretHint: e.SecretHint,
		CreatedAt:  e.CreatedAt,
		Deliveries: Deliveries{Delivered: e.Delivered, Pending: e.Pending, Dead: e.Dead},
	}
}
