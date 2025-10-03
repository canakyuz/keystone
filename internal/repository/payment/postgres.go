package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"nexpaces-api/internal/domain/payment"
	"nexpaces-api/internal/domain/refund"
	"nexpaces-api/internal/domain/webhook"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SetTenantContext sets the tenant context for RLS
func (r *PostgresRepository) SetTenantContext(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, "SET app.current_tenant = $1", tenantID)
	return err
}

// ==================== Payment Operations ====================

// CreatePayment creates a new payment
func (r *PostgresRepository) CreatePayment(ctx context.Context, p *payment.Payment) error {
	query := `
		INSERT INTO payments (
			id, tenant_id, user_id,
			provider, provider_payment_id,
			amount, currency,
			status, failure_code, failure_message,
			requires_3ds, threeds_version, threeds_html_content, threeds_callback_url, threeds_status,
			installment, installment_rate,
			card_brand, card_last4, card_bin, card_exp_month, card_exp_year,
			card_holder_name, card_fingerprint,
			customer_email, customer_first_name, customer_last_name, customer_phone, customer_ip_address,
			billing_country, billing_city, billing_address_line, billing_zip_code,
			description, reference_id, order_id, invoice_number, metadata,
			created_at, updated_at, succeeded_at, canceled_at,
			created_by, updated_by
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17,
			$18, $19, $20, $21, $22,
			$23, $24,
			$25, $26, $27, $28, $29,
			$30, $31, $32, $33,
			$34, $35, $36, $37, $38,
			$39, $40, $41, $42,
			$43, $44
		)
	`

	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		p.ID, p.TenantID, p.UserID,
		p.Provider, p.ProviderPaymentID,
		p.Amount, p.Currency,
		p.Status, p.FailureCode, p.FailureMessage,
		p.Requires3DS, p.ThreeDSVersion, p.ThreeDSHTMLContent, p.ThreeDSCallbackURL, p.ThreeDSStatus,
		p.Installment, p.InstallmentRate,
		p.CardBrand, p.CardLast4, p.CardBIN, p.CardExpMonth, p.CardExpYear,
		p.CardHolderName, p.CardFingerprint,
		p.CustomerEmail, p.CustomerFirstName, p.CustomerLastName, p.CustomerPhone, p.CustomerIPAddress,
		p.BillingCountry, p.BillingCity, p.BillingAddressLine, p.BillingZipCode,
		p.Description, p.ReferenceID, p.OrderID, p.InvoiceNumber, metadataJSON,
		p.CreatedAt, p.UpdatedAt, p.SucceededAt, p.CanceledAt,
		p.CreatedBy, p.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// GetPaymentByID retrieves a payment by ID
func (r *PostgresRepository) GetPaymentByID(ctx context.Context, id string) (*payment.Payment, error) {
	query := `
		SELECT
			id, tenant_id, user_id,
			provider, provider_payment_id,
			amount, currency,
			status, failure_code, failure_message,
			requires_3ds, threeds_version, threeds_html_content, threeds_callback_url, threeds_status,
			installment, installment_rate,
			card_brand, card_last4, card_bin, card_exp_month, card_exp_year,
			card_holder_name, card_fingerprint,
			customer_email, customer_first_name, customer_last_name, customer_phone, customer_ip_address,
			billing_country, billing_city, billing_address_line, billing_zip_code,
			description, reference_id, order_id, invoice_number, metadata,
			created_at, updated_at, succeeded_at, canceled_at,
			created_by, updated_by
		FROM payments
		WHERE id = $1
	`

	p := &payment.Payment{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.TenantID, &p.UserID,
		&p.Provider, &p.ProviderPaymentID,
		&p.Amount, &p.Currency,
		&p.Status, &p.FailureCode, &p.FailureMessage,
		&p.Requires3DS, &p.ThreeDSVersion, &p.ThreeDSHTMLContent, &p.ThreeDSCallbackURL, &p.ThreeDSStatus,
		&p.Installment, &p.InstallmentRate,
		&p.CardBrand, &p.CardLast4, &p.CardBIN, &p.CardExpMonth, &p.CardExpYear,
		&p.CardHolderName, &p.CardFingerprint,
		&p.CustomerEmail, &p.CustomerFirstName, &p.CustomerLastName, &p.CustomerPhone, &p.CustomerIPAddress,
		&p.BillingCountry, &p.BillingCity, &p.BillingAddressLine, &p.BillingZipCode,
		&p.Description, &p.ReferenceID, &p.OrderID, &p.InvoiceNumber, &metadataJSON,
		&p.CreatedAt, &p.UpdatedAt, &p.SucceededAt, &p.CanceledAt,
		&p.CreatedBy, &p.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, payment.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return p, nil
}

// GetPaymentByProviderID retrieves a payment by provider payment ID
func (r *PostgresRepository) GetPaymentByProviderID(ctx context.Context, tenantID, providerPaymentID string) (*payment.Payment, error) {
	query := `
		SELECT
			id, tenant_id, user_id,
			provider, provider_payment_id,
			amount, currency,
			status, failure_code, failure_message,
			requires_3ds, threeds_version, threeds_html_content, threeds_callback_url, threeds_status,
			installment, installment_rate,
			card_brand, card_last4, card_bin, card_exp_month, card_exp_year,
			card_holder_name, card_fingerprint,
			customer_email, customer_first_name, customer_last_name, customer_phone, customer_ip_address,
			billing_country, billing_city, billing_address_line, billing_zip_code,
			description, reference_id, order_id, invoice_number, metadata,
			created_at, updated_at, succeeded_at, canceled_at,
			created_by, updated_by
		FROM payments
		WHERE tenant_id = $1 AND provider_payment_id = $2
	`

	p := &payment.Payment{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, providerPaymentID).Scan(
		&p.ID, &p.TenantID, &p.UserID,
		&p.Provider, &p.ProviderPaymentID,
		&p.Amount, &p.Currency,
		&p.Status, &p.FailureCode, &p.FailureMessage,
		&p.Requires3DS, &p.ThreeDSVersion, &p.ThreeDSHTMLContent, &p.ThreeDSCallbackURL, &p.ThreeDSStatus,
		&p.Installment, &p.InstallmentRate,
		&p.CardBrand, &p.CardLast4, &p.CardBIN, &p.CardExpMonth, &p.CardExpYear,
		&p.CardHolderName, &p.CardFingerprint,
		&p.CustomerEmail, &p.CustomerFirstName, &p.CustomerLastName, &p.CustomerPhone, &p.CustomerIPAddress,
		&p.BillingCountry, &p.BillingCity, &p.BillingAddressLine, &p.BillingZipCode,
		&p.Description, &p.ReferenceID, &p.OrderID, &p.InvoiceNumber, &metadataJSON,
		&p.CreatedAt, &p.UpdatedAt, &p.SucceededAt, &p.CanceledAt,
		&p.CreatedBy, &p.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, payment.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment by provider ID: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return p, nil
}

// UpdatePayment updates a payment
func (r *PostgresRepository) UpdatePayment(ctx context.Context, p *payment.Payment) error {
	query := `
		UPDATE payments SET
			provider_payment_id = $2,
			status = $3, failure_code = $4, failure_message = $5,
			requires_3ds = $6, threeds_version = $7, threeds_html_content = $8,
			threeds_callback_url = $9, threeds_status = $10,
			installment = $11, installment_rate = $12,
			card_brand = $13, card_last4 = $14, card_bin = $15,
			card_exp_month = $16, card_exp_year = $17,
			card_holder_name = $18, card_fingerprint = $19,
			metadata = $20, updated_at = $21,
			succeeded_at = $22, canceled_at = $23,
			updated_by = $24
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		p.ID,
		p.ProviderPaymentID,
		p.Status, p.FailureCode, p.FailureMessage,
		p.Requires3DS, p.ThreeDSVersion, p.ThreeDSHTMLContent,
		p.ThreeDSCallbackURL, p.ThreeDSStatus,
		p.Installment, p.InstallmentRate,
		p.CardBrand, p.CardLast4, p.CardBIN,
		p.CardExpMonth, p.CardExpYear,
		p.CardHolderName, p.CardFingerprint,
		metadataJSON, p.UpdatedAt,
		p.SucceededAt, p.CanceledAt,
		p.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

// ListPaymentsByTenant retrieves payments by tenant
func (r *PostgresRepository) ListPaymentsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*payment.Payment, error) {
	query := `
		SELECT
			id, tenant_id, user_id,
			provider, provider_payment_id,
			amount, currency,
			status, failure_code, failure_message,
			requires_3ds, threeds_version, threeds_html_content, threeds_callback_url, threeds_status,
			installment, installment_rate,
			card_brand, card_last4, card_bin, card_exp_month, card_exp_year,
			card_holder_name, card_fingerprint,
			customer_email, customer_first_name, customer_last_name, customer_phone, customer_ip_address,
			billing_country, billing_city, billing_address_line, billing_zip_code,
			description, reference_id, order_id, invoice_number, metadata,
			created_at, updated_at, succeeded_at, canceled_at,
			created_by, updated_by
		FROM payments
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	var payments []*payment.Payment
	for rows.Next() {
		p := &payment.Payment{}
		var metadataJSON []byte

		err := rows.Scan(
			&p.ID, &p.TenantID, &p.UserID,
			&p.Provider, &p.ProviderPaymentID,
			&p.Amount, &p.Currency,
			&p.Status, &p.FailureCode, &p.FailureMessage,
			&p.Requires3DS, &p.ThreeDSVersion, &p.ThreeDSHTMLContent, &p.ThreeDSCallbackURL, &p.ThreeDSStatus,
			&p.Installment, &p.InstallmentRate,
			&p.CardBrand, &p.CardLast4, &p.CardBIN, &p.CardExpMonth, &p.CardExpYear,
			&p.CardHolderName, &p.CardFingerprint,
			&p.CustomerEmail, &p.CustomerFirstName, &p.CustomerLastName, &p.CustomerPhone, &p.CustomerIPAddress,
			&p.BillingCountry, &p.BillingCity, &p.BillingAddressLine, &p.BillingZipCode,
			&p.Description, &p.ReferenceID, &p.OrderID, &p.InvoiceNumber, &metadataJSON,
			&p.CreatedAt, &p.UpdatedAt, &p.SucceededAt, &p.CanceledAt,
			&p.CreatedBy, &p.UpdatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		payments = append(payments, p)
	}

	return payments, nil
}

// ==================== Refund Operations ====================

// CreateRefund creates a new refund
func (r *PostgresRepository) CreateRefund(ctx context.Context, rf *refund.Refund) error {
	query := `
		INSERT INTO refunds (
			id, tenant_id, payment_id,
			provider, provider_refund_id,
			amount, currency,
			status, failure_code, failure_message,
			reason, description, metadata,
			created_at, updated_at, succeeded_at,
			created_by, updated_by
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16,
			$17, $18
		)
	`

	metadataJSON, err := json.Marshal(rf.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		rf.ID, rf.TenantID, rf.PaymentID,
		rf.Provider, rf.ProviderRefundID,
		rf.Amount, rf.Currency,
		rf.Status, rf.FailureCode, rf.FailureMessage,
		rf.Reason, rf.Description, metadataJSON,
		rf.CreatedAt, rf.UpdatedAt, rf.SucceededAt,
		rf.CreatedBy, rf.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	return nil
}

// GetRefundByID retrieves a refund by ID
func (r *PostgresRepository) GetRefundByID(ctx context.Context, id string) (*refund.Refund, error) {
	query := `
		SELECT
			id, tenant_id, payment_id,
			provider, provider_refund_id,
			amount, currency,
			status, failure_code, failure_message,
			reason, description, metadata,
			created_at, updated_at, succeeded_at,
			created_by, updated_by
		FROM refunds
		WHERE id = $1
	`

	rf := &refund.Refund{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rf.ID, &rf.TenantID, &rf.PaymentID,
		&rf.Provider, &rf.ProviderRefundID,
		&rf.Amount, &rf.Currency,
		&rf.Status, &rf.FailureCode, &rf.FailureMessage,
		&rf.Reason, &rf.Description, &metadataJSON,
		&rf.CreatedAt, &rf.UpdatedAt, &rf.SucceededAt,
		&rf.CreatedBy, &rf.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, refund.ErrRefundNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refund: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &rf.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return rf, nil
}

// GetRefundByProviderID retrieves a refund by provider refund ID
func (r *PostgresRepository) GetRefundByProviderID(ctx context.Context, tenantID, providerRefundID string) (*refund.Refund, error) {
	query := `
		SELECT
			id, tenant_id, payment_id,
			provider, provider_refund_id,
			amount, currency,
			status, failure_code, failure_message,
			reason, description, metadata,
			created_at, updated_at, succeeded_at,
			created_by, updated_by
		FROM refunds
		WHERE tenant_id = $1 AND provider_refund_id = $2
	`

	rf := &refund.Refund{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, providerRefundID).Scan(
		&rf.ID, &rf.TenantID, &rf.PaymentID,
		&rf.Provider, &rf.ProviderRefundID,
		&rf.Amount, &rf.Currency,
		&rf.Status, &rf.FailureCode, &rf.FailureMessage,
		&rf.Reason, &rf.Description, &metadataJSON,
		&rf.CreatedAt, &rf.UpdatedAt, &rf.SucceededAt,
		&rf.CreatedBy, &rf.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, refund.ErrRefundNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refund by provider ID: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &rf.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return rf, nil
}

// UpdateRefund updates a refund
func (r *PostgresRepository) UpdateRefund(ctx context.Context, rf *refund.Refund) error {
	query := `
		UPDATE refunds SET
			provider_refund_id = $2,
			status = $3, failure_code = $4, failure_message = $5,
			description = $6, metadata = $7,
			updated_at = $8, succeeded_at = $9,
			updated_by = $10
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(rf.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		rf.ID,
		rf.ProviderRefundID,
		rf.Status, rf.FailureCode, rf.FailureMessage,
		rf.Description, metadataJSON,
		rf.UpdatedAt, rf.SucceededAt,
		rf.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update refund: %w", err)
	}

	return nil
}

// ListRefundsByPayment retrieves refunds by payment ID
func (r *PostgresRepository) ListRefundsByPayment(ctx context.Context, paymentID string) ([]*refund.Refund, error) {
	query := `
		SELECT
			id, tenant_id, payment_id,
			provider, provider_refund_id,
			amount, currency,
			status, failure_code, failure_message,
			reason, description, metadata,
			created_at, updated_at, succeeded_at,
			created_by, updated_by
		FROM refunds
		WHERE payment_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}
	defer rows.Close()

	var refunds []*refund.Refund
	for rows.Next() {
		rf := &refund.Refund{}
		var metadataJSON []byte

		err := rows.Scan(
			&rf.ID, &rf.TenantID, &rf.PaymentID,
			&rf.Provider, &rf.ProviderRefundID,
			&rf.Amount, &rf.Currency,
			&rf.Status, &rf.FailureCode, &rf.FailureMessage,
			&rf.Reason, &rf.Description, &metadataJSON,
			&rf.CreatedAt, &rf.UpdatedAt, &rf.SucceededAt,
			&rf.CreatedBy, &rf.UpdatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan refund: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &rf.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		refunds = append(refunds, rf)
	}

	return refunds, nil
}

// ListRefundsByTenant retrieves refunds by tenant
func (r *PostgresRepository) ListRefundsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*refund.Refund, error) {
	query := `
		SELECT
			id, tenant_id, payment_id,
			provider, provider_refund_id,
			amount, currency,
			status, failure_code, failure_message,
			reason, description, metadata,
			created_at, updated_at, succeeded_at,
			created_by, updated_by
		FROM refunds
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}
	defer rows.Close()

	var refunds []*refund.Refund
	for rows.Next() {
		rf := &refund.Refund{}
		var metadataJSON []byte

		err := rows.Scan(
			&rf.ID, &rf.TenantID, &rf.PaymentID,
			&rf.Provider, &rf.ProviderRefundID,
			&rf.Amount, &rf.Currency,
			&rf.Status, &rf.FailureCode, &rf.FailureMessage,
			&rf.Reason, &rf.Description, &metadataJSON,
			&rf.CreatedAt, &rf.UpdatedAt, &rf.SucceededAt,
			&rf.CreatedBy, &rf.UpdatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan refund: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &rf.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		refunds = append(refunds, rf)
	}

	return refunds, nil
}

// ==================== Webhook Event Operations ====================

// CreateEvent creates a new webhook event
func (r *PostgresRepository) CreateEvent(ctx context.Context, e *webhook.PaymentEvent) error {
	query := `
		INSERT INTO payment_events (
			id, tenant_id,
			provider, provider_event_id,
			event_type, event_version, payload,
			payment_id, refund_id,
			processed, processed_at, processing_error, retry_count,
			request_ip_address, request_headers, signature_valid,
			created_at, updated_at
		) VALUES (
			$1, $2,
			$3, $4,
			$5, $6, $7,
			$8, $9,
			$10, $11, $12, $13,
			$14, $15, $16,
			$17, $18
		)
	`

	payloadJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	headersJSON, err := json.Marshal(e.RequestHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal request headers: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		e.ID, e.TenantID,
		e.Provider, e.ProviderEventID,
		e.EventType, e.EventVersion, payloadJSON,
		e.PaymentID, e.RefundID,
		e.Processed, e.ProcessedAt, e.ProcessingError, e.RetryCount,
		e.RequestIPAddress, headersJSON, e.SignatureValid,
		e.CreatedAt, e.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// GetEventByID retrieves an event by ID
func (r *PostgresRepository) GetEventByID(ctx context.Context, id string) (*webhook.PaymentEvent, error) {
	query := `
		SELECT
			id, tenant_id,
			provider, provider_event_id,
			event_type, event_version, payload,
			payment_id, refund_id,
			processed, processed_at, processing_error, retry_count,
			request_ip_address, request_headers, signature_valid,
			created_at, updated_at
		FROM payment_events
		WHERE id = $1
	`

	e := &webhook.PaymentEvent{}
	var payloadJSON, headersJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&e.ID, &e.TenantID,
		&e.Provider, &e.ProviderEventID,
		&e.EventType, &e.EventVersion, &payloadJSON,
		&e.PaymentID, &e.RefundID,
		&e.Processed, &e.ProcessedAt, &e.ProcessingError, &e.RetryCount,
		&e.RequestIPAddress, &headersJSON, &e.SignatureValid,
		&e.CreatedAt, &e.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, webhook.ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if err := json.Unmarshal(headersJSON, &e.RequestHeaders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	return e, nil
}

// GetEventByProviderEventID retrieves an event by provider event ID (for idempotency)
func (r *PostgresRepository) GetEventByProviderEventID(ctx context.Context, tenantID, provider, providerEventID string) (*webhook.PaymentEvent, error) {
	query := `
		SELECT
			id, tenant_id,
			provider, provider_event_id,
			event_type, event_version, payload,
			payment_id, refund_id,
			processed, processed_at, processing_error, retry_count,
			request_ip_address, request_headers, signature_valid,
			created_at, updated_at
		FROM payment_events
		WHERE tenant_id = $1 AND provider = $2 AND provider_event_id = $3
	`

	e := &webhook.PaymentEvent{}
	var payloadJSON, headersJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, provider, providerEventID).Scan(
		&e.ID, &e.TenantID,
		&e.Provider, &e.ProviderEventID,
		&e.EventType, &e.EventVersion, &payloadJSON,
		&e.PaymentID, &e.RefundID,
		&e.Processed, &e.ProcessedAt, &e.ProcessingError, &e.RetryCount,
		&e.RequestIPAddress, &headersJSON, &e.SignatureValid,
		&e.CreatedAt, &e.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, webhook.ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event by provider ID: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if err := json.Unmarshal(headersJSON, &e.RequestHeaders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	return e, nil
}

// UpdateEvent updates an event
func (r *PostgresRepository) UpdateEvent(ctx context.Context, e *webhook.PaymentEvent) error {
	query := `
		UPDATE payment_events SET
			payment_id = $2, refund_id = $3,
			processed = $4, processed_at = $5,
			processing_error = $6, retry_count = $7,
			signature_valid = $8, updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		e.ID,
		e.PaymentID, e.RefundID,
		e.Processed, e.ProcessedAt,
		e.ProcessingError, e.RetryCount,
		e.SignatureValid, e.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// ListUnprocessedEvents retrieves unprocessed events
func (r *PostgresRepository) ListUnprocessedEvents(ctx context.Context, limit int) ([]*webhook.PaymentEvent, error) {
	query := `
		SELECT
			id, tenant_id,
			provider, provider_event_id,
			event_type, event_version, payload,
			payment_id, refund_id,
			processed, processed_at, processing_error, retry_count,
			request_ip_address, request_headers, signature_valid,
			created_at, updated_at
		FROM payment_events
		WHERE processed = false AND retry_count < 5
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unprocessed events: %w", err)
	}
	defer rows.Close()

	var events []*webhook.PaymentEvent
	for rows.Next() {
		e := &webhook.PaymentEvent{}
		var payloadJSON, headersJSON []byte

		err := rows.Scan(
			&e.ID, &e.TenantID,
			&e.Provider, &e.ProviderEventID,
			&e.EventType, &e.EventVersion, &payloadJSON,
			&e.PaymentID, &e.RefundID,
			&e.Processed, &e.ProcessedAt, &e.ProcessingError, &e.RetryCount,
			&e.RequestIPAddress, &headersJSON, &e.SignatureValid,
			&e.CreatedAt, &e.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		if err := json.Unmarshal(headersJSON, &e.RequestHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		events = append(events, e)
	}

	return events, nil
}

// ListEventsByTenant retrieves events by tenant
func (r *PostgresRepository) ListEventsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*webhook.PaymentEvent, error) {
	query := `
		SELECT
			id, tenant_id,
			provider, provider_event_id,
			event_type, event_version, payload,
			payment_id, refund_id,
			processed, processed_at, processing_error, retry_count,
			request_ip_address, request_headers, signature_valid,
			created_at, updated_at
		FROM payment_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []*webhook.PaymentEvent
	for rows.Next() {
		e := &webhook.PaymentEvent{}
		var payloadJSON, headersJSON []byte

		err := rows.Scan(
			&e.ID, &e.TenantID,
			&e.Provider, &e.ProviderEventID,
			&e.EventType, &e.EventVersion, &payloadJSON,
			&e.PaymentID, &e.RefundID,
			&e.Processed, &e.ProcessedAt, &e.ProcessingError, &e.RetryCount,
			&e.RequestIPAddress, &headersJSON, &e.SignatureValid,
			&e.CreatedAt, &e.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		if err := json.Unmarshal(headersJSON, &e.RequestHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		events = append(events, e)
	}

	return events, nil
}
