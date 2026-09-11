package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/canakyuz/keystone/examples/verticals/domain/payment"
	domainRefund "github.com/canakyuz/keystone/examples/verticals/domain/refund"
	"github.com/canakyuz/keystone/examples/verticals/domain/webhook"
	providerPayment "github.com/canakyuz/keystone/examples/verticals/provider/payment"
	paymentRepo "github.com/canakyuz/keystone/examples/verticals/repository/payment"
)

// Service handles payment business logic
type Service struct {
	repo         paymentRepo.Repository
	orchestrator *providerPayment.Orchestrator
}

// NewService creates a new payment service
func NewService(repo paymentRepo.Repository, orchestrator *providerPayment.Orchestrator) *Service {
	return &Service{
		repo:         repo,
		orchestrator: orchestrator,
	}
}

// CreatePayment creates a new payment
func (s *Service) CreatePayment(ctx context.Context, tenantID, userID string, tenantSettings map[string]any, req *CreatePaymentRequest) (*PaymentResponse, error) {
	// Select payment provider based on tenant settings and currency
	provider, err := s.orchestrator.SelectProvider(tenantSettings, req.Currency, req.BillingCountry)
	if err != nil {
		return nil, fmt.Errorf("failed to select provider: %w", err)
	}

	// Generate reference ID
	referenceID := uuid.New().String()

	// Build provider request
	providerReq := &providerPayment.PaymentRequest{
		TenantID:           tenantID,
		Amount:             req.Amount,
		Currency:           req.Currency,
		Description:        req.Description,
		OrderID:            req.OrderID,
		ReferenceID:        referenceID,
		CustomerEmail:      req.CustomerEmail,
		CustomerFirstName:  req.CustomerFirstName,
		CustomerLastName:   req.CustomerLastName,
		CustomerPhone:      req.CustomerPhone,
		CardToken:          req.CardToken,
		CardNumber:         req.CardNumber,
		CardExpMonth:       req.CardExpMonth,
		CardExpYear:        req.CardExpYear,
		CardCVV:            req.CardCVV,
		CardHolderName:     req.CardHolderName,
		BillingCountry:     req.BillingCountry,
		BillingCity:        req.BillingCity,
		BillingAddressLine: req.BillingAddressLine,
		BillingZipCode:     req.BillingZipCode,
		Installment:        req.Installment,
		CallbackURL:        req.CallbackURL,
		SuccessURL:         req.SuccessURL,
		FailureURL:         req.FailureURL,
		Metadata:           req.Metadata,
	}

	// Create payment via provider
	providerResp, err := provider.CreatePayment(ctx, providerReq)
	if err != nil {
		return nil, fmt.Errorf("provider payment creation failed: %w", err)
	}

	// Create domain payment entity
	p, err := payment.New(tenantID, payment.PaymentProvider(provider.GetName()), req.Amount, req.Currency)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment entity: %w", err)
	}

	// Set payment details from provider response
	p.UserID = userID
	p.ReferenceID = referenceID
	p.OrderID = req.OrderID
	p.Description = req.Description
	p.ProviderPaymentID = providerResp.ProviderPaymentID
	p.Metadata = req.Metadata
	p.CreatedBy = userID

	// Set customer info
	p.SetCustomerInfo(req.CustomerEmail, req.CustomerFirstName, req.CustomerLastName, req.CustomerPhone, "")
	p.SetBillingAddress(req.BillingCountry, req.BillingCity, req.BillingAddressLine, req.BillingZipCode)

	// Handle 3DS flow
	if providerResp.Requires3DS {
		if err := p.MarkAsRequires3DS(providerResp.ThreeDSVersion, providerResp.ThreeDSHTMLContent, req.CallbackURL); err != nil {
			return nil, fmt.Errorf("failed to mark as requires 3DS: %w", err)
		}
	} else if providerResp.Status == "succeeded" {
		if err := p.MarkAsSucceeded(providerResp.ProviderPaymentID); err != nil {
			return nil, fmt.Errorf("failed to mark as succeeded: %w", err)
		}
	} else if providerResp.Status == "failed" {
		if err := p.MarkAsFailed(providerResp.FailureCode, providerResp.FailureMessage); err != nil {
			return nil, fmt.Errorf("failed to mark as failed: %w", err)
		}
	}

	// Set card information (masked)
	if providerResp.CardBrand != "" {
		p.SetCardInfo(
			payment.CardBrand(providerResp.CardBrand),
			providerResp.CardLast4,
			providerResp.CardBIN,
			providerResp.CardExpMonth,
			providerResp.CardExpYear,
			req.CardHolderName,
			providerResp.CardFingerprint,
		)
	}

	// Set installment
	if req.Installment > 1 {
		if err := p.SetInstallment(req.Installment, providerResp.InstallmentRate); err != nil {
			return nil, fmt.Errorf("failed to set installment: %w", err)
		}
	}

	// Save to database
	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return ToPaymentResponse(p), nil
}

// Complete3DSPayment completes 3DS authentication
func (s *Service) Complete3DSPayment(ctx context.Context, req *Complete3DSRequest) (*PaymentResponse, error) {
	// Get payment
	p, err := s.repo.GetPaymentByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Check if payment requires 3DS
	if !p.Requires3DSAuth() {
		return nil, fmt.Errorf("payment does not require 3DS authentication")
	}

	// Complete 3DS via provider
	providerReq := &providerPayment.CompletePaymentRequest{
		ProviderPaymentID: p.ProviderPaymentID,
		PaymentID:         p.ID,
		ConversationData:  req.ConversationData,
		CallbackParams:    req.CallbackParams,
	}

	providerResp, err := s.orchestrator.CompletePayment(ctx, string(p.Provider), providerReq)
	if err != nil {
		return nil, fmt.Errorf("3DS completion failed: %w", err)
	}

	// Update payment status
	if providerResp.Status == "succeeded" {
		if err := p.Complete3DSAuthentication(true); err != nil {
			return nil, fmt.Errorf("failed to complete 3DS: %w", err)
		}
		if err := p.MarkAsSucceeded(providerResp.ProviderPaymentID); err != nil {
			return nil, fmt.Errorf("failed to mark as succeeded: %w", err)
		}
	} else {
		if err := p.Complete3DSAuthentication(false); err != nil {
			return nil, fmt.Errorf("failed to complete 3DS: %w", err)
		}
	}

	// Update in database
	if err := s.repo.UpdatePayment(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return ToPaymentResponse(p), nil
}

// GetPayment retrieves a payment by ID
func (s *Service) GetPayment(ctx context.Context, paymentID string) (*PaymentResponse, error) {
	p, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	return ToPaymentResponse(p), nil
}

// ListPayments lists payments for a tenant
func (s *Service) ListPayments(ctx context.Context, tenantID string, req *ListPaymentsRequest) ([]*PaymentResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	payments, err := s.repo.ListPaymentsByTenant(ctx, tenantID, limit, req.Offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*PaymentResponse, len(payments))
	for i, p := range payments {
		responses[i] = ToPaymentResponse(p)
	}

	return responses, nil
}

// CreateRefund creates a refund for a payment
func (s *Service) CreateRefund(ctx context.Context, tenantID, userID string, req *CreateRefundRequest) (*RefundResponse, error) {
	// Get payment
	p, err := s.repo.GetPaymentByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Check if payment can be refunded
	if !p.CanBeRefunded() {
		return nil, fmt.Errorf("payment cannot be refunded (status: %s)", p.Status)
	}

	// Validate refund amount
	if req.Amount > p.Amount {
		return nil, fmt.Errorf("refund amount exceeds payment amount")
	}

	// Create refund entity
	rf, err := domainRefund.New(
		tenantID,
		req.PaymentID,
		domainRefund.PaymentProvider(p.Provider),
		req.Amount,
		p.Currency,
		domainRefund.RefundReason(req.Reason),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund entity: %w", err)
	}

	rf.SetDescription(req.Description)
	rf.CreatedBy = userID

	// Create refund via provider
	providerReq := &providerPayment.RefundRequest{
		TenantID:          tenantID,
		PaymentID:         req.PaymentID,
		ProviderPaymentID: p.ProviderPaymentID,
		Amount:            req.Amount,
		Currency:          p.Currency,
		Reason:            req.Reason,
		Description:       req.Description,
	}

	providerResp, err := s.orchestrator.CreateRefund(ctx, string(p.Provider), providerReq)
	if err != nil {
		return nil, fmt.Errorf("provider refund creation failed: %w", err)
	}

	// Update refund with provider response
	if providerResp.Status == "succeeded" {
		if err := rf.MarkAsSucceeded(providerResp.ProviderRefundID); err != nil {
			return nil, fmt.Errorf("failed to mark refund as succeeded: %w", err)
		}

		// Mark payment as refunded
		if err := p.MarkAsRefunded(); err != nil {
			return nil, fmt.Errorf("failed to mark payment as refunded: %w", err)
		}
		if err := s.repo.UpdatePayment(ctx, p); err != nil {
			return nil, fmt.Errorf("failed to update payment: %w", err)
		}
	} else if providerResp.Status == "failed" {
		if err := rf.MarkAsFailed(providerResp.FailureCode, providerResp.FailureMessage); err != nil {
			return nil, fmt.Errorf("failed to mark refund as failed: %w", err)
		}
	}

	// Save refund
	if err := s.repo.CreateRefund(ctx, rf); err != nil {
		return nil, fmt.Errorf("failed to save refund: %w", err)
	}

	return ToRefundResponse(rf), nil
}

// ProcessWebhook processes a webhook event
func (s *Service) ProcessWebhook(ctx context.Context, tenantID, provider string, payload []byte, signature string, ipAddress string, headers map[string]any) (*WebhookEventResponse, error) {
	// Verify webhook signature
	if err := s.orchestrator.VerifyWebhookSignature(ctx, provider, payload, signature); err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Parse webhook
	parsedEvent, err := s.orchestrator.ParseWebhook(ctx, provider, payload)
	if err != nil {
		return nil, fmt.Errorf("webhook parsing failed: %w", err)
	}

	// Check for duplicate event (idempotency)
	existingEvent, err := s.repo.GetEventByProviderEventID(ctx, tenantID, provider, parsedEvent.ProviderEventID)
	if err == nil && existingEvent != nil {
		// Event already processed
		return ToWebhookEventResponse(existingEvent), nil
	}

	// Create webhook event entity
	event, err := webhook.New(
		tenantID,
		webhook.PaymentProvider(provider),
		parsedEvent.ProviderEventID,
		webhook.EventType(parsedEvent.EventType),
		parsedEvent.Payload,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook event: %w", err)
	}

	event.SetSignatureValid(true)
	event.SetRequestMetadata(ipAddress, headers)

	// Save event
	if err := s.repo.CreateEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to save webhook event: %w", err)
	}

	// Process event asynchronously (would be done by a background worker in production)
	go s.processWebhookEvent(context.Background(), event, parsedEvent)

	return ToWebhookEventResponse(event), nil
}

// processWebhookEvent processes webhook event (background job)
func (s *Service) processWebhookEvent(ctx context.Context, event *webhook.PaymentEvent, parsedEvent *providerPayment.WebhookEvent) {
	// This would be handled by a background worker in production
	// For now, we'll do basic processing

	if event.IsPaymentEvent() {
		// Find payment and update status
		if payment, err := s.repo.GetPaymentByProviderID(ctx, event.TenantID, parsedEvent.PaymentID); err == nil {
			switch parsedEvent.EventType {
			case "payment.succeeded":
				payment.MarkAsSucceeded(parsedEvent.PaymentID)
				s.repo.UpdatePayment(ctx, payment)
			case "payment.failed":
				payment.MarkAsFailed(parsedEvent.FailureCode, parsedEvent.FailureMessage)
				s.repo.UpdatePayment(ctx, payment)
			}

			// Mark event as processed
			event.SetPaymentID(payment.ID)
		}
	}

	event.MarkAsProcessed()
	s.repo.UpdateEvent(ctx, event)
}
