package ai

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of AI-related error
type ErrorType string

const (
	// ErrTypeProvider indicates provider-related errors
	ErrTypeProvider ErrorType = "provider"

	// ErrTypeConfiguration indicates configuration errors
	ErrTypeConfiguration ErrorType = "configuration"

	// ErrTypeAuthentication indicates authentication errors
	ErrTypeAuthentication ErrorType = "authentication"

	// ErrTypeRateLimit indicates rate limiting errors
	ErrTypeRateLimit ErrorType = "rate_limit"

	// ErrTypeQuota indicates quota/billing errors
	ErrTypeQuota ErrorType = "quota"

	// ErrTypeNetwork indicates network-related errors
	ErrTypeNetwork ErrorType = "network"

	// ErrTypeTimeout indicates timeout errors
	ErrTypeTimeout ErrorType = "timeout"

	// ErrTypeValidation indicates input validation errors
	ErrTypeValidation ErrorType = "validation"

	// ErrTypeRegistration indicates provider registration errors
	ErrTypeRegistration ErrorType = "registration"

	// ErrTypeNotFound indicates provider not found errors
	ErrTypeNotFound ErrorType = "not_found"

	// ErrTypeTokenLimit indicates token limit exceeded errors
	ErrTypeTokenLimit ErrorType = "token_limit"

	// ErrTypeModelUnavailable indicates model unavailable errors
	ErrTypeModelUnavailable ErrorType = "model_unavailable"

	// ErrTypeInternal indicates internal system errors
	ErrTypeInternal ErrorType = "internal"
)

// ProviderError represents errors specific to AI providers
type ProviderError struct {
	// Type categorizes the error
	Type ErrorType `json:"type"`

	// Message provides human-readable error description
	Message string `json:"message"`

	// Provider indicates which provider caused the error
	Provider string `json:"provider,omitempty"`

	// StatusCode for HTTP-related errors
	StatusCode int `json:"status_code,omitempty"`

	// Underlying error that caused this error
	Cause error `json:"-"`

	// Retryable indicates if the operation can be retried
	Retryable bool `json:"retryable"`

	// RetryAfter suggests when to retry (for rate limiting)
	RetryAfter int `json:"retry_after,omitempty"`

	// Details provides additional context
	Details map[string]any `json:"details,omitempty"`
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	var parts []string

	if e.Provider != "" {
		parts = append(parts, fmt.Sprintf("provider=%s", e.Provider))
	}

	parts = append(parts, fmt.Sprintf("type=%s", e.Type))

	if e.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.StatusCode))
	}

	parts = append(parts, e.Message)

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause=%s", e.Cause.Error()))
	}

	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying error
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error type
func (e *ProviderError) Is(target error) bool {
	if pe, ok := target.(*ProviderError); ok {
		return e.Type == pe.Type
	}
	return false
}

// IsRetryable returns whether the error is retryable
func (e *ProviderError) IsRetryable() bool {
	return e.Retryable
}

// ValidationError represents input validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// TokenLimitError represents token limit exceeded errors
type TokenLimitError struct {
	Requested int    `json:"requested"`
	Limit     int    `json:"limit"`
	Provider  string `json:"provider"`
}

// Error implements the error interface
func (e *TokenLimitError) Error() string {
	return fmt.Sprintf("token limit exceeded for provider '%s': requested %d, limit %d",
		e.Provider, e.Requested, e.Limit)
}

// RateLimitError represents rate limiting errors
type RateLimitError struct {
	Provider   string `json:"provider"`
	RetryAfter int    `json:"retry_after"`
	Type       string `json:"type"` // "requests" or "tokens"
}

// Error implements the error interface
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for provider '%s' (%s): retry after %d seconds",
		e.Provider, e.Type, e.RetryAfter)
}

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Provider string `json:"provider"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// Error implements the error interface
func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error for provider '%s', field '%s': %s",
		e.Provider, e.Field, e.Message)
}

// Error constructors for common error types

// NewProviderError creates a new provider error
func NewProviderError(errType ErrorType, message, provider string) *ProviderError {
	return &ProviderError{
		Type:      errType,
		Message:   message,
		Provider:  provider,
		Retryable: isRetryableError(errType),
	}
}

// NewProviderErrorWithCause creates a provider error with an underlying cause
func NewProviderErrorWithCause(errType ErrorType, message, provider string, cause error) *ProviderError {
	return &ProviderError{
		Type:      errType,
		Message:   message,
		Provider:  provider,
		Cause:     cause,
		Retryable: isRetryableError(errType),
	}
}

// NewValidationError creates a validation error
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewTokenLimitError creates a token limit error
func NewTokenLimitError(requested, limit int, provider string) *TokenLimitError {
	return &TokenLimitError{
		Requested: requested,
		Limit:     limit,
		Provider:  provider,
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(provider string, retryAfter int, limitType string) *RateLimitError {
	return &RateLimitError{
		Provider:   provider,
		RetryAfter: retryAfter,
		Type:       limitType,
	}
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(provider, field, message string) *ConfigurationError {
	return &ConfigurationError{
		Provider: provider,
		Field:    field,
		Message:  message,
	}
}

// isRetryableError determines if an error type is retryable
func isRetryableError(errType ErrorType) bool {
	switch errType {
	case ErrTypeRateLimit, ErrTypeTimeout, ErrTypeNetwork:
		return true
	case ErrTypeQuota, ErrTypeAuthentication, ErrTypeValidation, ErrTypeConfiguration:
		return false
	default:
		return false
	}
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if pe, ok := err.(*ProviderError); ok {
		return pe.IsRetryable()
	}
	return false
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if pe, ok := err.(*ProviderError); ok {
		return pe.Type == ErrTypeRateLimit
	}
	if _, ok := err.(*RateLimitError); ok {
		return true
	}
	return false
}

// IsTokenLimitError checks if an error is a token limit error
func IsTokenLimitError(err error) bool {
	if pe, ok := err.(*ProviderError); ok {
		return pe.Type == ErrTypeTokenLimit
	}
	if _, ok := err.(*TokenLimitError); ok {
		return true
	}
	return false
}

// IsConfigurationError checks if an error is a configuration error
func IsConfigurationError(err error) bool {
	if pe, ok := err.(*ProviderError); ok {
		return pe.Type == ErrTypeConfiguration
	}
	if _, ok := err.(*ConfigurationError); ok {
		return true
	}
	return false
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if pe, ok := err.(*ProviderError); ok {
		return pe.Type == ErrTypeValidation
	}
	if _, ok := err.(*ValidationError); ok {
		return true
	}
	return false
}
