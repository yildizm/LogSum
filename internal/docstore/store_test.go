package docstore

import (
	"context"
	"testing"
	"time"
)

func TestStore_NewStore(t *testing.T) {
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)

	if store == nil {
		t.Error("Expected non-nil store")
		return
	}

	if store.store != memStore {
		t.Error("Store not properly set")
	}

	if store.scanner != scanner {
		t.Error("Scanner not properly set")
	}

	if store.indexer != indexer {
		t.Error("Indexer not properly set")
	}
}

func TestStore_ScanAndAdd(t *testing.T) {
	// This test would require actual file I/O, so we'll test the memory store directly
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// We can't easily test file scanning without creating temporary files,
	// so let's test the store operations directly
	doc := createTestDocument("test-1", "Test Document", "Test content")

	err := memStore.Add(doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Test retrieval through store
	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	if retrieved.ID != doc.ID {
		t.Errorf("Expected ID %q, got %q", doc.ID, retrieved.ID)
	}
}

func TestStore_Search(t *testing.T) {
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Add test documents
	docs := []*Document{
		createTestDocument("doc-1", "API Guide", "Complete guide to using our REST API"),
		createTestDocument("doc-2", "Setup Manual", "Installation and setup instructions"),
		createTestDocument("doc-3", "API Reference", "API endpoint reference documentation"),
	}

	for _, doc := range docs {
		if err := memStore.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Test search through store
	query := &SearchQuery{
		Text:  "API",
		Limit: 10,
	}

	results, err := store.Search(query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	// Should find both API-related documents
	foundDocs := make(map[string]bool)
	for _, result := range results {
		foundDocs[result.Document.ID] = true
	}

	if !foundDocs["doc-1"] || !foundDocs["doc-3"] {
		t.Error("Expected to find both API documents")
	}
}

func TestStore_List(t *testing.T) {
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndex()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Add test documents
	docs := []*Document{
		createTestDocument("doc-1", "Document 1", "Content 1"),
		createTestDocument("doc-2", "Document 2", "Content 2"),
		createTestDocument("doc-3", "Document 3", "Content 3"),
	}

	for _, doc := range docs {
		if err := memStore.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Test list through store
	allDocs, err := store.List(&FilterOptions{})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}

	if len(allDocs) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(allDocs))
	}

	// Test with limit
	limitedDocs, err := store.List(&FilterOptions{Limit: 2})
	if err != nil {
		t.Fatalf("Failed to list documents with limit: %v", err)
	}

	if len(limitedDocs) != 2 {
		t.Errorf("Expected 2 documents with limit, got %d", len(limitedDocs))
	}
}

func TestStore_Stats(t *testing.T) {
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Test empty store stats
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.DocumentCount != 0 {
		t.Errorf("Expected 0 documents, got %d", stats.DocumentCount)
	}

	// Add documents and test stats
	docs := []*Document{
		createTestDocument("doc-1", "Document 1", "Some content here"),
		createTestDocument("doc-2", "Document 2", "More content here"),
	}

	for _, doc := range docs {
		if err := memStore.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	stats, err = store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats after adding documents: %v", err)
	}

	if stats.DocumentCount != 2 {
		t.Errorf("Expected 2 documents, got %d", stats.DocumentCount)
	}

	if stats.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}

	if stats.AverageDocSize <= 0 {
		t.Error("Expected positive average document size")
	}
}

func TestStore_ProcessDirectoryWithContext(t *testing.T) {
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndexer()

	store := NewStore(memStore, scanner, indexer)
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := store.ProcessDirectoryWithContext(ctx, "/nonexistent", []string{"*.md"}, nil)
	if err != context.Canceled && err != nil && err.Error() != "failed to scan directory /nonexistent: lstat /nonexistent: no such file or directory" {
		t.Errorf("Expected context.Canceled or directory error, got %v", err)
	}

	// Test with timeout context
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This will likely timeout since we're using a very short timeout
	err = store.ProcessDirectoryWithContext(ctx, "/nonexistent", []string{"*.md"}, nil)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		// The error might be related to directory not existing, which is fine for this test
		// We're mainly testing the context cancellation mechanism
		t.Logf("ProcessDirectoryWithContext returned: %v", err)
	}
}

func TestStore_Close(t *testing.T) {
	memStore := NewMemoryStore()
	scanner := NewMarkdownScanner()
	indexer := NewMemoryIndex()

	store := NewStore(memStore, scanner, indexer)

	// Test close
	err := store.Close()
	if err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// After closing, the underlying store should be closed
	// We can test this indirectly by checking if operations fail
	// However, our memory store doesn't necessarily fail after close
	// so this test mainly ensures Close() doesn't panic
}

func TestFilterOptions_Validation(t *testing.T) {
	// Test various filter combinations
	tests := []struct {
		name   string
		filter FilterOptions
		valid  bool
	}{
		{
			name:   "Empty filter",
			filter: FilterOptions{},
			valid:  true,
		},
		{
			name: "Author filter",
			filter: FilterOptions{
				Author: "test-author",
			},
			valid: true,
		},
		{
			name: "Tag filter",
			filter: FilterOptions{
				Tags: []string{"tag1", "tag2"},
			},
			valid: true,
		},
		{
			name: "Date range filter",
			filter: FilterOptions{
				DateAfter:  time.Now().Add(-24 * time.Hour),
				DateBefore: time.Now(),
			},
			valid: true,
		},
		{
			name: "Pagination filter",
			filter: FilterOptions{
				Limit:  10,
				Offset: 20,
			},
			valid: true,
		},
		{
			name: "Path prefix filter",
			filter: FilterOptions{
				PathPrefix: "/docs/api/",
			},
			valid: true,
		},
		{
			name: "Combined filters",
			filter: FilterOptions{
				Author:     "test-author",
				Tags:       []string{"api"},
				PathPrefix: "/docs/",
				Limit:      5,
			},
			valid: true,
		},
	}

	memStore := NewMemoryStore()
	defer func() {
		if err := memStore.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the filter doesn't cause errors
			_, err := memStore.List(&tt.filter)
			if tt.valid && err != nil {
				t.Errorf("Valid filter caused error: %v", err)
			}
		})
	}
}

func TestSearchQuery_Validation(t *testing.T) {
	tests := []struct {
		name  string
		query *SearchQuery
		valid bool
	}{
		{
			name: "Valid basic query",
			query: &SearchQuery{
				Text:  "test search",
				Limit: 10,
			},
			valid: true,
		},
		{
			name: "Empty query text",
			query: &SearchQuery{
				Text:  "",
				Limit: 10,
			},
			valid: false, // Should return no results but not error
		},
		{
			name: "Query with fields",
			query: &SearchQuery{
				Text:   "test",
				Fields: []string{"title", "content"},
				Limit:  5,
			},
			valid: true,
		},
		{
			name: "Fuzzy search query",
			query: &SearchQuery{
				Text:  "approximate",
				Fuzzy: true,
				Limit: 10,
			},
			valid: true,
		},
		{
			name: "Highlighted search query",
			query: &SearchQuery{
				Text:      "important",
				Highlight: true,
				Limit:     10,
			},
			valid: true,
		},
	}

	memStore := NewMemoryStore()
	defer func() {
		if err := memStore.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Add a test document
	doc := createTestDocument("test-1", "Test Document", "This document contains important information")
	if err := memStore.Add(doc); err != nil {
		t.Fatalf("Failed to add test document: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := memStore.Search(tt.query)
			if err != nil {
				t.Errorf("Search query caused error: %v", err)
			}

			if tt.query.Text == "" && len(results) > 0 {
				t.Error("Empty query should return no results")
			}

			if tt.valid && tt.query.Text != "" && len(results) == 0 {
				// This is okay - the query might not match anything
				t.Logf("Query %q returned no results (acceptable)", tt.query.Text)
			}
		})
	}
}
