// Package audit serves the trail to the people who have to read it.
//
// Reading it is an administrator's business rather than every member's: the trail says who
// suspended whom and when, which is exactly the history a suspended member would like to
// see. The route carries the role guard; this handler assumes it ran.
package audit

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/internal/middleware"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

// Handler serves the audit endpoints.
type Handler struct {
	reader *auditrepo.Reader
}

// NewHandler creates the handler.
func NewHandler(reader *auditrepo.Reader) *Handler {
	return &Handler{reader: reader}
}

// Entry is one line of the trail as the API returns it.
type Entry struct {
	ID          string         `json:"id"`
	Action      string         `json:"action"`
	ActorID     string         `json:"actor_id,omitempty"`
	ActorEmail  string         `json:"actor_email,omitempty"`
	ActorType   string         `json:"actor_type"`
	SubjectType string         `json:"subject_type"`
	SubjectID   string         `json:"subject_id"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ListResponse is a page of the trail.
type ListResponse struct {
	Entries []Entry `json:"entries"`

	// NextBefore is what to send as `before` for the following page. Absent on the last one.
	NextBefore *time.Time `json:"next_before,omitempty"`
}

// List returns the tenant's trail, newest first.
// GET /api/v1/audit?limit=50&before=2026-09-13T10:00:00Z
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "before must be an RFC 3339 timestamp",
			})
		}
		before = &parsed
	}

	page, err := h.reader.List(c.UserContext(), tenantID, limit, before)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "could not read the audit trail",
		})
	}

	entries := make([]Entry, 0, len(page.Records))
	for _, record := range page.Records {
		entries = append(entries, Entry{
			ID:          record.ID,
			Action:      record.Action,
			ActorID:     record.ActorID,
			ActorEmail:  record.ActorEmail,
			ActorType:   string(record.ActorType),
			SubjectType: record.SubjectType,
			SubjectID:   record.SubjectID,
			Metadata:    record.Metadata,
			CreatedAt:   record.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"data": ListResponse{Entries: entries, NextBefore: page.Before}})
}
