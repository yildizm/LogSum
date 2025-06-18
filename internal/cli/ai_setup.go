package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/ai/providers/ollama"
	"github.com/yildizm/LogSum/internal/ai/providers/openai"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/config"
	"github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/LogSum/internal/docstore"
)

// CorrelatorAdapter adapts the correlation.Correlator interface to work with
// the AI analyzer's correlation requirements, providing a bridge between
// the correlation package and AI analysis functionality.
type CorrelatorAdapter struct {
	correlator correlation.Correlator
}

// Correlate performs document correlation analysis using the wrapped correlator.
func (ca *CorrelatorAdapter) Correlate(ctx context.Context, analysis *common.Analysis) (any, error) {
	return ca.correlator.Correlate(ctx, analysis)
}

// SetDocumentStore sets the document store for the correlator.
func (ca *CorrelatorAdapter) SetDocumentStore(store any) error {
	if docStore, ok := store.(docstore.DocumentStore); ok {
		return ca.correlator.SetDocumentStore(docStore)
	}
	return fmt.Errorf("invalid document store type")
}

// AIAnalysisConfig holds configuration for AI-enhanced analysis.
type AIAnalysisConfig struct {
	Provider                ai.Provider
	MaxTokensPerRequest     int
	EnableErrorAnalysis     bool
	EnableRootCauseAnalysis bool
	EnableRecommendations   bool
	IncludeContext          bool
	EnableDocumentContext   bool
	MaxContextTokens        int
	MinConfidence           float64
	MaxConcurrentRequests   int
}

// DefaultAIAnalysisConfig returns sensible defaults for AI analysis.
func DefaultAIAnalysisConfig() *AIAnalysisConfig {
	return &AIAnalysisConfig{
		MaxTokensPerRequest:     2000,
		EnableErrorAnalysis:     true,
		EnableRootCauseAnalysis: true,
		EnableRecommendations:   true,
		IncludeContext:          true,
		EnableDocumentContext:   false,
		MaxContextTokens:        1000,
		MinConfidence:           0.6,
		MaxConcurrentRequests:   3,
	}
}

// performAIAnalysis runs AI-enhanced analysis using the provided configuration.
func performAIAnalysis(ctx context.Context, baseEngine analyzer.Analyzer, entries []*common.LogEntry) (*analyzer.Analysis, error) {
	cfg := GetGlobalConfig()

	// Create AI provider
	provider, err := createAIProvider(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI provider: %w", err)
	}

	// Create AI analysis configuration
	aiConfig := DefaultAIAnalysisConfig()
	aiConfig.Provider = provider
	aiConfig.EnableDocumentContext = analyzeCorrelate && analyzeDocsPath != ""

	// Create AI analyzer
	aiAnalyzer, err := createAIAnalyzer(baseEngine, aiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI analyzer: %w", err)
	}

	// Set up document correlation if enabled
	if aiConfig.EnableDocumentContext {
		if err := setupAIDocumentCorrelation(ctx, aiAnalyzer); err != nil {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: failed to setup document correlation: %v\n", err)
			}
		}
	}

	// Perform AI analysis
	aiResult, err := aiAnalyzer.AnalyzeWithAI(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Return the base analysis from AI result
	return aiResult.Analysis, nil
}

// createAIAnalyzer creates an AI analyzer with the provided configuration.
func createAIAnalyzer(baseEngine analyzer.Analyzer, analysisConfig *AIAnalysisConfig) (*analyzer.AIAnalyzer, error) {
	if analysisConfig.Provider == nil {
		return nil, fmt.Errorf("AI provider is required")
	}

	// Convert to analyzer options
	aiOptions := &analyzer.AIAnalyzerOptions{
		Provider:                analysisConfig.Provider,
		MaxTokensPerRequest:     analysisConfig.MaxTokensPerRequest,
		EnableErrorAnalysis:     analysisConfig.EnableErrorAnalysis,
		EnableRootCauseAnalysis: analysisConfig.EnableRootCauseAnalysis,
		EnableRecommendations:   analysisConfig.EnableRecommendations,
		IncludeContext:          analysisConfig.IncludeContext,
		EnableDocumentContext:   analysisConfig.EnableDocumentContext,
		MaxContextTokens:        analysisConfig.MaxContextTokens,
		MinConfidence:           analysisConfig.MinConfidence,
		MaxConcurrentRequests:   analysisConfig.MaxConcurrentRequests,
	}

	return analyzer.NewAIAnalyzer(baseEngine, aiOptions), nil
}

// createAIProvider creates an AI provider based on configuration.
func createAIProvider(aiConfig *config.AIConfig) (ai.Provider, error) {
	switch strings.ToLower(aiConfig.Provider) {
	case "openai":
		return createOpenAIProvider(aiConfig)
	case "ollama":
		return createOllamaProvider(aiConfig)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", aiConfig.Provider)
	}
}

// createOpenAIProvider creates an OpenAI provider with configuration.
func createOpenAIProvider(aiConfig *config.AIConfig) (ai.Provider, error) {
	openaiConfig := &openai.Config{
		APIKey:       aiConfig.APIKey,
		BaseURL:      aiConfig.Endpoint,
		DefaultModel: aiConfig.Model,
		MaxTokens:    openai.DefaultMaxTokens,
		Timeout:      aiConfig.Timeout,
	}

	// Apply defaults if not configured
	if openaiConfig.BaseURL == "" {
		openaiConfig.BaseURL = "https://api.openai.com"
	}
	if openaiConfig.DefaultModel == "" {
		openaiConfig.DefaultModel = "gpt-3.5-turbo"
	}

	return openai.New(openaiConfig)
}

// createOllamaProvider creates an Ollama provider with configuration.
func createOllamaProvider(aiConfig *config.AIConfig) (ai.Provider, error) {
	ollamaConfig := &ollama.Config{
		BaseURL:             aiConfig.Endpoint,
		DefaultModel:        aiConfig.Model,
		Timeout:             aiConfig.Timeout,
		MaxTokens:           4096,
		DefaultTemperature:  0.7,
		PullTimeout:         10 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
	}

	// Apply defaults if not configured
	if ollamaConfig.BaseURL == "" {
		ollamaConfig.BaseURL = "http://localhost:11434"
	}
	if ollamaConfig.DefaultModel == "" {
		ollamaConfig.DefaultModel = "llama3.2"
	}

	return ollama.New(ollamaConfig)
}

// setupAIDocumentCorrelation sets up document correlation for AI analysis.
func setupAIDocumentCorrelation(ctx context.Context, aiAnalyzer *analyzer.AIAnalyzer) error {
	// Create document store using the existing setupDocumentStore function
	store, err := setupDocumentStore(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup document store: %w", err)
	}

	// Create correlator adapter
	correlator := correlation.NewCorrelator()
	adapter := &CorrelatorAdapter{correlator: correlator}

	// Set up AI analyzer with correlation
	aiAnalyzer.SetCorrelator(adapter)
	return aiAnalyzer.SetDocumentStore(store)
}
