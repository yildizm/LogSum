package correlation

import (
	"context"
	"testing"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/LogSum/internal/vectorstore"
	"github.com/yildizm/go-logparser"
)

// MockDocumentStore for testing
type MockDocumentStore struct {
	documents map[string]*docstore.Document
}

func NewMockDocumentStore() *MockDocumentStore {
	return &MockDocumentStore{
		documents: make(map[string]*docstore.Document),
	}
}

func (m *MockDocumentStore) AddDocument(doc *docstore.Document) {
	m.documents[doc.ID] = doc
}

// Implement all DocumentStore interface methods
func (m *MockDocumentStore) Add(doc *docstore.Document) error {
	m.documents[doc.ID] = doc
	return nil
}

func (m *MockDocumentStore) AddBatch(docs []*docstore.Document) error {
	for _, doc := range docs {
		m.documents[doc.ID] = doc
	}
	return nil
}

func (m *MockDocumentStore) Get(id string) (*docstore.Document, error) {
	if doc, exists := m.documents[id]; exists {
		return doc, nil
	}
	return nil, nil
}

func (m *MockDocumentStore) Update(doc *docstore.Document) error {
	m.documents[doc.ID] = doc
	return nil
}

func (m *MockDocumentStore) Delete(id string) error {
	delete(m.documents, id)
	return nil
}

func (m *MockDocumentStore) List(filter *docstore.FilterOptions) ([]*docstore.Document, error) {
	docs := make([]*docstore.Document, 0, len(m.documents))
	for _, doc := range m.documents {
		docs = append(docs, doc)
	}
	return docs, nil
}

func (m *MockDocumentStore) SearchSections(query *docstore.SearchQuery) ([]*docstore.Section, error) {
	return nil, nil
}

func (m *MockDocumentStore) SearchSimple(text string) ([]*docstore.SearchResult, error) {
	query := &docstore.SearchQuery{Text: text}
	return m.Search(query)
}

func (m *MockDocumentStore) Reindex() error {
	return nil
}

func (m *MockDocumentStore) IndexDocument(doc *docstore.Document) error {
	return nil
}

func (m *MockDocumentStore) RemoveFromIndex(docID string) error {
	return nil
}

func (m *MockDocumentStore) Clear() error {
	m.documents = make(map[string]*docstore.Document)
	return nil
}

func (m *MockDocumentStore) Stats() (*docstore.StoreStats, error) {
	return &docstore.StoreStats{}, nil
}

func (m *MockDocumentStore) Close() error {
	return nil
}

func (m *MockDocumentStore) AddChangeListener(listener docstore.ChangeListener) {
	// No-op for mock
}

func (m *MockDocumentStore) ProcessChanges(changes []*docstore.DocumentChange) error {
	return nil
}

func (m *MockDocumentStore) Search(query *docstore.SearchQuery) ([]*docstore.SearchResult, error) {
	results := make([]*docstore.SearchResult, 0, len(m.documents))

	for _, doc := range m.documents {
		// Simple text search
		if query.Text != "" && !containsIgnoreCase(doc.Content, query.Text) {
			continue
		}
		score := 0.8 // Mock score
		if query.Text != "" {
			score = 0.9
		}

		result := &docstore.SearchResult{
			Document: doc,
			Score:    score,
		}

		if query.Highlight {
			result.Highlighted = doc.Content // Simplified highlighting
		}

		results = append(results, result)
	}

	return results, nil
}

func containsIgnoreCase(text, substr string) bool {
	return len(text) >= len(substr) &&
		(substr == "" || findSubstring(text, substr) != "")
}

func findSubstring(text, substr string) string {
	// Simplified case-insensitive search
	for i := 0; i <= len(text)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(text[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return text[i : i+len(substr)]
		}
	}
	return ""
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

// TestHybridSearchIntegration tests the complete hybrid search pipeline
func TestHybridSearchIntegration(t *testing.T) {
	// Setup
	correlator := NewCorrelator()

	// Create mock document store
	docStore := NewMockDocumentStore()
	docStore.AddDocument(&docstore.Document{
		ID:      "doc1",
		Content: "Database connection timeout error occurred while connecting to PostgreSQL",
		Path:    "/docs/database.md",
	})
	docStore.AddDocument(&docstore.Document{
		ID:      "doc2",
		Content: "Network connectivity issues can cause timeouts in database connections",
		Path:    "/docs/network.md",
	})
	docStore.AddDocument(&docstore.Document{
		ID:      "doc3",
		Content: "Authentication failure when accessing database with invalid credentials",
		Path:    "/docs/auth.md",
	})

	// Setup document store
	err := correlator.SetDocumentStore(docStore)
	if err != nil {
		t.Fatalf("Failed to set document store: %v", err)
	}

	// Create vector store and vectorizer
	vectorizer := vectorstore.NewTFIDFVectorizer(100)
	vectorStore := vectorstore.NewMemoryStore()

	// Setup vector store
	err = correlator.SetVectorStore(vectorStore, vectorizer)
	if err != nil {
		t.Fatalf("Failed to set vector store: %v", err)
	}

	// Index documents
	ctx := context.Background()
	err = correlator.IndexDocuments(ctx)
	if err != nil {
		t.Fatalf("Failed to index documents: %v", err)
	}

	// Create test analysis
	analysis := &common.Analysis{
		Patterns: []common.PatternMatch{
			{
				Pattern: &common.Pattern{
					Name:        "DatabaseConnectionError",
					Description: "Database connection timeout",
					Regex:       "connection.*timeout",
				},
				Matches: []*common.LogEntry{
					{
						LogEntry: logparser.LogEntry{
							Message: "connection timeout while accessing database",
						},
					},
				},
				Count: 1,
			},
		},
	}

	// Test correlation
	result, err := correlator.Correlate(ctx, analysis)
	if err != nil {
		t.Fatalf("Correlation failed: %v", err)
	}

	// Verify results
	if result == nil {
		t.Fatal("Expected correlation result, got nil")
	}

	if result.TotalPatterns != 1 {
		t.Errorf("Expected 1 total pattern, got %d", result.TotalPatterns)
	}

	if len(result.Correlations) == 0 {
		t.Fatal("Expected at least one correlation")
	}

	correlation := result.Correlations[0]
	if len(correlation.DocumentMatches) == 0 {
		t.Fatal("Expected at least one document match")
	}

	// Verify hybrid scoring is working
	foundHybridMatch := false
	for _, match := range correlation.DocumentMatches {
		if match.SearchMethod == "hybrid" || match.SearchMethod == "keyword" || match.SearchMethod == "vector" {
			foundHybridMatch = true
			break
		}
	}

	if !foundHybridMatch {
		t.Error("Expected to find matches with search method populated")
	}
}

// TestHybridSearchConfig tests configuration validation
func TestHybridSearchConfig(t *testing.T) {
	correlator := NewCorrelator()

	// Test invalid configuration (weights don't sum to ~1.0)
	invalidConfig := &HybridSearchConfig{
		KeywordWeight: 0.3,
		VectorWeight:  0.3, // Sum = 0.6, too low
		MaxResults:    5,
		VectorTopK:    10,
		EnableVector:  true,
	}

	err := correlator.SetHybridSearchConfig(invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid weight configuration")
	}

	// Test valid configuration
	validConfig := &HybridSearchConfig{
		KeywordWeight: 0.7,
		VectorWeight:  0.3,
		MaxResults:    10,
		VectorTopK:    15,
		EnableVector:  true,
	}

	err = correlator.SetHybridSearchConfig(validConfig)
	if err != nil {
		t.Errorf("Expected valid configuration to succeed, got: %v", err)
	}
}

// TestKeywordOnlySearch tests that correlation works without vector store
func TestKeywordOnlySearch(t *testing.T) {
	correlator := NewCorrelator()

	// Setup only document store (no vector store)
	docStore := NewMockDocumentStore()
	docStore.AddDocument(&docstore.Document{
		ID:      "doc1",
		Content: "Database connection error troubleshooting guide",
		Path:    "/docs/database.md",
	})

	err := correlator.SetDocumentStore(docStore)
	if err != nil {
		t.Fatalf("Failed to set document store: %v", err)
	}

	// Create test analysis
	analysis := &common.Analysis{
		Patterns: []common.PatternMatch{
			{
				Pattern: &common.Pattern{
					Name:        "DatabaseError",
					Description: "Database connection issue",
					Regex:       "database.*error",
				},
				Matches: []*common.LogEntry{
					{
						LogEntry: logparser.LogEntry{
							Message: "database connection failed",
						},
					},
				},
				Count: 1,
			},
		},
	}

	// Test correlation without vector store
	ctx := context.Background()
	result, err := correlator.Correlate(ctx, analysis)
	if err != nil {
		t.Fatalf("Correlation failed: %v", err)
	}

	// Should still work with keyword search only
	if result == nil || len(result.Correlations) == 0 {
		t.Fatal("Expected correlation results even without vector store")
	}

	// Verify search method is keyword-only
	correlation := result.Correlations[0]
	for _, match := range correlation.DocumentMatches {
		if match.SearchMethod != "keyword" {
			t.Errorf("Expected keyword search method, got %s", match.SearchMethod)
		}
		if match.VectorScore != 0.0 {
			t.Errorf("Expected zero vector score for keyword-only search, got %f", match.VectorScore)
		}
	}
}

// TestVectorSearchFiltering tests minimum vector score filtering
func TestVectorSearchFiltering(t *testing.T) {
	correlator := NewCorrelator()

	// Setup with high minimum vector score
	config := &HybridSearchConfig{
		KeywordWeight:  0.5,
		VectorWeight:   0.5,
		MaxResults:     5,
		VectorTopK:     10,
		MinVectorScore: 0.9, // Very high threshold
		EnableVector:   true,
	}

	err := correlator.SetHybridSearchConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Setup stores
	docStore := NewMockDocumentStore()
	docStore.AddDocument(&docstore.Document{
		ID:      "doc1",
		Content: "Some unrelated content about cooking recipes",
		Path:    "/docs/cooking.md",
	})

	vectorizer := vectorstore.NewTFIDFVectorizer(50)
	vectorStore := vectorstore.NewMemoryStore()

	if err := correlator.SetDocumentStore(docStore); err != nil {
		t.Fatalf("Failed to set document store: %v", err)
	}
	if err := correlator.SetVectorStore(vectorStore, vectorizer); err != nil {
		t.Fatalf("Failed to set vector store: %v", err)
	}

	ctx := context.Background()
	if err := correlator.IndexDocuments(ctx); err != nil {
		t.Fatalf("Failed to index documents: %v", err)
	}

	// Test with analysis that should have low vector similarity
	analysis := &common.Analysis{
		Patterns: []common.PatternMatch{
			{
				Pattern: &common.Pattern{
					Name:        "DatabaseError",
					Description: "Database connection timeout error",
					Regex:       "database.*error",
				},
				Matches: []*common.LogEntry{
					{
						LogEntry: logparser.LogEntry{
							Message: "database connection timeout error occurred",
						},
					},
				},
				Count: 1,
			},
		},
	}

	result, err := correlator.Correlate(ctx, analysis)
	if err != nil {
		t.Fatalf("Correlation failed: %v", err)
	}

	// With high minimum vector score, should fallback to keyword results only
	if result != nil && len(result.Correlations) > 0 {
		correlation := result.Correlations[0]
		for _, match := range correlation.DocumentMatches {
			// Should either be keyword-only or have high vector scores
			if match.SearchMethod == "vector" && match.VectorScore < 0.9 {
				t.Errorf("Expected vector scores above threshold, got %f", match.VectorScore)
			}
		}
	}
}

// TestDefaultConfiguration tests the default hybrid search configuration
func TestDefaultConfiguration(t *testing.T) {
	config := DefaultHybridSearchConfig()

	if config.KeywordWeight+config.VectorWeight < 0.9 || config.KeywordWeight+config.VectorWeight > 1.1 {
		t.Errorf("Default weights should sum to ~1.0, got %.2f",
			config.KeywordWeight+config.VectorWeight)
	}

	if config.MaxResults <= 0 {
		t.Errorf("Default MaxResults should be positive, got %d", config.MaxResults)
	}

	if config.VectorTopK <= 0 {
		t.Errorf("Default VectorTopK should be positive, got %d", config.VectorTopK)
	}

	if !config.EnableVector {
		t.Error("Default configuration should enable vector search")
	}
}

// TestEmptyAnalysis tests handling of empty analysis
func TestEmptyAnalysis(t *testing.T) {
	correlator := NewCorrelator()
	docStore := NewMockDocumentStore()
	if err := correlator.SetDocumentStore(docStore); err != nil {
		t.Fatalf("Failed to set document store: %v", err)
	}

	ctx := context.Background()

	// Test with nil analysis
	_, err := correlator.Correlate(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil analysis")
	}

	// Test with empty patterns
	emptyAnalysis := &common.Analysis{
		Patterns: []common.PatternMatch{},
	}

	result, err := correlator.Correlate(ctx, emptyAnalysis)
	if err != nil {
		t.Errorf("Expected to handle empty analysis, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected result even for empty analysis")
		return
	}

	if result.TotalPatterns != 0 {
		t.Errorf("Expected 0 total patterns for empty analysis, got %d", result.TotalPatterns)
	}
}
