package ollama

import (
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

// Config holds Ollama-specific configuration
type Config struct {
	// BaseURL is the Ollama API endpoint
	BaseURL string `json:"base_url"`

	// DefaultModel is the default model to use if none specified
	DefaultModel string `json:"default_model"`

	// Timeout for HTTP requests
	Timeout time.Duration `json:"timeout"`

	// MaxTokens is the maximum context window size
	MaxTokens int `json:"max_tokens"`

	// DefaultTemperature for requests
	DefaultTemperature float64 `json:"default_temperature"`

	// PullTimeout for model pull operations
	PullTimeout time.Duration `json:"pull_timeout"`

	// HealthCheckInterval for periodic health checks
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// RetryAttempts for failed requests
	RetryAttempts int `json:"retry_attempts"`

	// RetryDelay between retry attempts
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultConfig returns a default Ollama configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:             "http://localhost:11434",
		DefaultModel:        "llama2",
		Timeout:             30 * time.Second,
		MaxTokens:           4096,
		DefaultTemperature:  0.7,
		PullTimeout:         10 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return ai.NewConfigurationError("ollama", "base_url", "base URL is required")
	}

	if c.DefaultModel == "" {
		return ai.NewConfigurationError("ollama", "default_model", "default model is required")
	}

	if c.Timeout <= 0 {
		return ai.NewConfigurationError("ollama", "timeout", "timeout must be positive")
	}

	if c.MaxTokens <= 0 {
		return ai.NewConfigurationError("ollama", "max_tokens", "max tokens must be positive")
	}

	if c.DefaultTemperature < 0 || c.DefaultTemperature > 1 {
		return ai.NewConfigurationError("ollama", "default_temperature", "temperature must be between 0 and 1")
	}

	return nil
}

// ToProviderConfig converts Ollama config to generic provider config
func (c *Config) ToProviderConfig() *ai.ProviderConfig {
	return &ai.ProviderConfig{
		Name:               "ollama",
		Type:               "ollama",
		BaseURL:            c.BaseURL,
		DefaultModel:       c.DefaultModel,
		MaxTokens:          c.MaxTokens,
		DefaultTemperature: c.DefaultTemperature,
		Timeout:            c.Timeout,
		Options: map[string]interface{}{
			"pull_timeout":          c.PullTimeout,
			"health_check_interval": c.HealthCheckInterval,
			"retry_attempts":        c.RetryAttempts,
			"retry_delay":           c.RetryDelay,
		},
	}
}

// FromProviderConfig creates Ollama config from generic provider config
func FromProviderConfig(pc *ai.ProviderConfig) *Config {
	config := DefaultConfig()

	if pc.BaseURL != "" {
		config.BaseURL = pc.BaseURL
	}

	if pc.DefaultModel != "" {
		config.DefaultModel = pc.DefaultModel
	}

	if pc.MaxTokens > 0 {
		config.MaxTokens = pc.MaxTokens
	}

	if pc.DefaultTemperature >= 0 && pc.DefaultTemperature <= 1 {
		config.DefaultTemperature = pc.DefaultTemperature
	}

	if pc.Timeout > 0 {
		config.Timeout = pc.Timeout
	}

	if pc.Options != nil {
		if pullTimeout, ok := pc.Options["pull_timeout"].(time.Duration); ok {
			config.PullTimeout = pullTimeout
		}

		if healthCheckInterval, ok := pc.Options["health_check_interval"].(time.Duration); ok {
			config.HealthCheckInterval = healthCheckInterval
		}

		if retryAttempts, ok := pc.Options["retry_attempts"].(int); ok {
			config.RetryAttempts = retryAttempts
		}

		if retryDelay, ok := pc.Options["retry_delay"].(time.Duration); ok {
			config.RetryDelay = retryDelay
		}
	}

	return config
}
