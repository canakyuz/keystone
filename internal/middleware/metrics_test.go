package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormaliseRoute_CollapsesTrailingSlash verifies one endpoint yields one label.
//
// Fiber reports the group's path for a request rejected in group middleware and the
// handler's path for one that reached the handler, and those differ by a trailing
// slash. Without this, the rejected requests and the served ones land in different
// series and cannot be compared.
func TestNormaliseRoute_CollapsesTrailingSlash(t *testing.T) {
	for input, want := range map[string]string{
		"/api/v1/users/":    "/api/v1/users",
		"/api/v1/users":     "/api/v1/users",
		"/api/v1/users/:id": "/api/v1/users/:id",
		"/":                 "/",
		"":                  "",
	} {
		assert.Equal(t, want, normaliseRoute(input), "input %q", input)
	}
}
