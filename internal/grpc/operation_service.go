package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	keystonev1 "github.com/canakyuz/keystone/internal/grpc/keystone/v1"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	"github.com/canakyuz/keystone/pkg/tracing"
)

// OperationStore is the storage behaviour this service needs.
//
// Declared here, on the consumer side, and identical to the one the HTTP handler
// declares. Both point at the same repository: the guarantees live there, and this file
// is a translation layer between protobuf and the domain, nothing more.
type OperationStore interface {
	CreateTenantProvision(ctx context.Context, req oprepo.ProvisionRequest) (*oprepo.ProvisionResult, error)
	GetOperation(ctx context.Context, id, subject string) (*domain.Operation, error)
}

// OperationService serves keystone.v1.OperationService.
type OperationService struct {
	keystonev1.UnimplementedOperationServiceServer

	store OperationStore
}

// NewOperationService builds the service.
func NewOperationService(store OperationStore) *OperationService {
	return &OperationService{store: store}
}

// CreateTenant accepts the provisioning work.
//
// It goes through the same repository call as the REST endpoint, so it inherits the same
// behaviour without restating it: the tenant, operation, job and idempotency record are
// written in one transaction, and a repeated key returns the existing operation rather
// than starting a second provisioning.
func (s *OperationService) CreateTenant(
	ctx context.Context, req *keystonev1.CreateTenantRequest,
) (*keystonev1.CreateTenantResponse, error) {
	if req.GetName() == "" || req.GetSlug() == "" || req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "name, slug and email are required")
	}

	subject := SubjectFrom(ctx)

	result, err := s.store.CreateTenantProvision(ctx, oprepo.ProvisionRequest{
		Name:      req.GetName(),
		Slug:      req.GetSlug(),
		Email:     req.GetEmail(),
		Plan:      req.GetPlan(),
		CreatedBy: subject,
		// The scope binds the key to the subject, so one customer's key cannot match
		// another's request. Same rule as the REST surface, and it has to be, or a key
		// would mean different things depending on which port it arrived on.
		Scope:          "subject:" + subject,
		IdempotencyKey: req.GetIdempotencyKey(),
		RequestBody:    []byte(req.GetName() + "|" + req.GetSlug() + "|" + req.GetEmail() + "|" + req.GetPlan()),
		TraceContext:   tracing.Marshal(ctx),
	})
	if err != nil {
		return nil, mapCreateError(err)
	}

	return &keystonev1.CreateTenantResponse{
		Operation: toProtoOperation(result.Operation),
		TenantId:  result.TenantID,
		Replayed:  result.Replayed,
	}, nil
}

// GetOperation returns the current state of an operation.
func (s *OperationService) GetOperation(
	ctx context.Context, req *keystonev1.GetOperationRequest,
) (*keystonev1.GetOperationResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	op, err := s.store.GetOperation(ctx, req.GetId(), SubjectFrom(ctx))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "operation not found")
		}

		return nil, status.Error(codes.Internal, "could not read the operation")
	}

	return &keystonev1.GetOperationResponse{Operation: toProtoOperation(op)}, nil
}

// mapCreateError turns a domain error into a gRPC code.
//
// The codes are chosen to mean the same thing the REST statuses do: a conflict is a
// conflict on both surfaces. A client that reads the code should not have to know which
// port it used.
func mapCreateError(err error) error {
	switch {
	case errors.Is(err, domain.ErrIdempotencyConflict):
		return status.Error(codes.AlreadyExists, "this idempotency key was used with a different request")
	case errors.Is(err, oprepo.ErrSlugTaken):
		return status.Error(codes.AlreadyExists, "slug already taken")
	default:
		return status.Error(codes.Internal, "could not process the request")
	}
}

// toProtoOperation converts the domain record into its wire form.
func toProtoOperation(op *domain.Operation) *keystonev1.Operation {
	if op == nil {
		return nil
	}

	out := &keystonev1.Operation{
		Id:           op.ID,
		TenantId:     op.TenantID,
		Kind:         string(op.Kind),
		Status:       toProtoStatus(op.Status),
		ErrorCode:    op.ErrorCode,
		ErrorMessage: op.ErrorMessage,
		CreatedAt:    timestamppb.New(op.CreatedAt),
		UpdatedAt:    timestamppb.New(op.UpdatedAt),
	}

	if op.CompletedAt != nil {
		out.CompletedAt = timestamppb.New(*op.CompletedAt)
	}

	return out
}

// toProtoStatus maps the domain status onto the enum.
//
// An unrecognised status becomes UNSPECIFIED rather than defaulting to PENDING. Guessing
// would make a new state added later look like a state the client already understands,
// which is worse than admitting the value is unknown.
func toProtoStatus(s domain.Status) keystonev1.OperationStatus {
	switch s {
	case domain.StatusPending:
		return keystonev1.OperationStatus_OPERATION_STATUS_PENDING
	case domain.StatusRunning:
		return keystonev1.OperationStatus_OPERATION_STATUS_RUNNING
	case domain.StatusSucceeded:
		return keystonev1.OperationStatus_OPERATION_STATUS_SUCCEEDED
	case domain.StatusFailed:
		return keystonev1.OperationStatus_OPERATION_STATUS_FAILED
	default:
		return keystonev1.OperationStatus_OPERATION_STATUS_UNSPECIFIED
	}
}
