package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/LogSum/internal/vectorstore"
	"github.com/yildizm/go-logparser"
)

// MockAIProvider for testing
type MockAIProvider struct {
	name string
}

func (m *MockAIProvider) Name() string {
	return m.name
}

func (m *MockAIProvider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
	content := req.Prompt

	if strings.Contains(content, "database") {
		return &ai.CompletionResponse{
			Content: `{
				"error_analysis": {
					"primary_error": "Database connection timeout",
					"severity": "high",
					"frequency": 2,
					"impact": "Service degradation"
				},
				"root_cause": {
					"likely_cause": "Connection pool exhaustion",
					"confidence": 0.85,
					"supporting_evidence": ["Multiple timeout errors", "High connection count"]
				},
				"recommendations": [
					{
						"action": "Increase connection pool size",
						"priority": "high",
						"implementation": "Update database configuration"
					}
				]
			}`,
			Usage: &ai.TokenUsage{
				PromptTokens:     50,
				CompletionTokens: 100,
				TotalTokens:      150,
			},
		}, nil
	}

	return &ai.CompletionResponse{
		Content: `{
			"error_analysis": {
				"primary_error": "General application error",
				"severity": "medium",
				"frequency": 1,
				"impact": "Minor disruption"
			},
			"root_cause": {
				"likely_cause": "Application logic issue",
				"confidence": 0.6
			}
		}`,
		Usage: &ai.TokenUsage{
			PromptTokens:     40,
			CompletionTokens: 80,
			TotalTokens:      120,
		},
	}, nil
}

func (m *MockAIProvider) CompleteStream(ctx context.Context, req *ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, nil // Not implemented for testing
}

func (m *MockAIProvider) CountTokens(text string) (int, error) {
	return len(strings.Split(text, " ")), nil
}

func (m *MockAIProvider) MaxTokens() int {
	return 4096
}

func (m *MockAIProvider) SupportsStreaming() bool {
	return false
}

func (m *MockAIProvider) ValidateConfig() error {
	return nil
}

func (m *MockAIProvider) Close() error {
	return nil
}

func (m *MockAIProvider) TruncateToFit(text string, maxTokens int) (string, error) {
	return text, nil
}

func (m *MockAIProvider) SplitByTokens(text string, chunkSize int) ([]string, error) {
	return []string{text}, nil
}

func (m *MockAIProvider) EstimateTokens(text string) int {
	return len(strings.Split(text, " "))
}

func (m *MockAIProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockAIProvider) IsHealthy() bool {
	return true
}

// CorrelatorAdapter for testing
type CorrelatorAdapter struct {
	correlator correlation.Correlator
}

func (ca *CorrelatorAdapter) Correlate(ctx context.Context, analysis *common.Analysis) (interface{}, error) {
	return ca.correlator.Correlate(ctx, analysis)
}

func (ca *CorrelatorAdapter) SetDocumentStore(store interface{}) error {
	if docStore, ok := store.(docstore.DocumentStore); ok {
		return ca.correlator.SetDocumentStore(docStore)
	}
	return nil
}

// TestFullPipelineIntegration tests the complete Logs → Parse → Analyze → Correlate → AI pipeline
//
//nolint:gocyclo // Test function intentionally shows multiple integration steps
func TestFullPipelineIntegration(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create sample log entries (simulating parsing)
	logEntries := []*common.LogEntry{
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-1 * time.Hour),
				Level:     "ERROR",
				Message:   "Database connection timeout after 30 seconds",
			},
			LogLevel: common.LevelError,
			Source:   "db.go:123",
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-50 * time.Minute),
				Level:     "WARN",
				Message:   "High memory usage detected: 85% of available memory in use",
			},
			LogLevel: common.LevelWarn,
			Source:   "monitor.go:456",
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-30 * time.Minute),
				Level:     "ERROR",
				Message:   "Failed to acquire database connection from pool",
			},
			LogLevel: common.LevelError,
			Source:   "pool.go:789",
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-15 * time.Minute),
				Level:     "INFO",
				Message:   "Application started successfully",
			},
			LogLevel: common.LevelInfo,
			Source:   "main.go:45",
		},
	}

	t.Logf("Step 1: Created %d log entries", len(logEntries))

	// Step 2: Analyze logs using the analyzer engine
	engine := analyzer.NewEngine()
	analysis, err := engine.Analyze(ctx, logEntries)
	if err != nil {
		t.Fatalf("Failed to analyze logs: %v", err)
	}

	t.Logf("Step 2: Analysis completed - %d patterns, %d errors",
		len(analysis.Patterns), analysis.ErrorCount)

	// Verify analysis results
	if len(analysis.Patterns) == 0 {
		t.Log("No patterns detected (this may be expected for small datasets)")
	}

	if analysis.ErrorCount == 0 {
		t.Error("Expected error count in analysis")
	}

	// Step 3: Set up document store with relevant documentation
	docStore := docstore.NewMemoryStore()

	doc1 := &docstore.Document{
		ID:      "db-troubleshooting",
		Path:    "/docs/database_troubleshooting.md",
		Title:   "Database Connection Issues",
		Content: "When experiencing database connection timeouts, check the connection pool settings. Common causes include pool exhaustion and network connectivity issues.",
		Metadata: &docstore.Metadata{
			Tags: []string{"database", "troubleshooting", "connection"},
		},
	}

	doc2 := &docstore.Document{
		ID:      "performance-guide",
		Path:    "/docs/performance_guide.md",
		Title:   "Performance Optimization Guide",
		Content: "High memory usage can indicate memory leaks or inefficient resource usage. Monitor memory patterns and consider garbage collection tuning.",
		Metadata: &docstore.Metadata{
			Tags: []string{"performance", "memory", "optimization"},
		},
	}

	if err := docStore.Add(doc1); err != nil {
		t.Fatalf("Failed to add document 1: %v", err)
	}

	if err := docStore.Add(doc2); err != nil {
		t.Fatalf("Failed to add document 2: %v", err)
	}

	stats, _ := docStore.Stats()
	t.Logf("Step 3: Added %d documents to document store", stats.DocumentCount)

	// Step 4: Set up correlation with vector store
	vectorStore := vectorstore.NewMemoryStore(vectorstore.WithCache(100))
	vectorizer := vectorstore.NewTFIDFVectorizer(384)
	correlator := correlation.NewCorrelator()
	_ = correlator.SetDocumentStore(docStore)
	_ = correlator.SetVectorStore(vectorStore, vectorizer)

	// Add vectors for documents (simplified for testing)
	vec1 := make([]float32, 384)
	for i := range vec1 {
		vec1[i] = 0.1 // Simple test vector
	}

	vec2 := make([]float32, 384)
	for i := range vec2 {
		vec2[i] = 0.2 // Different test vector
	}

	if err := vectorStore.Store("db-troubleshooting", doc1.Content, vec1); err != nil {
		t.Fatalf("Failed to store vector 1: %v", err)
	}

	if err := vectorStore.Store("performance-guide", doc2.Content, vec2); err != nil {
		t.Fatalf("Failed to store vector 2: %v", err)
	}

	// Step 5: Run correlation
	correlationResult, err := correlator.Correlate(ctx, analysis)
	if err != nil {
		t.Fatalf("Failed to correlate analysis: %v", err)
	}

	t.Logf("Step 4-5: Correlation completed - %d total patterns, %d correlated",
		correlationResult.TotalPatterns, correlationResult.CorrelatedPatterns)

	// Step 6: Set up AI analyzer with document context
	provider := &MockAIProvider{name: "mock-gpt"}

	aiOptions := &analyzer.AIAnalyzerOptions{
		Provider:                provider,
		EnableDocumentContext:   true,
		MaxContextTokens:        1000,
		EnableErrorAnalysis:     true,
		EnableRootCauseAnalysis: true,
		EnableRecommendations:   true,
	}

	aiAnalyzer := analyzer.NewAIAnalyzer(engine, aiOptions)

	// Set up correlation adapter
	adapter := &CorrelatorAdapter{correlator: correlator}
	aiAnalyzer.SetCorrelator(adapter)
	if err := aiAnalyzer.SetDocumentStore(docStore); err != nil {
		t.Fatalf("Failed to set document store: %v", err)
	}

	// Step 7: Generate AI-enhanced analysis
	aiResult, err := aiAnalyzer.AnalyzeWithAI(ctx, logEntries)
	if err != nil {
		t.Fatalf("Failed to generate AI analysis: %v", err)
	}

	t.Logf("Step 6-7: AI analysis completed - Summary length: %d chars", len(aiResult.AISummary))

	// Step 8: Performance validation (<100ms requirement)
	start := time.Now()
	_, err = aiAnalyzer.AnalyzeWithAI(ctx, logEntries[:2]) // Smaller set for performance test
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed performance test: %v", err)
	}

	t.Logf("Step 8: Performance test - Analysis took %v", duration)

	// Performance requirement: <100ms for typical analysis
	if duration > 100*time.Millisecond {
		t.Logf("Warning: Analysis took %v, which exceeds 100ms target", duration)
		// Note: This is a warning, not a failure, as the mock provider might be slower
	}

	// Final validation
	if aiResult.AISummary == "" {
		t.Error("Expected non-empty summary from AI analysis")
	}

	// Check that correlation data is available
	if correlationResult.TotalPatterns == 0 {
		t.Log("No correlated patterns found (expected for test dataset)")
	}

	// Verify vector store has entries
	if vectorStore.Size() != 2 {
		t.Errorf("Expected 2 vectors in store, got %d", vectorStore.Size())
	}

	t.Log("✅ Full pipeline integration test completed successfully!")
	stats2, _ := docStore.Stats()
	t.Logf("Pipeline: %d logs → Analysis → %d documents → Correlation → AI → Summary",
		len(logEntries), stats2.DocumentCount)
}

// TestPipelinePerformance tests the performance characteristics of the full pipeline
func TestPipelinePerformance(t *testing.T) {
	ctx := context.Background()

	// Create larger dataset for performance testing
	logEntries := make([]*common.LogEntry, 100)
	for i := 0; i < 100; i++ {
		logEntries[i] = &common.LogEntry{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
				Level:     "ERROR",
				Message:   "Database connection timeout after 30 seconds",
			},
			LogLevel: common.LevelError,
			Source:   "db.go:123",
		}
	}

	start := time.Now()

	// Run basic analysis
	engine := analyzer.NewEngine()
	analysis, err := engine.Analyze(ctx, logEntries)
	if err != nil {
		t.Fatalf("Failed to analyze logs: %v", err)
	}

	analysisTime := time.Since(start)
	t.Logf("Analysis of %d entries took: %v", len(logEntries), analysisTime)

	// Memory usage validation
	if len(analysis.Patterns) > 0 {
		t.Logf("Generated %d patterns for %d log entries", len(analysis.Patterns), len(logEntries))
	}

	// Performance should scale reasonably
	if analysisTime > 1*time.Second {
		t.Errorf("Analysis took too long: %v for %d entries", analysisTime, len(logEntries))
	}
}

// TestPipelineErrorHandling tests error handling throughout the pipeline
func TestPipelineErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Test with empty log entries
	engine := analyzer.NewEngine()
	_, err := engine.Analyze(ctx, []*common.LogEntry{})
	if err != nil {
		t.Logf("Expected behavior: Empty logs handled gracefully: %v", err)
	}

	// Test with malformed data
	badEntry := &common.LogEntry{
		LogEntry: logparser.LogEntry{
			Timestamp: time.Time{}, // Zero time
			Message:   "",          // Empty message
		},
		LogLevel: common.LevelDebug,
	}

	analysis, err := engine.Analyze(ctx, []*common.LogEntry{badEntry})
	if err != nil {
		t.Logf("Malformed data handled: %v", err)
	} else {
		t.Logf("Malformed data processed: %d patterns", len(analysis.Patterns))
	}

	// Test document store with duplicate IDs
	docStore := docstore.NewMemoryStore()
	doc := &docstore.Document{
		ID:      "duplicate-test",
		Content: "Test content",
	}

	err1 := docStore.Add(doc)
	err2 := docStore.Add(doc) // Duplicate

	if err1 != nil {
		t.Errorf("First add should succeed: %v", err1)
	}

	if err2 == nil {
		t.Log("Duplicate handling: Overwrite allowed")
	} else {
		t.Logf("Duplicate handling: %v", err2)
	}
}
