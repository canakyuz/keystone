// Package webhook holds the rules for registering the addresses a tenant is notified at.
package webhook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
	webhookrepo "github.com/canakyuz/keystone/internal/repository/webhook"
	"github.com/canakyuz/keystone/pkg/outbound"
)

// Endpoint is a registered destination.
type Endpoint = webhookrepo.Endpoint

// Errors the store reports, passed through so callers need not import the repository.
var (
	ErrNotFound     = webhookrepo.ErrNotFound
	ErrLimitReached = webhookrepo.ErrLimitReached
)

// InvalidURLError is a destination the delivery dialler would refuse. Its message is
// written for the administrator who typed the address.
type InvalidURLError struct{ Reason string }

func (e *InvalidURLError) Error() string { return e.Reason }

// Store is what the service needs from the webhook repository.
type Store interface {
	Create(ctx context.Context, tenantID, url, secret string, entry auditrepo.Entry) (*webhookrepo.Endpoint, error)
	List(ctx context.Context, tenantID string) ([]webhookrepo.Endpoint, error)
	SetActive(ctx context.Context, tenantID, id string, active bool, entry auditrepo.Entry) error
}

// Service manages a tenant's webhook destinations.
type Service struct {
	store Store
}

// NewService creates the service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns the tenant's destinations.
func (s *Service) List(ctx context.Context, tenantID string) ([]Endpoint, error) {
	return s.store.List(ctx, tenantID)
}

// Create registers a destination and returns it with its signing secret. The secret is not
// stored anywhere a reader can fetch it again, so this is the only time it is returned.
func (s *Service) Create(ctx context.Context, tenantID, rawURL string) (*Endpoint, string, error) {
	destination := strings.TrimSpace(rawURL)

	// The cheap check, which gives the administrator a message they can act on. The
	// security boundary is the dialler at delivery time; see pkg/outbound.
	if err := outbound.ValidateURL(destination); err != nil {
		return nil, "", &InvalidURLError{Reason: strings.TrimPrefix(err.Error(), "outbound: ")}
	}

	secret, err := newSecret()
	if err != nil {
		return nil, "", fmt.Errorf("could not create a secret: %w", err)
	}

	created, err := s.store.Create(ctx, tenantID, destination, secret,
		// The host, not the url. Some services put the secret in the path of the url they
		// hand out, and the trail keeps whatever it is given forever.
		entry(ctx, tenantID, "webhook.created", map[string]any{"host": hostOf(destination)}))
	if err != nil {
		return nil, "", err
	}

	return created, secret, nil
}

// SetActive turns a destination on or off.
func (s *Service) SetActive(ctx context.Context, tenantID, id string, active bool) error {
	action := "webhook.disabled"
	if active {
		action = "webhook.enabled"
	}

	return s.store.SetActive(ctx, tenantID, id, active, entry(ctx, tenantID, action, nil))
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
