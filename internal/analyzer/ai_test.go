package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/common"
	correlationpkg "github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/go-logparser"
)

// MockProvider implements the ai.Provider interface for testing
type MockProvider struct {
	name            string
	completionFunc  func(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error)
	maxTokens       int
	healthCheckFunc func(ctx context.Context) error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if m.completionFunc != nil {
		return m.completionFunc(ctx, req)
	}

	return &ai.CompletionResponse{
		Content:      "Mock AI response for: " + req.Prompt[:minInt(50, len(req.Prompt))],
		FinishReason: "stop",
		Usage: &ai.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Model:     "mock-model",
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockProvider) CompleteStream(ctx context.Context, req *ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	ch := make(chan ai.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- ai.StreamChunk{
			Content: "Mock streaming response",
			Done:    true,
		}
	}()
	return ch, nil
}

func (m *MockProvider) CountTokens(text string) (int, error) {
	return len(text) / 4, nil // Rough estimate
}

func (m *MockProvider) MaxTokens() int {
	if m.maxTokens > 0 {
		return m.maxTokens
	}
	return 4096
}

func (m *MockProvider) SupportsStreaming() bool {
	return true
}

func (m *MockProvider) ValidateConfig() error {
	return nil
}

func (m *MockProvider) Close() error {
	return nil
}

func (m *MockProvider) TruncateToFit(text string, maxTokens int) (string, error) {
	if len(text) <= maxTokens*4 {
		return text, nil
	}
	return text[:maxTokens*4], nil
}

func (m *MockProvider) SplitByTokens(text string, chunkSize int) ([]string, error) {
	chunks := make([]string, 0)
	for i := 0; i < len(text); i += chunkSize * 4 {
		end := i + chunkSize*4
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks, nil
}

func (m *MockProvider) EstimateTokens(text string) int {
	return len(text) / 4
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

func (m *MockProvider) IsHealthy() bool {
	return true
}

func createTestLogEntries() []*common.LogEntry {
	return []*common.LogEntry{
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-10 * time.Minute),
				Level:     "INFO",
				Message:   "Application started successfully",
			},
			LogLevel:   common.LevelInfo,
			Source:     "app.go:123",
			LineNumber: 1,
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-8 * time.Minute),
				Level:     "ERROR",
				Message:   "Database connection failed: timeout after 30s",
			},
			LogLevel:   common.LevelError,
			Source:     "db.go:45",
			LineNumber: 2,
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-5 * time.Minute),
				Level:     "WARN",
				Message:   "High memory usage detected: 85%",
			},
			LogLevel:   common.LevelWarn,
			Source:     "monitor.go:78",
			LineNumber: 3,
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-3 * time.Minute),
				Level:     "ERROR",
				Message:   "Failed to process request: invalid JSON",
			},
			LogLevel:   common.LevelError,
			Source:     "handler.go:234",
			LineNumber: 4,
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-1 * time.Minute),
				Level:     "INFO",
				Message:   "Request processed successfully",
			},
			LogLevel:   common.LevelInfo,
			Source:     "handler.go:245",
			LineNumber: 5,
		},
	}
}

func TestNewAIAnalyzer(t *testing.T) {
	mockProvider := &MockProvider{name: "test-provider"}
	baseAnalyzer := NewEngine() // Use the concrete implementation

	options := &AIAnalyzerOptions{
		Provider:                mockProvider,
		MaxTokensPerRequest:     1000,
		EnableErrorAnalysis:     true,
		EnableRootCauseAnalysis: true,
		EnableRecommendations:   true,
	}

	aiAnalyzer := NewAIAnalyzer(baseAnalyzer, options)

	if aiAnalyzer == nil {
		t.Fatal("Expected AIAnalyzer to be created, got nil")
		return
	}

	if aiAnalyzer.baseAnalyzer != baseAnalyzer {
		t.Error("Expected base analyzer to be set correctly")
	}

	if aiAnalyzer.options != options {
		t.Error("Expected options to be set correctly")
	}
}

func TestNewAIAnalyzerWithDefaults(t *testing.T) {
	baseAnalyzer := NewEngine()

	aiAnalyzer := NewAIAnalyzer(baseAnalyzer, nil)

	if aiAnalyzer == nil {
		t.Fatal("Expected AIAnalyzer to be created with defaults, got nil")
		return
	}

	if aiAnalyzer.options.MaxTokensPerRequest != 2000 {
		t.Errorf("Expected default MaxTokensPerRequest to be 2000, got %d", aiAnalyzer.options.MaxTokensPerRequest)
	}

	if !aiAnalyzer.options.EnableErrorAnalysis {
		t.Error("Expected EnableErrorAnalysis to be true by default")
	}
}

func TestGenerateSummary(t *testing.T) {
	mockProvider := &MockProvider{
		name: "test-provider",
		completionFunc: func(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
			return &ai.CompletionResponse{
				Content:      "Test summary of log analysis",
				FinishReason: "stop",
				Usage: &ai.TokenUsage{
					TotalTokens: 20,
				},
			}, nil
		},
	}

	options := &AIAnalyzerOptions{
		Provider: mockProvider,
	}

	aiAnalyzer := NewAIAnalyzer(NewEngine(), options)

	analysis := &common.Analysis{
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		TotalEntries: 100,
		ErrorCount:   5,
		WarnCount:    10,
	}

	summary, err := aiAnalyzer.generateSummary(context.Background(), analysis, createTestLogEntries(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	if summary != "Test summary of log analysis" {
		t.Errorf("Expected specific summary, got %s", summary)
	}
}

func TestExtractErrorEntries(t *testing.T) {
	aiAnalyzer := NewAIAnalyzer(NewEngine(), nil)
	entries := createTestLogEntries()

	errorEntries := aiAnalyzer.extractErrorEntries(entries)

	if len(errorEntries) != 2 {
		t.Errorf("Expected 2 error entries, got %d", len(errorEntries))
	}

	for _, entry := range errorEntries {
		if entry.LogLevel != common.LevelError && entry.LogLevel != common.LevelFatal {
			t.Errorf("Expected error or fatal level, got %s", entry.LogLevel)
		}
	}
}

func TestBuildSummaryPrompt(t *testing.T) {
	aiAnalyzer := NewAIAnalyzer(NewEngine(), nil)

	analysis := &common.Analysis{
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		TotalEntries: 100,
		ErrorCount:   5,
		WarnCount:    10,
		Patterns: []common.PatternMatch{
			{
				Pattern: &common.Pattern{Name: "Database Error"},
				Count:   3,
			},
		},
		Insights: []common.Insight{
			{
				Title:       "High Error Rate",
				Description: "Error rate increased by 50%",
			},
		},
	}

	prompt := aiAnalyzer.buildSummaryPrompt(analysis, createTestLogEntries(), nil)

	if prompt == nil {
		t.Error("Expected non-nil prompt")
		return
	}

	promptText := prompt.String()
	if !contains(promptText, "Total Entries: 100") {
		t.Error("Expected prompt to contain total entries")
	}

	if !contains(promptText, "Errors: 5, Warnings: 10") {
		t.Error("Expected prompt to contain error and warning counts")
	}

	if !contains(promptText, "Database Error") {
		t.Error("Expected prompt to contain pattern name")
	}

	if !contains(promptText, "High Error Rate") {
		t.Error("Expected prompt to contain insight title")
	}
}

func TestBuildErrorAnalysisPrompt(t *testing.T) {
	aiAnalyzer := NewAIAnalyzer(NewEngine(), nil)

	entries := createTestLogEntries()
	errorEntries := aiAnalyzer.extractErrorEntries(entries)

	analysis := &common.Analysis{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		ErrorCount: 2,
	}

	prompt := aiAnalyzer.buildErrorAnalysisPrompt(errorEntries, analysis, nil)

	if prompt == nil {
		t.Error("Expected non-nil prompt")
		return
	}

	promptText := prompt.String()
	if !contains(promptText, "Total Errors: 2") {
		t.Error("Expected prompt to contain error count")
	}

	if !contains(promptText, "Database connection failed") {
		t.Error("Expected prompt to contain sample error message")
	}
}

func TestAnalyzeErrorsWithJSONResponse(t *testing.T) {
	jsonResponse := `{"summary": "Multiple database connection failures detected", "critical_errors": [{"title": "Database Connection Timeout", "description": "Repeated connection timeouts to database", "severity": 3, "occurrences": 3, "confidence": 0.9, "first_seen": "2023-01-01T00:00:00Z", "last_seen": "2023-01-01T01:00:00Z"}], "error_patterns": [{"pattern": "connection timeout", "frequency": 3, "trend": "increasing"}], "severity_breakdown": {"error": 2, "warn": 1}}`

	mockProvider := &MockProvider{
		name: "test-provider",
		completionFunc: func(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
			return &ai.CompletionResponse{
				Content: jsonResponse,
			}, nil
		},
	}

	options := &AIAnalyzerOptions{
		Provider: mockProvider,
	}

	aiAnalyzer := NewAIAnalyzer(NewEngine(), options)

	analysis := &common.Analysis{
		ErrorCount: 2,
		WarnCount:  1,
	}

	errorAnalysis, err := aiAnalyzer.analyzeErrors(context.Background(), analysis, createTestLogEntries(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if errorAnalysis == nil {
		t.Fatal("Expected error analysis result, got nil")
	}

	if errorAnalysis.Summary != "Multiple database connection failures detected" {
		t.Errorf("Expected specific summary, got %s", errorAnalysis.Summary)
	}

	if len(errorAnalysis.CriticalErrors) != 1 {
		t.Errorf("Expected 1 critical error, got %d", len(errorAnalysis.CriticalErrors))
	}

	if errorAnalysis.SeverityBreakdown["error"] != 2 {
		t.Errorf("Expected 2 errors in breakdown, got %d", errorAnalysis.SeverityBreakdown["error"])
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIdentifyRootCauses(t *testing.T) {
	jsonResponse := `[
		{
			"title": "Database Connection Pool Exhaustion",
			"description": "Connection pool is exhausted due to high load",
			"confidence": 0.8,
			"category": "database",
			"impact": "high"
		},
		{
			"title": "Memory Leak in Handler",
			"description": "Possible memory leak causing high memory usage",
			"confidence": 0.6,
			"category": "application",
			"impact": "medium"
		}
	]`

	mockProvider := &MockProvider{
		name: "test-provider",
		completionFunc: func(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
			return &ai.CompletionResponse{
				Content: jsonResponse,
			}, nil
		},
	}

	options := &AIAnalyzerOptions{
		Provider:      mockProvider,
		MinConfidence: 0.7,
	}

	aiAnalyzer := NewAIAnalyzer(NewEngine(), options)

	analysis := &common.Analysis{
		ErrorCount: 2,
	}

	rootCauses, err := aiAnalyzer.identifyRootCauses(context.Background(), analysis, createTestLogEntries(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only return causes above confidence threshold (0.7)
	if len(rootCauses) != 1 {
		t.Errorf("Expected 1 root cause above confidence threshold, got %d", len(rootCauses))
	}

	if len(rootCauses) > 0 {
		if rootCauses[0].Title != "Database Connection Pool Exhaustion" {
			t.Errorf("Expected specific title, got %s", rootCauses[0].Title)
		}

		if rootCauses[0].Confidence != 0.8 {
			t.Errorf("Expected confidence 0.8, got %f", rootCauses[0].Confidence)
		}
	}
}

func TestGenerateRecommendations(t *testing.T) {
	jsonResponse := `[
		{
			"title": "Increase Database Connection Pool Size",
			"description": "Scale up the connection pool to handle increased load",
			"priority": "high",
			"category": "configuration",
			"action_items": ["Update config", "Restart service"],
			"effort": "low"
		}
	]`

	mockProvider := &MockProvider{
		name: "test-provider",
		completionFunc: func(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
			return &ai.CompletionResponse{
				Content: jsonResponse,
			}, nil
		},
	}

	options := &AIAnalyzerOptions{
		Provider: mockProvider,
	}

	aiAnalyzer := NewAIAnalyzer(NewEngine(), options)

	analysis := &common.Analysis{
		ErrorCount: 2,
		WarnCount:  1,
	}

	recommendations, err := aiAnalyzer.generateRecommendations(context.Background(), analysis, createTestLogEntries(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(recommendations) != 1 {
		t.Errorf("Expected 1 recommendation, got %d", len(recommendations))
	}

	if len(recommendations) > 0 {
		rec := recommendations[0]
		if rec.Title != "Increase Database Connection Pool Size" {
			t.Errorf("Expected specific title, got %s", rec.Title)
		}

		if rec.Priority != PriorityHigh {
			t.Errorf("Expected high priority, got %s", rec.Priority)
		}

		if len(rec.ActionItems) != 2 {
			t.Errorf("Expected 2 action items, got %d", len(rec.ActionItems))
		}
	}
}

func TestDocumentContextIntegration(t *testing.T) {
	mockProvider := &MockProvider{
		name: "test-provider",
		completionFunc: func(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
			// Check if the prompt contains context section
			if !contains(req.Prompt, "Context: Relevant Documentation") {
				t.Errorf("Expected prompt to contain document context, but it didn't")
			}

			return &ai.CompletionResponse{
				Content:      "Analysis with document context",
				FinishReason: "stop",
				Usage: &ai.TokenUsage{
					TotalTokens: 20,
				},
			}, nil
		},
	}

	options := &AIAnalyzerOptions{
		Provider:              mockProvider,
		EnableDocumentContext: true,
		MaxContextTokens:      1000,
	}

	aiAnalyzer := NewAIAnalyzer(NewEngine(), options)

	// Create a mock document context
	docContext := &DocumentContext{
		CorrelatedDocuments: []ContextDocument{
			{
				Title:           "Database Troubleshooting Guide",
				Path:            "/docs/database.md",
				MatchedKeywords: []string{"connection", "timeout"},
				Score:           0.85,
				Excerpt:         "When facing database connection issues, check the connection pool settings...",
				RelevantSection: "Connection Pool Configuration",
			},
		},
		TotalDocuments:   1,
		TokensUsed:       50,
		TruncatedContext: false,
	}

	analysis := &common.Analysis{
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		TotalEntries: 100,
		ErrorCount:   5,
		WarnCount:    10,
	}

	// Test that buildContextSection works correctly
	contextSection := aiAnalyzer.buildContextSection(docContext)
	if contextSection == "" {
		t.Error("Expected non-empty context section")
	}

	if !contains(contextSection, "Database Troubleshooting Guide") {
		t.Error("Expected context section to contain document title")
	}

	if !contains(contextSection, "Score: 0.85") {
		t.Error("Expected context section to contain relevance score")
	}

	// Test that citations are extracted correctly
	citations := aiAnalyzer.extractCitations(docContext)
	if len(citations) != 1 {
		t.Errorf("Expected 1 citation, got %d", len(citations))
	}

	if len(citations) > 0 {
		citation := citations[0]
		if citation.DocumentTitle != "Database Troubleshooting Guide" {
			t.Errorf("Expected citation title to match, got %s", citation.DocumentTitle)
		}
		if citation.Relevance != 0.85 {
			t.Errorf("Expected citation relevance to be 0.85, got %f", citation.Relevance)
		}
	}

	// Test that summary generation includes context
	summary, err := aiAnalyzer.generateSummary(context.Background(), analysis, createTestLogEntries(), docContext)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summary != "Analysis with document context" {
		t.Errorf("Expected summary to include document context, got %s", summary)
	}
}

func TestBuildDocumentContext(t *testing.T) {
	aiAnalyzer := NewAIAnalyzer(NewEngine(), &AIAnalyzerOptions{
		MaxContextTokens: 1000,
	})

	// Test with nil correlation result
	docContext := aiAnalyzer.buildDocumentContext(nil)
	if docContext != nil {
		t.Error("Expected nil document context for nil correlation result")
	}

	// Test with empty correlations
	emptyResult := &correlationpkg.CorrelationResult{
		TotalPatterns:      0,
		CorrelatedPatterns: 0,
		Correlations:       []*correlationpkg.PatternCorrelation{},
	}

	docContext = aiAnalyzer.buildDocumentContext(emptyResult)
	if docContext != nil {
		t.Error("Expected nil document context for empty correlations")
	}
}
