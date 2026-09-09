// Package tenantctx defines how tenant information is carried through a Go context.
//
// WHY a separate package: these keys used to live in internal/middleware. To read
// them, the repository layer had to import the HTTP middleware package, which made
// the data access layer depend on the transport layer. That inverts the dependency
// direction of clean architecture, and in practice it produced an import cycle once
// the test helpers came into play.
//
// tenantctx is a leaf package: it imports no project package, so it can be used
// safely from every layer.
package tenantctx

import "context"

// Key is the type used for context keys. A distinct type is used rather than a
// plain string so that keys written by other packages cannot collide with these.
type Key string

const (
	// SchemaKey carries the tenant's PostgreSQL schema name.
	SchemaKey Key = "tenant_schema"
	// IDKey carries the tenant's identifier.
	IDKey Key = "tenant_id"
)

// WithSchema places the schema name into the context.
func WithSchema(ctx context.Context, schemaName string) context.Context {
	return context.WithValue(ctx, SchemaKey, schemaName)
}

// WithID places the tenant identifier into the context.
func WithID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, IDKey, tenantID)
}

// Schema returns the schema name from the context, or the empty string.
func Schema(ctx context.Context) string {
	schema, _ := ctx.Value(SchemaKey).(string)
	return schema
}

// ID returns the tenant identifier from the context, or the empty string.
func ID(ctx context.Context) string {
	id, _ := ctx.Value(IDKey).(string)
	return id
}
