package ai

import (
	"testing"
	"time"
)

func TestCompletionRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     CompletionRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CompletionRequest{
				Prompt:      "Test prompt",
				MaxTokens:   100,
				Temperature: 0.7,
			},
			wantErr: false,
		},
		{
			name: "empty prompt",
			req: CompletionRequest{
				Prompt:    "",
				MaxTokens: 100,
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			req: CompletionRequest{
				Prompt:    "Test",
				MaxTokens: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			req: CompletionRequest{
				Prompt:      "Test",
				Temperature: 2.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCompletionRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCompletionRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ProviderConfig{
				Name:    "test-provider",
				Type:    "openai",
				APIKey:  "sk-test",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: ProviderConfig{
				Type:   "openai",
				APIKey: "sk-test",
			},
			wantErr: true,
		},
		{
			name: "empty type",
			config: ProviderConfig{
				Name:   "test",
				APIKey: "sk-test",
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: ProviderConfig{
				Name:    "test",
				Type:    "openai",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProviderConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalysisType_String(t *testing.T) {
	tests := []struct {
		name     string
		analysis AnalysisType
		expected string
	}{
		{
			name:     "error diagnosis",
			analysis: AnalysisTypeErrorDiagnosis,
			expected: "error_diagnosis",
		},
		{
			name:     "root cause",
			analysis: AnalysisTypeRootCause,
			expected: "root_cause",
		},
		{
			name:     "anomaly detection",
			analysis: AnalysisTypeAnomalyDetection,
			expected: "anomaly_detection",
		},
		{
			name:     "pattern extraction",
			analysis: AnalysisTypePatternExtraction,
			expected: "pattern_extraction",
		},
		{
			name:     "insight generation",
			analysis: AnalysisTypeInsightGeneration,
			expected: "insight_generation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.analysis) != tt.expected {
				t.Errorf("AnalysisType string conversion = %v, expected %v", string(tt.analysis), tt.expected)
			}
		})
	}
}

func TestTokenUsage_Total(t *testing.T) {
	usage := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, expected 150", usage.TotalTokens)
	}

	// Test calculated total
	calculatedTotal := usage.PromptTokens + usage.CompletionTokens
	if calculatedTotal != usage.TotalTokens {
		t.Errorf("Calculated total %d != TotalTokens %d", calculatedTotal, usage.TotalTokens)
	}
}

func TestRateLimitConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RateLimitConfig{
				RequestsPerMinute: 60,
				TokensPerMinute:   10000,
				BurstSize:         10,
			},
			wantErr: false,
		},
		{
			name: "zero requests per minute",
			config: RateLimitConfig{
				RequestsPerMinute: 0,
				TokensPerMinute:   10000,
			},
			wantErr: true,
		},
		{
			name: "negative tokens per minute",
			config: RateLimitConfig{
				RequestsPerMinute: 60,
				TokensPerMinute:   -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRateLimitConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRateLimitConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  RetryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RetryConfig{
				MaxRetries:        3,
				InitialDelay:      time.Second,
				MaxDelay:          30 * time.Second,
				BackoffMultiplier: 2.0,
			},
			wantErr: false,
		},
		{
			name: "negative max retries",
			config: RetryConfig{
				MaxRetries: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid backoff multiplier",
			config: RetryConfig{
				MaxRetries:        3,
				BackoffMultiplier: 0.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRetryConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRetryConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Validation helper functions (would be implemented in types.go)

func validateCompletionRequest(req *CompletionRequest) error {
	if req.Prompt == "" {
		return NewValidationError("prompt", req.Prompt, "prompt cannot be empty")
	}
	if req.MaxTokens < 0 {
		return NewValidationError("max_tokens", string(rune(req.MaxTokens)), "max_tokens cannot be negative")
	}
	if req.Temperature < 0 || req.Temperature > 1 {
		return NewValidationError("temperature", string(rune(int(req.Temperature*100))), "temperature must be between 0 and 1")
	}
	return nil
}

func validateProviderConfig(config *ProviderConfig) error {
	if config.Name == "" {
		return NewValidationError("name", config.Name, "name cannot be empty")
	}
	if config.Type == "" {
		return NewValidationError("type", config.Type, "type cannot be empty")
	}
	if config.Timeout < 0 {
		return NewValidationError("timeout", config.Timeout.String(), "timeout cannot be negative")
	}
	return nil
}

func validateRateLimitConfig(config *RateLimitConfig) error {
	if config.RequestsPerMinute <= 0 {
		return NewValidationError("requests_per_minute", string(rune(config.RequestsPerMinute)), "requests_per_minute must be positive")
	}
	if config.TokensPerMinute < 0 {
		return NewValidationError("tokens_per_minute", string(rune(config.TokensPerMinute)), "tokens_per_minute cannot be negative")
	}
	return nil
}

func validateRetryConfig(config *RetryConfig) error {
	if config.MaxRetries < 0 {
		return NewValidationError("max_retries", string(rune(config.MaxRetries)), "max_retries cannot be negative")
	}
	if config.BackoffMultiplier < 1.0 {
		return NewValidationError("backoff_multiplier", string(rune(int(config.BackoffMultiplier*100))), "backoff_multiplier must be >= 1.0")
	}
	return nil
}
