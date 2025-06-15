package ai

import (
	"context"
	"sync"
)

// Registry manages available LLM providers
type Registry interface {
	// Register adds a provider to the registry
	Register(name string, factory ProviderFactory) error

	// Unregister removes a provider from the registry
	Unregister(name string) error

	// Get retrieves a provider by name, creating it if necessary
	Get(name string) (Provider, error)

	// GetWithConfig retrieves a provider with specific configuration
	GetWithConfig(name string, config *ProviderConfig) (Provider, error)

	// List returns all registered provider names
	List() []string

	// IsRegistered checks if a provider is registered
	IsRegistered(name string) bool

	// Default returns the default provider
	Default() (Provider, error)

	// SetDefault sets the default provider
	SetDefault(name string) error

	// Close shuts down all providers and cleans up resources
	Close() error
}

// ProviderFactory creates provider instances
type ProviderFactory interface {
	// Create creates a new provider instance with the given config
	Create(config *ProviderConfig) (Provider, error)

	// Type returns the provider type this factory creates
	Type() string

	// ValidateConfig validates configuration for this provider type
	ValidateConfig(config *ProviderConfig) error

	// DefaultConfig returns a default configuration
	DefaultConfig() *ProviderConfig
}

// ProviderManager extends Registry with additional management features
type ProviderManager interface {
	Registry

	// HealthCheck performs health checks on all providers
	HealthCheck(ctx context.Context) map[string]error

	// GetHealthy returns only healthy providers
	GetHealthy() []string

	// Metrics returns usage metrics for providers
	Metrics() map[string]*ProviderMetrics

	// LoadBalance selects a provider based on load balancing strategy
	LoadBalance(strategy LoadBalanceStrategy) (Provider, error)

	// Warmup pre-initializes providers for faster access
	Warmup(ctx context.Context) error
}

// LoadBalanceStrategy defines load balancing strategies
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin LoadBalanceStrategy = "round_robin"
	LoadBalanceLeastUsed  LoadBalanceStrategy = "least_used"
	LoadBalanceRandom     LoadBalanceStrategy = "random"
	LoadBalanceHealthy    LoadBalanceStrategy = "healthy_only"
)

// ProviderMetrics tracks provider usage statistics
type ProviderMetrics struct {
	// TotalRequests is the total number of requests made
	TotalRequests int64 `json:"total_requests"`

	// SuccessfulRequests is the number of successful requests
	SuccessfulRequests int64 `json:"successful_requests"`

	// FailedRequests is the number of failed requests
	FailedRequests int64 `json:"failed_requests"`

	// TotalTokensUsed is the total number of tokens consumed
	TotalTokensUsed int64 `json:"total_tokens_used"`

	// AverageResponseTime in milliseconds
	AverageResponseTime float64 `json:"average_response_time_ms"`

	// LastUsed timestamp of last request
	LastUsed int64 `json:"last_used"`

	// IsHealthy indicates current health status
	IsHealthy bool `json:"is_healthy"`

	// ErrorRate is the percentage of failed requests
	ErrorRate float64 `json:"error_rate"`
}

// defaultRegistry implements Registry interface
type defaultRegistry struct {
	mu              sync.RWMutex
	factories       map[string]ProviderFactory
	providers       map[string]Provider
	configs         map[string]*ProviderConfig
	defaultProvider string
	metrics         map[string]*ProviderMetrics
}

// NewRegistry creates a new provider registry
func NewRegistry() Registry {
	return &defaultRegistry{
		factories: make(map[string]ProviderFactory),
		providers: make(map[string]Provider),
		configs:   make(map[string]*ProviderConfig),
		metrics:   make(map[string]*ProviderMetrics),
	}
}

// Register adds a provider factory to the registry
func (r *defaultRegistry) Register(name string, factory ProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return &ProviderError{
			Type:     ErrTypeRegistration,
			Message:  "provider already registered",
			Provider: name,
		}
	}

	r.factories[name] = factory
	r.metrics[name] = &ProviderMetrics{}
	return nil
}

// Unregister removes a provider from the registry
func (r *defaultRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider, exists := r.providers[name]; exists {
		if err := provider.Close(); err != nil {
			return err
		}
		delete(r.providers, name)
	}

	delete(r.factories, name)
	delete(r.configs, name)
	delete(r.metrics, name)

	if r.defaultProvider == name {
		r.defaultProvider = ""
	}

	return nil
}

// Get retrieves a provider by name
func (r *defaultRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	if provider, exists := r.providers[name]; exists {
		r.mu.RUnlock()
		return provider, nil
	}

	factory, exists := r.factories[name]
	config := r.configs[name]
	r.mu.RUnlock()

	if !exists {
		return nil, &ProviderError{
			Type:     ErrTypeNotFound,
			Message:  "provider not registered",
			Provider: name,
		}
	}

	if config == nil {
		config = factory.DefaultConfig()
	}

	return r.GetWithConfig(name, config)
}

// GetWithConfig retrieves a provider with specific configuration
func (r *defaultRegistry) GetWithConfig(name string, config *ProviderConfig) (Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, &ProviderError{
			Type:     ErrTypeNotFound,
			Message:  "provider not registered",
			Provider: name,
		}
	}

	if err := factory.ValidateConfig(config); err != nil {
		return nil, err
	}

	provider, err := factory.Create(config)
	if err != nil {
		return nil, err
	}

	r.providers[name] = provider
	r.configs[name] = config

	return provider, nil
}

// List returns all registered provider names
func (r *defaultRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered
func (r *defaultRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// Default returns the default provider
func (r *defaultRegistry) Default() (Provider, error) {
	r.mu.RLock()
	defaultName := r.defaultProvider
	r.mu.RUnlock()

	if defaultName == "" {
		return nil, &ProviderError{
			Type:    ErrTypeConfiguration,
			Message: "no default provider set",
		}
	}

	return r.Get(defaultName)
}

// SetDefault sets the default provider
func (r *defaultRegistry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; !exists {
		return &ProviderError{
			Type:     ErrTypeNotFound,
			Message:  "provider not registered",
			Provider: name,
		}
	}

	r.defaultProvider = name
	return nil
}

// Close shuts down all providers
func (r *defaultRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for name, provider := range r.providers {
		if err := provider.Close(); err != nil {
			lastErr = err
		}
		delete(r.providers, name)
	}

	return lastErr
}

// Global registry instance
var globalRegistry = NewRegistry()

// GlobalRegistry returns the global provider registry
func GlobalRegistry() Registry {
	return globalRegistry
}

// RegisterProvider registers a provider in the global registry
func RegisterProvider(name string, factory ProviderFactory) error {
	return globalRegistry.Register(name, factory)
}

// GetProvider retrieves a provider from the global registry
func GetProvider(name string) (Provider, error) {
	return globalRegistry.Get(name)
}
