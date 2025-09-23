package services

import (
	"context"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// NotificationService defines the interface for notification operations
type NotificationService interface {
	// SendEmail sends email notification
	SendEmail(ctx context.Context, req SendEmailRequest) error

	// SendSMS sends SMS notification
	SendSMS(ctx context.Context, req SendSMSRequest) error

	// SendPushNotification sends push notification
	SendPushNotification(ctx context.Context, req SendPushNotificationRequest) error

	// SendInAppNotification sends in-app notification
	SendInAppNotification(ctx context.Context, req SendInAppNotificationRequest) error

	// SendBulkEmail sends email to multiple recipients
	SendBulkEmail(ctx context.Context, req SendBulkEmailRequest) error

	// SendTemplatedEmail sends templated email
	SendTemplatedEmail(ctx context.Context, req SendTemplatedEmailRequest) error

	// GetNotificationHistory retrieves notification history
	GetNotificationHistory(ctx context.Context, filter NotificationFilter) ([]*NotificationRecord, error)

	// GetNotificationStatus retrieves notification status
	GetNotificationStatus(ctx context.Context, notificationID string) (*NotificationStatus, error)

	// CreateNotificationTemplate creates notification template
	CreateNotificationTemplate(ctx context.Context, template *NotificationTemplate) error

	// UpdateNotificationTemplate updates notification template
	UpdateNotificationTemplate(ctx context.Context, template *NotificationTemplate) error

	// GetNotificationTemplate retrieves notification template
	GetNotificationTemplate(ctx context.Context, templateID string) (*NotificationTemplate, error)

	// ListNotificationTemplates lists notification templates
	ListNotificationTemplates(ctx context.Context, filter TemplateFilter) ([]*NotificationTemplate, error)

	// DeleteNotificationTemplate deletes notification template
	DeleteNotificationTemplate(ctx context.Context, templateID string) error
}

// EmailService defines the interface for email operations
type EmailService interface {
	// Send sends email
	Send(ctx context.Context, email *Email) error

	// SendBulk sends bulk email
	SendBulk(ctx context.Context, emails []*Email) error

	// ValidateEmail validates email address
	ValidateEmail(email string) error

	// GetDeliveryStatus gets email delivery status
	GetDeliveryStatus(ctx context.Context, messageID string) (*DeliveryStatus, error)

	// GetBounces gets bounced emails
	GetBounces(ctx context.Context, filter BounceFilter) ([]*Bounce, error)

	// GetComplaints gets complaint emails
	GetComplaints(ctx context.Context, filter ComplaintFilter) ([]*Complaint, error)

	// AddToSuppressionList adds email to suppression list
	AddToSuppressionList(ctx context.Context, email string, reason SuppressionReason) error

	// RemoveFromSuppressionList removes email from suppression list
	RemoveFromSuppressionList(ctx context.Context, email string) error

	// IsEmailSuppressed checks if email is suppressed
	IsEmailSuppressed(ctx context.Context, email string) (bool, error)
}

// SMSService defines the interface for SMS operations
type SMSService interface {
	// Send sends SMS
	Send(ctx context.Context, sms *SMS) error

	// SendBulk sends bulk SMS
	SendBulk(ctx context.Context, messages []*SMS) error

	// ValidatePhoneNumber validates phone number
	ValidatePhoneNumber(phoneNumber string) error

	// GetDeliveryStatus gets SMS delivery status
	GetDeliveryStatus(ctx context.Context, messageID string) (*DeliveryStatus, error)

	// GetOptOuts gets opt-out phone numbers
	GetOptOuts(ctx context.Context) ([]string, error)

	// AddOptOut adds phone number to opt-out list
	AddOptOut(ctx context.Context, phoneNumber string) error

	// RemoveOptOut removes phone number from opt-out list
	RemoveOptOut(ctx context.Context, phoneNumber string) error

	// IsOptedOut checks if phone number is opted out
	IsOptedOut(ctx context.Context, phoneNumber string) (bool, error)
}

// PushNotificationService defines the interface for push notification operations
type PushNotificationService interface {
	// Send sends push notification
	Send(ctx context.Context, notification *PushNotification) error

	// SendBulk sends bulk push notifications
	SendBulk(ctx context.Context, notifications []*PushNotification) error

	// RegisterDevice registers device for push notifications
	RegisterDevice(ctx context.Context, req RegisterDeviceRequest) error

	// UnregisterDevice unregisters device
	UnregisterDevice(ctx context.Context, deviceToken string) error

	// GetDevices gets registered devices for user
	GetDevices(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) ([]*Device, error)

	// UpdateDeviceSettings updates device notification settings
	UpdateDeviceSettings(ctx context.Context, deviceToken string, settings DeviceSettings) error
}

// Request types

// SendEmailRequest represents email send request
type SendEmailRequest struct {
	TenantID    tenant.TenantID        `json:"tenant_id"`
	To          []string               `json:"to"`
	CC          []string               `json:"cc,omitempty"`
	BCC         []string               `json:"bcc,omitempty"`
	From        string                 `json:"from"`
	FromName    string                 `json:"from_name,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	Subject     string                 `json:"subject"`
	TextContent string                 `json:"text_content,omitempty"`
	HTMLContent string                 `json:"html_content,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SendSMSRequest represents SMS send request
type SendSMSRequest struct {
	TenantID  tenant.TenantID        `json:"tenant_id"`
	To        string                 `json:"to"`
	From      string                 `json:"from,omitempty"`
	Message   string                 `json:"message"`
	MediaURLs []string               `json:"media_urls,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SendPushNotificationRequest represents push notification send request
type SendPushNotificationRequest struct {
	TenantID     tenant.TenantID        `json:"tenant_id"`
	UserID       user.UserID            `json:"user_id"`
	DeviceTokens []string               `json:"device_tokens,omitempty"` // If empty, send to all user devices
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Icon         string                 `json:"icon,omitempty"`
	Image        string                 `json:"image,omitempty"`
	Sound        string                 `json:"sound,omitempty"`
	Badge        *int                   `json:"badge,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ClickAction  string                 `json:"click_action,omitempty"`
	TTL          *int                   `json:"ttl,omitempty"` // Time to live in seconds
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SendInAppNotificationRequest represents in-app notification send request
type SendInAppNotificationRequest struct {
	TenantID   tenant.TenantID        `json:"tenant_id"`
	UserID     user.UserID            `json:"user_id"`
	Type       InAppNotificationType  `json:"type"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	ActionURL  string                 `json:"action_url,omitempty"`
	ActionText string                 `json:"action_text,omitempty"`
	Priority   NotificationPriority   `json:"priority"`
	ExpiresAt  *shared.Timestamp      `json:"expires_at,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// SendBulkEmailRequest represents bulk email send request
type SendBulkEmailRequest struct {
	TenantID    tenant.TenantID        `json:"tenant_id"`
	Recipients  []BulkEmailRecipient   `json:"recipients"`
	From        string                 `json:"from"`
	FromName    string                 `json:"from_name,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	Subject     string                 `json:"subject"`
	TextContent string                 `json:"text_content,omitempty"`
	HTMLContent string                 `json:"html_content,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SendTemplatedEmailRequest represents templated email send request
type SendTemplatedEmailRequest struct {
	TenantID     tenant.TenantID        `json:"tenant_id"`
	TemplateID   string                 `json:"template_id"`
	To           []string               `json:"to"`
	CC           []string               `json:"cc,omitempty"`
	BCC          []string               `json:"bcc,omitempty"`
	From         string                 `json:"from,omitempty"`
	FromName     string                 `json:"from_name,omitempty"`
	ReplyTo      string                 `json:"reply_to,omitempty"`
	TemplateData map[string]interface{} `json:"template_data"`
	Attachments  []Attachment           `json:"attachments,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BulkEmailRecipient represents bulk email recipient
type BulkEmailRecipient struct {
	Email        string                 `json:"email"`
	Name         string                 `json:"name,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
}

// RegisterDeviceRequest represents device registration request
type RegisterDeviceRequest struct {
	TenantID    tenant.TenantID `json:"tenant_id"`
	UserID      user.UserID     `json:"user_id"`
	DeviceToken string          `json:"device_token"`
	Platform    DevicePlatform  `json:"platform"`
	AppVersion  string          `json:"app_version,omitempty"`
	OSVersion   string          `json:"os_version,omitempty"`
	DeviceModel string          `json:"device_model,omitempty"`
	Language    string          `json:"language,omitempty"`
	Timezone    string          `json:"timezone,omitempty"`
}

// Data types

// Email represents an email message
type Email struct {
	To          []string               `json:"to"`
	CC          []string               `json:"cc,omitempty"`
	BCC         []string               `json:"bcc,omitempty"`
	From        string                 `json:"from"`
	FromName    string                 `json:"from_name,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	Subject     string                 `json:"subject"`
	TextContent string                 `json:"text_content,omitempty"`
	HTMLContent string                 `json:"html_content,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SMS represents an SMS message
type SMS struct {
	To        string                 `json:"to"`
	From      string                 `json:"from,omitempty"`
	Message   string                 `json:"message"`
	MediaURLs []string               `json:"media_urls,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PushNotification represents a push notification
type PushNotification struct {
	DeviceTokens []string               `json:"device_tokens"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Icon         string                 `json:"icon,omitempty"`
	Image        string                 `json:"image,omitempty"`
	Sound        string                 `json:"sound,omitempty"`
	Badge        *int                   `json:"badge,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ClickAction  string                 `json:"click_action,omitempty"`
	TTL          *int                   `json:"ttl,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Attachment represents email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id,omitempty"`  // For inline attachments
	Disposition string `json:"disposition,omitempty"` // attachment or inline
}

// NotificationRecord represents notification record
type NotificationRecord struct {
	ID           string                 `json:"id"`
	TenantID     tenant.TenantID        `json:"tenant_id"`
	UserID       *user.UserID           `json:"user_id,omitempty"`
	Type         NotificationType       `json:"type"`
	Channel      NotificationChannel    `json:"channel"`
	Recipient    string                 `json:"recipient"`
	Subject      string                 `json:"subject,omitempty"`
	Content      string                 `json:"content"`
	Status       NotificationStatus     `json:"status"`
	SentAt       *shared.Timestamp      `json:"sent_at,omitempty"`
	DeliveredAt  *shared.Timestamp      `json:"delivered_at,omitempty"`
	ReadAt       *shared.Timestamp      `json:"read_at,omitempty"`
	FailedAt     *shared.Timestamp      `json:"failed_at,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ProviderID   string                 `json:"provider_id,omitempty"`
	MessageID    string                 `json:"message_id,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    shared.Timestamp       `json:"created_at"`
}

// NotificationType represents notification type
type NotificationType string

const (
	NotificationTypeWelcome           NotificationType = "welcome"
	NotificationTypePasswordReset     NotificationType = "password_reset"
	NotificationTypeInvitation        NotificationType = "invitation"
	NotificationTypePaymentSucceeded  NotificationType = "payment_succeeded"
	NotificationTypePaymentFailed     NotificationType = "payment_failed"
	NotificationTypeTrialExpiring     NotificationType = "trial_expiring"
	NotificationTypeSubscriptionEnded NotificationType = "subscription_ended"
	NotificationTypeUsageAlert        NotificationType = "usage_alert"
	NotificationTypeSecurityAlert     NotificationType = "security_alert"
	NotificationTypeSystemAlert       NotificationType = "system_alert"
	NotificationTypeMarketing         NotificationType = "marketing"
	NotificationTypeCustom            NotificationType = "custom"
)

// NotificationChannel represents notification channel
type NotificationChannel string

const (
	ChannelEmail   NotificationChannel = "email"
	ChannelSMS     NotificationChannel = "sms"
	ChannelPush    NotificationChannel = "push"
	ChannelInApp   NotificationChannel = "in_app"
	ChannelWebhook NotificationChannel = "webhook"
)

// NotificationStatus represents notification status
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusRead      NotificationStatus = "read"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusBounced   NotificationStatus = "bounced"
	NotificationStatusComplaint NotificationStatus = "complaint"
)

// InAppNotificationType represents in-app notification type
type InAppNotificationType string

const (
	InAppTypeInfo    InAppNotificationType = "info"
	InAppTypeSuccess InAppNotificationType = "success"
	InAppTypeWarning InAppNotificationType = "warning"
	InAppTypeError   InAppNotificationType = "error"
)

// NotificationPriority represents notification priority
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// DevicePlatform represents device platform
type DevicePlatform string

const (
	PlatformIOS     DevicePlatform = "ios"
	PlatformAndroid DevicePlatform = "android"
	PlatformWeb     DevicePlatform = "web"
)

// Device represents a registered device
type Device struct {
	ID          string           `json:"id"`
	UserID      user.UserID      `json:"user_id"`
	TenantID    tenant.TenantID  `json:"tenant_id"`
	DeviceToken string           `json:"device_token"`
	Platform    DevicePlatform   `json:"platform"`
	AppVersion  string           `json:"app_version,omitempty"`
	OSVersion   string           `json:"os_version,omitempty"`
	DeviceModel string           `json:"device_model,omitempty"`
	Language    string           `json:"language,omitempty"`
	Timezone    string           `json:"timezone,omitempty"`
	IsActive    bool             `json:"is_active"`
	LastSeenAt  shared.Timestamp `json:"last_seen_at"`
	CreatedAt   shared.Timestamp `json:"created_at"`
	Settings    DeviceSettings   `json:"settings"`
}

// DeviceSettings represents device notification settings
type DeviceSettings struct {
	EnablePush      bool                   `json:"enable_push"`
	EnableSound     bool                   `json:"enable_sound"`
	EnableVibration bool                   `json:"enable_vibration"`
	QuietHoursStart string                 `json:"quiet_hours_start,omitempty"` // HH:MM format
	QuietHoursEnd   string                 `json:"quiet_hours_end,omitempty"`   // HH:MM format
	EnabledTypes    []NotificationType     `json:"enabled_types"`
	DisabledTypes   []NotificationType     `json:"disabled_types"`
	CustomSettings  map[string]interface{} `json:"custom_settings,omitempty"`
}

// NotificationTemplate represents notification template
type NotificationTemplate struct {
	ID          string                 `json:"id"`
	TenantID    *tenant.TenantID       `json:"tenant_id,omitempty"` // nil for global templates
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        NotificationType       `json:"type"`
	Channel     NotificationChannel    `json:"channel"`
	Subject     string                 `json:"subject,omitempty"`
	TextContent string                 `json:"text_content,omitempty"`
	HTMLContent string                 `json:"html_content,omitempty"`
	Variables   []TemplateVariable     `json:"variables"`
	IsActive    bool                   `json:"is_active"`
	IsDefault   bool                   `json:"is_default"`
	CreatedAt   shared.Timestamp       `json:"created_at"`
	UpdatedAt   shared.Timestamp       `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateVariable represents template variable
type TemplateVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // string, number, boolean, date
	Description  string `json:"description,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	Required     bool   `json:"required"`
}

// DeliveryStatus represents delivery status
type DeliveryStatus struct {
	MessageID    string             `json:"message_id"`
	Status       NotificationStatus `json:"status"`
	DeliveredAt  *shared.Timestamp  `json:"delivered_at,omitempty"`
	BouncedAt    *shared.Timestamp  `json:"bounced_at,omitempty"`
	ComplaintAt  *shared.Timestamp  `json:"complaint_at,omitempty"`
	ErrorMessage string             `json:"error_message,omitempty"`
}

// Bounce represents email bounce
type Bounce struct {
	MessageID    string           `json:"message_id"`
	Email        string           `json:"email"`
	BounceType   BounceType       `json:"bounce_type"`
	BounceReason string           `json:"bounce_reason"`
	BouncedAt    shared.Timestamp `json:"bounced_at"`
}

// BounceType represents bounce type
type BounceType string

const (
	BounceTypePermanent BounceType = "permanent"
	BounceTypeTransient BounceType = "transient"
	BounceTypeComplaint BounceType = "complaint"
)

// Complaint represents email complaint
type Complaint struct {
	MessageID     string           `json:"message_id"`
	Email         string           `json:"email"`
	ComplaintType string           `json:"complaint_type"`
	ComplaintAt   shared.Timestamp `json:"complaint_at"`
}

// SuppressionReason represents suppression reason
type SuppressionReason string

const (
	SuppressionBounce      SuppressionReason = "bounce"
	SuppressionComplaint   SuppressionReason = "complaint"
	SuppressionUnsubscribe SuppressionReason = "unsubscribe"
	SuppressionManual      SuppressionReason = "manual"
)

// Filter types

// NotificationFilter represents notification filtering options
type NotificationFilter struct {
	TenantID   *tenant.TenantID     `json:"tenant_id,omitempty"`
	UserID     *user.UserID         `json:"user_id,omitempty"`
	Type       *NotificationType    `json:"type,omitempty"`
	Channel    *NotificationChannel `json:"channel,omitempty"`
	Status     *NotificationStatus  `json:"status,omitempty"`
	Tags       []string             `json:"tags,omitempty"`
	SentAfter  *shared.Timestamp    `json:"sent_after,omitempty"`
	SentBefore *shared.Timestamp    `json:"sent_before,omitempty"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
}

// TemplateFilter represents template filtering options
type TemplateFilter struct {
	TenantID  *tenant.TenantID     `json:"tenant_id,omitempty"`
	Type      *NotificationType    `json:"type,omitempty"`
	Channel   *NotificationChannel `json:"channel,omitempty"`
	IsActive  *bool                `json:"is_active,omitempty"`
	IsDefault *bool                `json:"is_default,omitempty"`
	Search    string               `json:"search,omitempty"`
	Limit     int                  `json:"limit"`
	Offset    int                  `json:"offset"`
}

// BounceFilter represents bounce filtering options
type BounceFilter struct {
	BounceType    *BounceType       `json:"bounce_type,omitempty"`
	BouncedAfter  *shared.Timestamp `json:"bounced_after,omitempty"`
	BouncedBefore *shared.Timestamp `json:"bounced_before,omitempty"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
}

// ComplaintFilter represents complaint filtering options
type ComplaintFilter struct {
	ComplaintAfter  *shared.Timestamp `json:"complaint_after,omitempty"`
	ComplaintBefore *shared.Timestamp `json:"complaint_before,omitempty"`
	Limit           int               `json:"limit"`
	Offset          int               `json:"offset"`
}

// Helper methods
func (n *NotificationRecord) IsDelivered() bool {
	return n.Status == NotificationStatusDelivered || n.Status == NotificationStatusRead
}

func (n *NotificationRecord) IsFailed() bool {
	return n.Status == NotificationStatusFailed || n.Status == NotificationStatusBounced
}

func (d *Device) IsExpired() bool {
	// Device is considered expired if not seen for 90 days
	return shared.Now().Time().Sub(d.LastSeenAt.Time()).Hours() > 90*24
}

func (t *NotificationTemplate) HasVariable(name string) bool {
	for _, v := range t.Variables {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (s *DeviceSettings) IsNotificationEnabled(notificationType NotificationType) bool {
	// Check if type is explicitly disabled
	for _, disabled := range s.DisabledTypes {
		if disabled == notificationType {
			return false
		}
	}

	// Check if type is explicitly enabled
	for _, enabled := range s.EnabledTypes {
		if enabled == notificationType {
			return true
		}
	}

	// Default behavior - enable most types except marketing
	return notificationType != NotificationTypeMarketing
}
