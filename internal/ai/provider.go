package ai

import (
	"context"
	"io"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// Name returns the provider name (e.g., "openai", "anthropic", "local")
	Name() string

	// Complete performs text completion/analysis
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// CompleteStream performs streaming text completion
	CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)

	// CountTokens estimates token count for the given text
	CountTokens(text string) (int, error)

	// MaxTokens returns the maximum context window size
	MaxTokens() int

	// SupportsStreaming indicates if the provider supports streaming
	SupportsStreaming() bool

	// ValidateConfig validates the provider configuration
	ValidateConfig() error

	// Close cleans up provider resources
	Close() error
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// ContextManager handles context window management
type ContextManager interface {
	// TruncateToFit truncates text to fit within token limits
	TruncateToFit(text string, maxTokens int) (string, error)

	// SplitByTokens splits text into chunks by token count
	SplitByTokens(text string, chunkSize int) ([]string, error)

	// EstimateTokens provides a rough token count estimate
	EstimateTokens(text string) int
}

// RateLimiter handles rate limiting for provider requests
type RateLimiter interface {
	// Allow checks if a request is allowed under rate limits
	Allow(ctx context.Context, tokens int) error

	// Wait blocks until the next request is allowed
	Wait(ctx context.Context, tokens int) error

	// Reset resets the rate limiter state
	Reset()
}

// HealthChecker provides health checking capabilities
type HealthChecker interface {
	// HealthCheck verifies provider connectivity and status
	HealthCheck(ctx context.Context) error

	// IsHealthy returns current health status
	IsHealthy() bool
}

// Provider combines all provider capabilities
type Provider interface {
	LLMProvider
	ContextManager
	HealthChecker
	io.Closer
}
