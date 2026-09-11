package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/canakyuz/keystone/internal/authz"
	keystonev1 "github.com/canakyuz/keystone/internal/grpc/keystone/v1"
	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
)

// Config describes the gRPC listener.
type Config struct {
	// Addr is where the server listens. Empty disables it.
	Addr string

	// JWTSecret verifies the same tokens the REST API accepts. One secret, because a
	// token is a statement about a subject, not about a transport.
	JWTSecret string

	// Reflection exposes the service descriptors so grpcurl and similar tools can call
	// the API without a copy of the .proto files.
	//
	// It is off by default. Reflection lists every method and message the server knows,
	// which is a map of the attack surface handed to anyone who can reach the port.
	// Useful in development, not something to leave on by accident.
	Reflection bool
}

// Server wraps the gRPC listener and its lifecycle.
type Server struct {
	server *grpc.Server
	addr   string
	log    *logger.Logger
}

// New builds the gRPC server with its interceptor chain.
//
// Order matters and is the same order as the HTTP middleware, for the same reasons.
// Recovery is outermost so it catches a panic in anything inside it, including the other
// interceptors. Observability comes next so a call rejected by authentication is still
// counted and traced: during an incident, when most calls are being rejected, is exactly
// when the dashboards must not disagree with reality. Authentication is innermost, so
// everything below it can assume a verified tenant.
func New(
	cfg Config,
	schemaCache *middleware.TenantSchemaCache,
	members authz.MemberLookup,
	operations *OperationService,
	users *UserService,
	reg *metrics.Registry,
	log *logger.Logger,
) *Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			observabilityInterceptor(reg, log),
			authInterceptor(cfg.JWTSecret, schemaCache, members),
		),
	)

	keystonev1.RegisterOperationServiceServer(server, operations)
	keystonev1.RegisterUserServiceServer(server, users)

	if cfg.Reflection {
		reflection.Register(server)
	}

	return &Server{server: server, addr: cfg.Addr, log: log}
}

// Start begins serving and returns once the listener is bound.
//
// Binding happens synchronously so that a port already in use is reported at startup
// rather than swallowed by a goroutine, where it would leave a process that looks healthy
// and answers nothing.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("could not listen on %s: %w", s.addr, err)
	}

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) && s.log != nil {
			s.log.WithFields(logger.Fields{"error": err.Error()}).Error("grpc server stopped")
		}
	}()

	if s.log != nil {
		s.log.WithFields(logger.Fields{"addr": s.addr}).Info("grpc server started")
	}

	return nil
}

// Stop drains the server, cancelling anything still running after the grace period.
//
// GracefulStop alone can hang indefinitely on a streaming call that never ends, which
// turns a deployment into an outage. The timeout is the difference between draining and
// waiting forever.
func (s *Server) Stop(grace time.Duration) {
	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(grace):
		s.server.Stop()
	}
}

// Handler exposes the underlying server, for tests that drive it over a pipe rather than
// a real port.
func (s *Server) Handler() *grpc.Server { return s.server }

var _ = context.Background
