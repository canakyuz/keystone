// Package webhook lets a tenant's administrators say where the platform should notify them.
package webhook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/canakyuz/keystone/internal/middleware"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
	webhookrepo "github.com/canakyuz/keystone/internal/repository/webhook"
	"github.com/canakyuz/keystone/pkg/outbound"
)

// Store is what the handler needs from the repository. The interface sits here, on the
// consumer side.
type Store interface {
	Create(ctx context.Context, tenantID, url, secret string, entry auditrepo.Entry) (*webhookrepo.Endpoint, error)
	List(ctx context.Context, tenantID string) ([]webhookrepo.Endpoint, error)
	SetActive(ctx context.Context, tenantID, id string, active bool, entry auditrepo.Entry) error
}

// Handler serves the webhook endpoint routes.
type Handler struct {
	store Store
}

// NewHandler creates the handler.
func NewHandler(store Store) *Handler {
	return &Handler{store: store}
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
	endpoints, err := h.store.List(c.UserContext(), middleware.GetTenantID(c))
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

	destination := strings.TrimSpace(req.URL)

	// The cheap check, which gives the administrator a message they can act on. The
	// security boundary is the dialler at delivery time; see pkg/outbound.
	if err := outbound.ValidateURL(destination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": strings.TrimPrefix(err.Error(), "outbound: "),
		})
	}

	secret, err := newSecret()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create a secret"})
	}

	created, err := h.store.Create(c.UserContext(), middleware.GetTenantID(c), destination, secret,
		entry(c.UserContext(), middleware.GetTenantID(c), "webhook.created",
			// The host, not the url. Some services put the secret in the path of the url they
			// hand out, and the trail keeps whatever it is given forever.
			map[string]any{"host": hostOf(destination)}))

	switch {
	case errors.Is(err, webhookrepo.ErrLimitReached):
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

	action := "webhook.disabled"
	if *req.Active {
		action = "webhook.enabled"
	}

	tenantID := middleware.GetTenantID(c)
	err := h.store.SetActive(c.UserContext(), tenantID, id, *req.Active, entry(c.UserContext(), tenantID, action, nil))

	switch {
	case errors.Is(err, webhookrepo.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "webhook endpoint not found"})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not change the endpoint"})
	}

	return c.JSON(fiber.Map{"data": fiber.Map{"id": id, "active": *req.Active}})
}

func toResponse(e webhookrepo.Endpoint) EndpointResponse {
	return EndpointResponse{
		ID:         e.ID,
		URL:        e.URL,
		Active:     e.Active,
		SecretHint: e.SecretHint,
		CreatedAt:  e.CreatedAt,
		Deliveries: Deliveries{Delivered: e.Delivered, Pending: e.Pending, Dead: e.Dead},
	}
}

// entry builds the audit line. The subject id is filled in by the repository, which is the
// only place that knows it before the transaction commits.
func entry(ctx context.Context, tenantID, action string, metadata map[string]any) auditrepo.Entry {
	actorID, actorType := auditrepo.ActorFrom(ctx)

	return auditrepo.Entry{
		TenantID:    tenantID,
		ActorID:     actorID,
		ActorType:   actorType,
		Action:      action,
		SubjectType: "webhook_endpoint",
		Metadata:    metadata,
	}
}

// newSecret returns 32 random bytes as hex. The deliveries are signed with it, so it has to
// be unguessable rather than merely unique.
func newSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func hostOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	return parsed.Hostname()
}
