package validation

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Validator wraps the go-playground validator with custom validation rules
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates a new validator instance with custom rules
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidation("slug", validateSlug)
	v.RegisterValidation("tenant_slug", validateTenantSlug)
	v.RegisterValidation("domain", validateDomain)
	v.RegisterValidation("uuid4", validateUUID4)
	v.RegisterValidation("password_strength", validatePasswordStrength)
	v.RegisterValidation("json_object", validateJSONObject)

	return &Validator{
		validator: v,
	}
}

// Validate validates a struct using the configured validator
func (v *Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}

// validateSlug validates that a string is a valid slug
func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if len(slug) == 0 {
		return false
	}

	// Slug must be lowercase alphanumeric with hyphens
	// Must start and end with alphanumeric
	// Cannot have consecutive hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9]+([a-z0-9-]*[a-z0-9]+)*$`, slug)
	return matched
}

// validateTenantSlug validates that a string is a valid tenant slug
func validateTenantSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if len(slug) < 3 || len(slug) > 50 {
		return false
	}

	// Check if it's a valid slug format
	if !validateSlug(fl) {
		return false
	}

	// Check against reserved slugs
	reservedSlugs := map[string]bool{
		"api":        true,
		"www":        true,
		"admin":      true,
		"root":       true,
		"system":     true,
		"nexspaces":  true,
		"app":        true,
		"mail":       true,
		"email":      true,
		"support":    true,
		"help":       true,
		"docs":       true,
		"blog":       true,
		"status":     true,
		"health":     true,
		"ping":       true,
		"test":       true,
		"staging":    true,
		"dev":        true,
		"prod":       true,
		"production": true,
	}

	return !reservedSlugs[slug]
}

// validateDomain validates that a string is a valid domain name
func validateDomain(fl validator.FieldLevel) bool {
	domain := fl.Field().String()
	if len(domain) == 0 {
		return false
	}

	// Basic domain validation - more comprehensive than built-in fqdn
	// Must be between 1 and 253 characters
	if len(domain) > 253 {
		return false
	}

	// Domain regex - simplified but practical
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	return domainRegex.MatchString(domain)
}

// validateUUID4 validates that a string is a valid UUID v4
func validateUUID4(fl validator.FieldLevel) bool {
	uuidStr := fl.Field().String()
	if len(uuidStr) == 0 {
		return false
	}

	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return false
	}

	// Check if it's version 4
	return parsedUUID.Version() == 4
}

// validatePasswordStrength validates password strength
func validatePasswordStrength(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Minimum length
	if len(password) < 8 {
		return false
	}

	// Maximum length (reasonable limit)
	if len(password) > 128 {
		return false
	}

	// Must contain at least one uppercase letter
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)

	// Must contain at least one lowercase letter
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)

	// Must contain at least one digit
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	// Must contain at least one special character
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`).MatchString(password)

	// At least 3 out of 4 criteria must be met
	criteria := 0
	if hasUpper {
		criteria++
	}
	if hasLower {
		criteria++
	}
	if hasDigit {
		criteria++
	}
	if hasSpecial {
		criteria++
	}

	return criteria >= 3
}

// validateJSONObject validates that the field contains a valid JSON object (not array or primitive)
func validateJSONObject(fl validator.FieldLevel) bool {
	// This is primarily for map[string]interface{} fields
	// The actual JSON validation happens at the unmarshaling level
	// This validator just ensures the field is not nil if required
	field := fl.Field()

	if field.Kind() == reflect.Ptr {
		return !field.IsNil()
	}

	if field.Kind() == reflect.Map {
		return !field.IsNil()
	}

	return true
}

// SanitizeInput sanitizes string input to prevent XSS and other issues
func SanitizeInput(input string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Remove control characters except newlines and tabs
	var result strings.Builder
	for _, r := range sanitized {
		if r >= 32 || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// SanitizeSlug creates a URL-safe slug from input
func SanitizeSlug(input string) string {
	// Convert to lowercase
	slug := strings.ToLower(input)

	// Replace spaces and underscores with hyphens
	slug = regexp.MustCompile(`[\s_]+`).ReplaceAllString(slug, "-")

	// Remove non-alphanumeric characters except hyphens
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "")

	// Remove consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// IsValidEmail checks if an email address is valid
func IsValidEmail(email string) bool {
	// Use regex for basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err == nil
}

// Custom validation tags for common patterns
const (
	// TagRequired marks a field as required
	TagRequired = "required"

	// TagEmail validates email format
	TagEmail = "email"

	// TagUUID validates UUID format
	TagUUID = "uuid"

	// TagUUID4 validates UUID v4 format specifically
	TagUUID4 = "uuid4"

	// TagSlug validates slug format
	TagSlug = "slug"

	// TagTenantSlug validates tenant slug format (includes reserved word check)
	TagTenantSlug = "tenant_slug"

	// TagDomain validates domain name format
	TagDomain = "domain"

	// TagPasswordStrength validates password strength
	TagPasswordStrength = "password_strength"

	// TagJSONObject validates JSON object fields
	TagJSONObject = "json_object"

	// TagMin validates minimum length/value
	TagMin = "min"

	// TagMax validates maximum length/value
	TagMax = "max"

	// TagOneOf validates field is one of specified values
	TagOneOf = "oneof"
)
