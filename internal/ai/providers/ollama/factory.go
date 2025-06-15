package ollama

import (
	"github.com/yildizm/LogSum/internal/ai"
)

// Factory implements the ProviderFactory interface for Ollama
type Factory struct{}

// NewFactory creates a new Ollama provider factory
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a new Ollama provider instance with the given config
func (f *Factory) Create(config *ai.ProviderConfig) (ai.Provider, error) {
	if config == nil {
		config = f.DefaultConfig()
	}

	ollamaConfig := FromProviderConfig(config)
	return New(ollamaConfig)
}

// Type returns the provider type this factory creates
func (f *Factory) Type() string {
	return "ollama"
}

// ValidateConfig validates configuration for this provider type
func (f *Factory) ValidateConfig(config *ai.ProviderConfig) error {
	if config == nil {
		return ai.NewConfigurationError("ollama", "config", "configuration is required")
	}

	if config.Type != "" && config.Type != "ollama" {
		return ai.NewConfigurationError("ollama", "type", "invalid provider type: expected 'ollama'")
	}

	ollamaConfig := FromProviderConfig(config)
	return ollamaConfig.Validate()
}

// DefaultConfig returns a default configuration
func (f *Factory) DefaultConfig() *ai.ProviderConfig {
	ollamaConfig := DefaultConfig()
	return ollamaConfig.ToProviderConfig()
}

// Register registers the Ollama provider with the global registry
func Register() error {
	factory := NewFactory()
	return ai.RegisterProvider("ollama", factory)
}
