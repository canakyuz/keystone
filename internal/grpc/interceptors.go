// Package grpc serves the typed contract defined in proto/keystone/v1.
//
// It runs alongside the REST API rather than replacing it, on its own port and in the
// same process. Both surfaces call the same repositories, so the guarantees — idempotency,
// tenant isolation, the operation state machine — have one implementation and two ways in.
// A second implementation would be a second set of bugs.
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/canakyuz/keystone/pkg/authn"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
	"github.com/canakyuz/keystone/pkg/tenantctx"
	"github.com/canakyuz/keystone/pkg/tracing"
)

// publicMethods need no credentials.
//
// An allowlist rather than a per-method flag: a new method is authenticated unless
// somebody deliberately adds it here, so forgetting to think about it fails closed.
var publicMethods = map[string]bool{
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
	"/grpc.health.v1.Health/Check":                                   true,
}

// recoveryInterceptor turns a panic into an error instead of a dead process.
//
// gRPC does not recover panics in handlers by default, so one nil dereference takes the
// whole server down along with every in-flight request on it.
func recoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if log != nil {
					log.WithFields(logger.Fields{
						"method": info.FullMethod,
						"panic":  recovered,
					}).Error("handler panicked")
				}

				// The panic value is not returned to the caller. It routinely contains a
				// pointer, a query fragment or a row, none of which the caller should see.
				err = status.Error(codes.Internal, "internal error")
			}
		}()

		return handler(ctx, req)
	}
}

// authInterceptor verifies the caller's token and puts the tenant into the context.
//
// It is the gRPC half of the same decision the HTTP middleware makes, and the decision
// itself lives in pkg/authn so the two cannot drift. The tenant is read from the verified
// claim and from nowhere else: gRPC metadata is caller-controlled, so a tenant header
// here would be exactly the escalation the HTTP side already closed once.
func authInterceptor(jwtSecret string, schemaCache *middleware.TenantSchemaCache) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		claims, err := authn.Verify(authorizationFrom(ctx), jwtSecret)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "unauthenticated")
		}

		if claims.TenantID == "" {
			return nil, status.Error(codes.Unauthenticated, "the token carries no tenant")
		}

		// Resolving the schema here rather than in each handler keeps the guarantee in one
		// place, and reuses the same cache the HTTP path uses, including its negative
		// caching: a flood of requests naming tenants that do not exist does not reach the
		// database twice.
		schema, err := schemaCache.GetTenantSchema(ctx, claims.TenantID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "tenant not found")
		}

		ctx = tenantctx.WithID(ctx, claims.TenantID)
		ctx = tenantctx.WithSchema(ctx, schema)
		ctx = context.WithValue(ctx, subjectKey{}, claims.UserID)

		return handler(ctx, req)
	}
}

// subjectKey carries the authenticated subject through the context.
type subjectKey struct{}

// SubjectFrom returns the authenticated subject, or the empty string.
func SubjectFrom(ctx context.Context) string {
	id, _ := ctx.Value(subjectKey{}).(string)

	return id
}

// observabilityInterceptor records the call and traces it.
//
// Metrics and tracing share one interceptor because they need the same three facts —
// method, outcome, duration — and splitting them would mean walking the same call twice.
func observabilityInterceptor(reg *metrics.Registry, log *logger.Logger) grpc.UnaryServerInterceptor {
	tracer := tracing.Tracer("keystone-grpc")

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		var finish func(method, route string, status int, elapsed time.Duration)
		if reg != nil {
			finish = reg.HTTPStarted()
		}

		resp, err := handler(ctx, req)

		code := status.Code(err)

		if finish != nil {
			// The full method is a bounded label: it comes from the generated service
			// descriptor, not from the request, so a caller cannot invent new values.
			finish("grpc", info.FullMethod, httpStatusFor(code), time.Since(start))
		}

		if log != nil {
			log.WithFields(logger.Fields{
				"method":     info.FullMethod,
				"code":       code.String(),
				"latency_ms": time.Since(start).Milliseconds(),
			}).Info("grpc call")
		}

		return resp, err
	}
}

// httpStatusFor maps a gRPC code onto the status class the metrics use.
//
// The two surfaces share the RED metrics, so they have to agree on what counts as an
// error. Without this, the same failure appears as a 5xx on one surface and as nothing on
// the other, and the error rate on a shared dashboard means neither.
func httpStatusFor(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return 400
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.NotFound:
		return 404
	case codes.AlreadyExists, codes.Aborted:
		return 409
	case codes.ResourceExhausted:
		return 429
	case codes.Unimplemented:
		return 501
	case codes.Unavailable:
		return 503
	case codes.DeadlineExceeded:
		return 504
	default:
		return 500
	}
}

// authorizationFrom reads the authorization metadata entry.
//
// gRPC lowercases metadata keys on the wire, so the lookup is on "authorization"
// regardless of what the client wrote.
func authorizationFrom(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
