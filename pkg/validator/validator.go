package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Validator wraps go-playground validator with custom validations
type Validator struct {
	validate *validator.Validate
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var messages []string
	for _, err := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// New creates a new validator instance with custom validations
func New() *Validator {
	validate := validator.New()

	// Register custom validations
	_ = validate.RegisterValidation("uuid", validateUUID)
	_ = validate.RegisterValidation("slug", validateSlug)
	_ = validate.RegisterValidation("domain", validateDomain)
	_ = validate.RegisterValidation("nospecialchars", validateNoSpecialChars)
	_ = validate.RegisterValidation("alphanumspace", validateAlphaNumSpace)

	return &Validator{
		validate: validate,
	}
}

// Validate validates a struct and returns formatted errors
func (v *Validator) Validate(data interface{}) error {
	err := v.validate.Struct(data)
	if err == nil {
		return nil
	}

	validationErrors := ValidationErrors{}

	// Type assert to validator.ValidationErrors
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range errs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   fieldErr.Field(),
				Message: formatErrorMessage(fieldErr),
				Tag:     fieldErr.Tag(),
				Value:   fmt.Sprintf("%v", fieldErr.Value()),
			})
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return err
}

// ValidateVar validates a single variable
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	err := v.validate.Var(field, tag)
	if err == nil {
		return nil
	}

	if errs, ok := err.(validator.ValidationErrors); ok {
		validationErrors := ValidationErrors{}
		for _, fieldErr := range errs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "value",
				Message: formatErrorMessage(fieldErr),
				Tag:     fieldErr.Tag(),
			})
		}
		return validationErrors
	}

	return err
}

// Custom validation functions

// validateUUID validates UUID v4 format
func validateUUID(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Let required handle empty values
	}
	_, err := uuid.Parse(value)
	return err == nil
}

// validateSlug validates URL-friendly slug format
func validateSlug(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	// Slug: lowercase letters, numbers, and hyphens only
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return slugRegex.MatchString(value)
}

// validateDomain validates domain name format
func validateDomain(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	// Basic domain validation (simplified)
	domainRegex := regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	return domainRegex.MatchString(strings.ToLower(value))
}

// validateNoSpecialChars ensures no special characters except basic punctuation
func validateNoSpecialChars(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	// Allow letters, numbers, spaces, and basic punctuation
	noSpecialCharsRegex := regexp.MustCompile(`^[a-zA-Z0-9\s.,!?()-]+$`)
	return noSpecialCharsRegex.MatchString(value)
}

// validateAlphaNumSpace validates alphanumeric characters and spaces only
func validateAlphaNumSpace(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	alphaNumSpaceRegex := regexp.MustCompile(`^[a-zA-Z0-9\s]+$`)
	return alphaNumSpaceRegex.MatchString(value)
}

// formatErrorMessage formats validation error messages
func formatErrorMessage(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must not exceed %s characters", field, err.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, err.Param())
	case "eq":
		return fmt.Sprintf("%s must equal %s", field, err.Param())
	case "ne":
		return fmt.Sprintf("%s must not equal %s", field, err.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, err.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, err.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, err.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, err.Param())
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	case "numeric":
		return fmt.Sprintf("%s must be a valid number", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uri":
		return fmt.Sprintf("%s must be a valid URI", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "slug":
		return fmt.Sprintf("%s must be a valid slug (lowercase, alphanumeric with hyphens)", field)
	case "domain":
		return fmt.Sprintf("%s must be a valid domain name", field)
	case "nospecialchars":
		return fmt.Sprintf("%s contains invalid special characters", field)
	case "alphanumspace":
		return fmt.Sprintf("%s must contain only letters, numbers, and spaces", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, err.Param())
	case "eqfield":
		return fmt.Sprintf("%s must match %s", field, err.Param())
	case "nefield":
		return fmt.Sprintf("%s must not match %s", field, err.Param())
	default:
		return fmt.Sprintf("%s failed validation: %s", field, tag)
	}
}

// Common validation helpers

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

// IsValidEmail checks if a string is a valid email
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidSlug checks if a string is a valid slug
func IsValidSlug(slug string) bool {
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return slugRegex.MatchString(slug)
}

// IsValidDomain checks if a string is a valid domain
func IsValidDomain(domain string) bool {
	domainRegex := regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	return domainRegex.MatchString(strings.ToLower(domain))
}

// SanitizeString removes leading/trailing whitespace
func SanitizeString(value string) string {
	return strings.TrimSpace(value)
}

// SanitizeSlug converts a string to a valid slug
func SanitizeSlug(value string) string {
	// Convert to lowercase
	slug := strings.ToLower(value)
	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove non-alphanumeric characters except hyphens
	slugRegex := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = slugRegex.ReplaceAllString(slug, "")
	// Remove multiple consecutive hyphens
	multiHyphenRegex := regexp.MustCompile(`-+`)
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")
	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")
	return slug
}

// NormalizeEmail normalizes an email address
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
