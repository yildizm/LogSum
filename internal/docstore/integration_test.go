package docstore

import (
	"path/filepath"
	"testing"
)

func TestDocStoreIntegration(t *testing.T) {
	// Create store components
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Test with our actual test data
	testDataDir := "../../testdata/docs"

	// Check if test data exists
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		t.Skipf("Cannot resolve test data path: %v", err)
	}

	// Scan and add directory
	err = store.ScanAndAddDirectory(absPath, []string{"*.md"})
	if err != nil {
		t.Skipf("Test data directory not available: %v", err)
	}

	// Get store statistics
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get store stats: %v", err)
	}

	// Verify we indexed some documents
	if stats.DocumentCount == 0 {
		t.Error("Expected to index some documents")
	}

	t.Logf("Indexed %d documents with %d total sections",
		stats.DocumentCount, stats.SectionCount)
	t.Logf("Total size: %d bytes, average doc size: %.1f bytes",
		stats.TotalSize, stats.AverageDocSize)

	// Test search functionality
	query := &SearchQuery{
		Text:      "API",
		Highlight: true,
		Limit:     10,
	}

	results, err := store.Search(query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	t.Logf("Found %d results for 'API' search", len(results))

	// Test another search
	query2 := &SearchQuery{
		Text:  "setup installation",
		Limit: 5,
	}

	results2, err := store.Search(query2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	t.Logf("Found %d results for 'setup installation' search", len(results2))

	// Test list functionality
	allDocs, err := store.List(&FilterOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(allDocs) != int(stats.DocumentCount) {
		t.Errorf("List returned %d docs, stats show %d", len(allDocs), stats.DocumentCount)
	}

	// Test filtering
	for _, doc := range allDocs {
		if doc.Metadata == nil || len(doc.Metadata.Tags) == 0 {
			continue
		}
		// Test tag filtering
		tagFilter := &FilterOptions{Tags: doc.Metadata.Tags[:1]}
		tagDocs, err := store.List(tagFilter)
		if err != nil {
			t.Errorf("Tag filtering failed: %v", err)
		}
		if len(tagDocs) == 0 {
			t.Error("Tag filter should return at least the source document")
		}
		break
	}
}

func TestDocStorePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create store components
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Create many test documents
	docs := make([]*Document, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = createTestDocument(
			string(rune(i)),
			"Performance Test Document",
			"This is a performance test document with various terms for searching and indexing performance evaluation",
		)
	}

	// Measure indexing performance
	err := memStore.AddBatch(docs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	// Verify all documents were indexed
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.DocumentCount != 1000 {
		t.Errorf("Expected 1000 documents, got %d", stats.DocumentCount)
	}

	// Test search performance
	query := &SearchQuery{
		Text:  "performance test",
		Limit: 50,
	}

	results, err := store.Search(query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	t.Logf("Performance test: indexed 1000 docs, search returned %d results", len(results))
	t.Logf("Memory usage: %d bytes, indexed terms: %d", stats.MemoryUsage, stats.IndexedTerms)
}
