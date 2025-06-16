package openai

import (
	"fmt"
	"net/url"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

const (
	DefaultBaseURL     = "https://api.openai.com"
	DefaultModel       = "gpt-3.5-turbo"
	DefaultMaxTokens   = 4096
	DefaultTemperature = 0.7
	DefaultTimeout     = 30 * time.Second
)

type Config struct {
	APIKey             string        `json:"api_key"`
	BaseURL            string        `json:"base_url"`
	DefaultModel       string        `json:"default_model"`
	MaxTokens          int           `json:"max_tokens"`
	DefaultTemperature float64       `json:"default_temperature"`
	Timeout            time.Duration `json:"timeout"`
	OrganizationID     string        `json:"organization_id,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:            DefaultBaseURL,
		DefaultModel:       DefaultModel,
		MaxTokens:          DefaultMaxTokens,
		DefaultTemperature: DefaultTemperature,
		Timeout:            DefaultTimeout,
	}
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return ai.NewConfigurationError("openai", "api_key", "API key is required")
	}

	if c.BaseURL == "" {
		return ai.NewConfigurationError("openai", "base_url", "base URL is required")
	}

	if _, err := url.Parse(c.BaseURL); err != nil {
		return ai.NewConfigurationError("openai", "base_url", fmt.Sprintf("invalid base URL: %v", err))
	}

	if c.DefaultModel == "" {
		return ai.NewConfigurationError("openai", "default_model", "default model is required")
	}

	if c.MaxTokens <= 0 {
		return ai.NewConfigurationError("openai", "max_tokens", "max tokens must be positive")
	}

	if c.DefaultTemperature < 0 || c.DefaultTemperature > 2 {
		return ai.NewConfigurationError("openai", "default_temperature", "temperature must be between 0 and 2")
	}

	if c.Timeout <= 0 {
		return ai.NewConfigurationError("openai", "timeout", "timeout must be positive")
	}

	return nil
}

func (c *Config) ToProviderConfig() *ai.ProviderConfig {
	headers := map[string]string{
		"Authorization": "Bearer " + c.APIKey,
		"Content-Type":  "application/json",
	}

	if c.OrganizationID != "" {
		headers["OpenAI-Organization"] = c.OrganizationID
	}

	return &ai.ProviderConfig{
		Name:               "openai",
		Type:               "openai",
		APIKey:             c.APIKey,
		BaseURL:            c.BaseURL,
		DefaultModel:       c.DefaultModel,
		MaxTokens:          c.MaxTokens,
		DefaultTemperature: c.DefaultTemperature,
		Timeout:            c.Timeout,
		Headers:            headers,
		Options: map[string]interface{}{
			"organization_id": c.OrganizationID,
		},
	}
}

func FromProviderConfig(config *ai.ProviderConfig) *Config {
	if config == nil {
		return DefaultConfig()
	}

	c := &Config{
		APIKey:             config.APIKey,
		BaseURL:            config.BaseURL,
		DefaultModel:       config.DefaultModel,
		MaxTokens:          config.MaxTokens,
		DefaultTemperature: config.DefaultTemperature,
		Timeout:            config.Timeout,
	}

	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.DefaultModel == "" {
		c.DefaultModel = DefaultModel
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = DefaultMaxTokens
	}
	if c.DefaultTemperature == 0 {
		c.DefaultTemperature = DefaultTemperature
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}

	if config.Options != nil {
		if orgID, ok := config.Options["organization_id"].(string); ok {
			c.OrganizationID = orgID
		}
	}

	return c
}
