package main

// This is an example demonstrating the AI document context integration.
// It shows the structure for connecting AI analysis with document correlation.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yildizm/LogSum/internal/ai/providers/openai"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/go-logparser"
)

// CorrelatorAdapter adapts the correlation.Correlator to work with AI analyzer
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
	return fmt.Errorf("invalid document store type")
}

func main() {
	fmt.Println("🚀 AI Document Context Integration Demo")
	fmt.Println("=====================================")

	// Create AI provider (using mock for demo)
	provider := &openai.Provider{} // Would need proper configuration

	// Create AI analyzer
	aiOptions := &analyzer.AIAnalyzerOptions{
		Provider:                provider,
		EnableDocumentContext:   true,
		MaxContextTokens:        1000,
		EnableErrorAnalysis:     true,
		EnableRootCauseAnalysis: true,
		EnableRecommendations:   true,
	}

	baseAnalyzer := analyzer.NewEngine()
	aiAnalyzer := analyzer.NewAIAnalyzer(baseAnalyzer, aiOptions)

	// Create document store and correlator
	memStore := docstore.NewMemoryStore()
	correlator := correlation.NewCorrelator()

	// Set up the correlation adapter
	adapter := &CorrelatorAdapter{correlator: correlator}
	aiAnalyzer.SetCorrelator(adapter)
	err := aiAnalyzer.SetDocumentStore(memStore)
	if err != nil {
		log.Printf("Failed to set document store: %v", err)
	}

	// Add some sample documents
	doc1 := &docstore.Document{
		ID:      "doc1",
		Path:    "/docs/database_troubleshooting.md",
		Title:   "Database Connection Issues",
		Content: "When experiencing database connection timeouts, check the connection pool settings...",
		Metadata: &docstore.Metadata{
			Tags: []string{"database", "troubleshooting"},
		},
	}

	doc2 := &docstore.Document{
		ID:      "doc2",
		Path:    "/docs/performance_guide.md",
		Title:   "Performance Optimization Guide",
		Content: "High memory usage can be caused by memory leaks in application code...",
		Metadata: &docstore.Metadata{
			Tags: []string{"performance", "memory"},
		},
	}

	err = memStore.Add(doc1)
	if err != nil {
		log.Printf("Failed to store document 1: %v", err)
	}

	err = memStore.Add(doc2)
	if err != nil {
		log.Printf("Failed to store document 2: %v", err)
	}

	// Create sample log entries
	entries := []*common.LogEntry{
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-1 * time.Hour),
				Level:     "ERROR",
				Message:   "Database connection timeout after 30 seconds",
			},
			LogLevel: common.LevelError,
			Source:   "app.go:123",
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
			Source:   "db.go:789",
		},
	}

	fmt.Printf("\n📊 Analyzing %d log entries...\n", len(entries))

	// Note: This demo shows the structure but won't run fully without proper AI provider setup
	// The integration demonstrates how to connect AI analysis with document correlation

	// In a real implementation, this would:
	// 1. Analyze the log entries for patterns
	// 2. Correlate error patterns with documentation
	// 3. Include relevant documentation excerpts in AI prompts
	// 4. Generate context-aware analysis with source citations

	fmt.Println("\n✅ Integration structure demonstrated!")
	fmt.Println("\nFeatures added:")
	fmt.Println("- Document context integration interface")
	fmt.Println("- Source citations in AI responses")
	fmt.Println("- Context-aware prompt enhancement")
	fmt.Println("- Configurable context token limits")
	fmt.Println("\nTo complete the integration:")
	fmt.Println("1. Resolve import cycle between analyzer and correlation packages")
	fmt.Println("2. Implement correlation result processing in AI analyzer")
	fmt.Println("3. Enable document context in production usage")
}
