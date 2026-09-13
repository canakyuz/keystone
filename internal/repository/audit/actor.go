package audit

import (
	"context"

	"github.com/canakyuz/keystone/pkg/tenantctx"
)

// ActorFrom reads who is making the change out of the request context.
//
// An absent subject is recorded as the system rather than as an empty user. The two are
// different answers, and a trail that blurs them is the one nobody can use afterwards: a
// background job and a request whose identity was lost look the same only if you let them.
func ActorFrom(ctx context.Context) (string, ActorType) {
	if subject := tenantctx.Subject(ctx); subject != "" {
		return subject, ActorUser
	}

	return "", ActorSystem
}
