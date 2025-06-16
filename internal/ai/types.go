package ai

import (
	"time"
)

// CompletionRequest represents a request for text completion/analysis
type CompletionRequest struct {
	// Prompt is the input text for completion
	Prompt string `json:"prompt"`

	// Context provides additional context for the completion
	Context string `json:"context,omitempty"`

	// MaxTokens limits the response length
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 to 1.0)
	Temperature float64 `json:"temperature,omitempty"`

	// SystemPrompt provides system-level instructions
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Model specifies which model to use (provider-specific)
	Model string `json:"model,omitempty"`

	// Stream indicates if streaming response is requested
	Stream bool `json:"stream,omitempty"`

	// Metadata for request tracking
	RequestID string            `json:"request_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// CompletionResponse represents the response from a completion request
type CompletionResponse struct {
	// Content is the generated text
	Content string `json:"content"`

	// FinishReason indicates why the completion finished
	FinishReason string `json:"finish_reason"`

	// Usage contains token usage information
	Usage *TokenUsage `json:"usage"`

	// Model indicates which model was used
	Model string `json:"model"`

	// RequestID matches the original request
	RequestID string `json:"request_id,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// Provider-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens
	TotalTokens int `json:"total_tokens"`
}

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	// Name is the provider identifier
	Name string `json:"name"`

	// Type is the provider type (openai, anthropic, local, etc.)
	Type string `json:"type"`

	// APIKey for authentication
	APIKey string `json:"api_key,omitempty"`

	// BaseURL for the API endpoint
	BaseURL string `json:"base_url,omitempty"`

	// DefaultModel is the default model to use
	DefaultModel string `json:"default_model,omitempty"`

	// MaxTokens is the maximum context window
	MaxTokens int `json:"max_tokens,omitempty"`

	// DefaultTemperature for requests
	DefaultTemperature float64 `json:"default_temperature,omitempty"`

	// Timeout for requests
	Timeout time.Duration `json:"timeout,omitempty"`

	// RateLimit configuration
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`

	// RetryConfig for handling failures
	RetryConfig *RetryConfig `json:"retry_config,omitempty"`

	// Custom headers for requests
	Headers map[string]string `json:"headers,omitempty"`

	// Provider-specific options
	Options map[string]interface{} `json:"options,omitempty"`
}

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	// RequestsPerMinute limits requests per minute
	RequestsPerMinute int `json:"requests_per_minute"`

	// TokensPerMinute limits tokens per minute
	TokensPerMinute int `json:"tokens_per_minute"`

	// BurstSize allows burst requests
	BurstSize int `json:"burst_size"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries"`

	// InitialDelay is the initial delay between retries
	InitialDelay time.Duration `json:"initial_delay"`

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration `json:"max_delay"`

	// BackoffMultiplier for exponential backoff
	BackoffMultiplier float64 `json:"backoff_multiplier"`

	// RetryableErrors defines which errors should trigger retries
	RetryableErrors []string `json:"retryable_errors,omitempty"`
}

// AnalysisRequest represents a request for log analysis
type AnalysisRequest struct {
	// LogEntries to analyze
	LogEntries []string `json:"log_entries"`

	// AnalysisType specifies the type of analysis
	AnalysisType AnalysisType `json:"analysis_type"`

	// Context provides domain-specific context
	Context string `json:"context,omitempty"`

	// MaxResults limits the number of results
	MaxResults int `json:"max_results,omitempty"`

	// Confidence threshold for results
	MinConfidence float64 `json:"min_confidence,omitempty"`
}

// AnalysisType defines types of log analysis
type AnalysisType string

const (
	AnalysisTypeErrorDiagnosis    AnalysisType = "error_diagnosis"
	AnalysisTypeRootCause         AnalysisType = "root_cause"
	AnalysisTypeAnomalyDetection  AnalysisType = "anomaly_detection"
	AnalysisTypePatternExtraction AnalysisType = "pattern_extraction"
	AnalysisTypeInsightGeneration AnalysisType = "insight_generation"
)

// AnalysisResult represents the result of log analysis
type AnalysisResult struct {
	// Type of analysis performed
	Type AnalysisType `json:"type"`

	// Summary of findings
	Summary string `json:"summary"`

	// Detailed findings
	Findings []Finding `json:"findings"`

	// Confidence score (0.0 to 1.0)
	Confidence float64 `json:"confidence"`

	// Recommendations for action
	Recommendations []string `json:"recommendations,omitempty"`

	// Related log entries
	RelatedEntries []string `json:"related_entries,omitempty"`
}

// Finding represents a specific analysis finding
type Finding struct {
	// Title of the finding
	Title string `json:"title"`

	// Description with details
	Description string `json:"description"`

	// Severity level
	Severity string `json:"severity"`

	// Confidence in this finding
	Confidence float64 `json:"confidence"`

	// Evidence supporting the finding
	Evidence []string `json:"evidence,omitempty"`

	// Tags for categorization
	Tags []string `json:"tags,omitempty"`
}

// Model represents an AI model with provider-agnostic information
type Model struct {
	// ID is the unique identifier for the model
	ID string `json:"id"`

	// Name is the human-readable name of the model
	Name string `json:"name"`

	// Description provides details about the model
	Description string `json:"description"`

	// Provider is the name of the provider offering this model
	Provider string `json:"provider"`

	// MaxTokens is the maximum context window size for this model
	MaxTokens int `json:"max_tokens"`

	// CreatedAt is when the model was created (optional)
	CreatedAt time.Time `json:"created_at,omitempty"`

	// OwnedBy indicates who owns or maintains the model
	OwnedBy string `json:"owned_by,omitempty"`

	// Tags for categorization
	Tags []string `json:"tags,omitempty"`
}
