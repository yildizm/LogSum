package openai

import (
	"github.com/yildizm/LogSum/internal/ai"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(config *ai.ProviderConfig) (ai.Provider, error) {
	if config == nil {
		config = f.DefaultConfig()
	}

	providerConfig := FromProviderConfig(config)
	return New(providerConfig)
}

func (f *Factory) Type() string {
	return "openai"
}

func (f *Factory) ValidateConfig(config *ai.ProviderConfig) error {
	if config == nil {
		return ai.NewConfigurationError("openai", "config", "configuration is required")
	}

	providerConfig := FromProviderConfig(config)
	return providerConfig.Validate()
}

func (f *Factory) DefaultConfig() *ai.ProviderConfig {
	return DefaultConfig().ToProviderConfig()
}

func (f *Factory) GetModels(config *ai.ProviderConfig) ([]ai.Model, error) {
	provider, err := f.Create(config)
	if err != nil {
		return nil, err
	}
	defer func() { _ = provider.Close() }()

	openaiProvider, ok := provider.(*Provider)
	if !ok {
		return nil, ai.NewProviderError(ai.ErrTypeInternal, "invalid provider type", "openai")
	}

	return openaiProvider.GetModels()
}

func Register() error {
	factory := NewFactory()
	return ai.RegisterProvider("openai", factory)
}
