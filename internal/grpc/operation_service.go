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
	opuc "github.com/canakyuz/keystone/internal/usecase/operation"
	"github.com/canakyuz/keystone/pkg/tracing"
)

// Operations is what this service needs from the operation usecase.
//
// The REST handler goes through the same usecase, so the validation and the idempotency
// scope are one rule, not two copies of it. This file translates between protobuf and the
// usecase, nothing more.
type Operations interface {
	ProvisionTenant(ctx context.Context, in opuc.ProvisionInput) (*oprepo.ProvisionResult, error)
	Get(ctx context.Context, id, subject string) (*domain.Operation, error)
}

// OperationService serves keystone.v1.OperationService.
type OperationService struct {
	keystonev1.UnimplementedOperationServiceServer

	ops Operations
}

// NewOperationService builds the service.
func NewOperationService(ops Operations) *OperationService {
	return &OperationService{ops: ops}
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
	result, err := s.ops.ProvisionTenant(ctx, opuc.ProvisionInput{
		Name:           req.GetName(),
		Slug:           req.GetSlug(),
		Email:          req.GetEmail(),
		Plan:           req.GetPlan(),
		Subject:        SubjectFrom(ctx),
		TraceContext:   tracing.Marshal(ctx),
		IdempotencyKey: req.GetIdempotencyKey(),
		// There is no raw body on this port, so the fingerprint is taken over the fields.
		Body: []byte(req.GetName() + "|" + req.GetSlug() + "|" + req.GetEmail() + "|" + req.GetPlan()),
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
	op, err := s.ops.Get(ctx, req.GetId(), SubjectFrom(ctx))
	if err != nil {
		var invalid *opuc.ValidationError
		if errors.As(err, &invalid) {
			return nil, status.Error(codes.InvalidArgument, invalid.Message)
		}
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
	var invalid *opuc.ValidationError
	switch {
	case errors.As(err, &invalid):
		return status.Error(codes.InvalidArgument, invalid.Message)
	case errors.Is(err, opuc.ErrNotPlatformOperator):
		return status.Error(codes.PermissionDenied, "permission denied")
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
