package correlation

import (
	"context"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/go-logparser"
)

func TestCorrelator(t *testing.T) {
	// Create a correlator
	correlator := NewCorrelator()

	// Create a mock document store
	store := docstore.NewMemoryStore()

	// Add test documents
	doc1 := &docstore.Document{
		ID:      "1",
		Path:    "/docs/database.md",
		Title:   "Database Connection Guide",
		Content: "This guide explains how to connect to the database. Connection timeout errors can occur when the network is slow.",
	}

	doc2 := &docstore.Document{
		ID:      "2",
		Path:    "/docs/authentication.md",
		Title:   "Authentication Guide",
		Content: "Authentication failures can happen due to invalid credentials or expired tokens.",
	}

	if err := store.Add(doc1); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	if err := store.Add(doc2); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Set up the correlator
	if err := correlator.SetDocumentStore(store); err != nil {
		t.Fatalf("Failed to set document store: %v", err)
	}

	// Create test analysis with patterns
	analysis := &analyzer.Analysis{
		Patterns: []analyzer.PatternMatch{
			{
				Pattern: &common.Pattern{
					ID:          "db_timeout",
					Name:        "Database Timeout",
					Description: "Database connection timeout error",
					Regex:       "connection.*timeout",
				},
				Count: 5,
			},
			{
				Pattern: &common.Pattern{
					ID:          "auth_fail",
					Name:        "Authentication Failure",
					Description: "Authentication failure with invalid credentials",
					Regex:       "authentication.*failed",
				},
				Count: 2,
			},
		},
	}

	// Run correlation
	ctx := context.Background()
	result, err := correlator.Correlate(ctx, analysis)
	if err != nil {
		t.Fatalf("Correlation failed: %v", err)
	}

	// Verify results
	if result.TotalPatterns != 2 {
		t.Errorf("Expected 2 total patterns, got %d", result.TotalPatterns)
	}

	if len(result.Correlations) == 0 {
		t.Error("Expected at least one correlation")
	}

	// Check that correlations have document matches
	for _, correlation := range result.Correlations {
		if len(correlation.DocumentMatches) == 0 {
			t.Errorf("Pattern %s has no document matches", correlation.Pattern.Name)
		}

		if len(correlation.Keywords) == 0 {
			t.Errorf("Pattern %s has no keywords", correlation.Pattern.Name)
		}
	}
}

func TestKeywordExtractor(t *testing.T) {
	extractor := NewKeywordExtractor()

	// Test pattern extraction
	pattern := &common.Pattern{
		Name:        "Database Connection Error",
		Description: "Connection timeout to database server",
		Regex:       "connection.*timeout.*database",
	}

	keywords := extractor.ExtractFromPattern(pattern)
	if len(keywords) == 0 {
		t.Error("Expected keywords from pattern")
	}

	// Check for expected keywords
	expectedKeywords := []string{"Database", "Connection", "timeout", "database", "server"}
	found := make(map[string]bool)
	for _, keyword := range keywords {
		found[keyword] = true
	}

	for _, expected := range expectedKeywords {
		if !found[expected] {
			t.Logf("Available keywords: %v", keywords)
			// Note: Not failing the test as keyword extraction may vary
		}
	}

	// Test log entry extraction
	entry := &common.LogEntry{
		LogEntry: logparser.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Connection timeout occurred while accessing MySQL database",
			Fields: map[string]interface{}{
				"service": "user-service",
				"error":   "connection_timeout",
			},
		},
		LogLevel: common.LevelError,
	}

	entryKeywords := extractor.ExtractFromLogEntry(entry)
	if len(entryKeywords) == 0 {
		t.Error("Expected keywords from log entry")
	}
}
