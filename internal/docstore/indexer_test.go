package docstore

import (
	"strings"
	"testing"
)

func TestMemoryIndex_IndexDocument(t *testing.T) {
	index := NewMemoryIndex()

	doc := createTestDocument("test-1", "Test Document", "This is a test document with searchable content")

	err := index.IndexDocument(doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}

	// Verify document is stored
	if _, exists := index.documents[doc.ID]; !exists {
		t.Error("Document not found in index")
	}

	// Verify terms are indexed
	if len(index.terms) == 0 {
		t.Error("No terms were indexed")
	}

	// Check for specific terms
	expectedTerms := []string{"test", "document", "searchable", "content"}
	for _, term := range expectedTerms {
		if _, exists := index.terms[term]; !exists {
			t.Errorf("Term %q not found in index", term)
		}
	}
}

func TestMemoryIndex_RemoveDocument(t *testing.T) {
	index := NewMemoryIndex()

	doc := createTestDocument("test-1", "Test Document", "Unique content here")

	// Index document
	err := index.IndexDocument(doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}

	// Verify document is indexed
	if _, exists := index.documents[doc.ID]; !exists {
		t.Error("Document not found in index")
	}

	// Remove document
	if err := index.RemoveDocument(doc.ID); err != nil {
		t.Fatalf("Failed to remove document: %v", err)
	}

	// Verify document is removed
	if _, exists := index.documents[doc.ID]; exists {
		t.Error("Document still found in index after removal")
	}

	// Verify terms with only this document are removed
	if term, exists := index.terms["unique"]; exists {
		if len(term.Documents) > 0 {
			t.Error("Term should be removed when no documents reference it")
		}
	}
}

func TestMemoryIndex_Search(t *testing.T) {
	index := NewMemoryIndex()

	// Index multiple documents
	docs := []*Document{
		createTestDocument("doc-1", "API Documentation", "This document describes REST API endpoints and authentication"),
		createTestDocument("doc-2", "Setup Guide", "Installation and configuration guide for the application"),
		createTestDocument("doc-3", "API Reference", "Complete API reference with examples and parameters"),
	}

	for _, doc := range docs {
		if err := index.IndexDocument(doc); err != nil {
			t.Fatalf("Failed to index document %s: %v", doc.ID, err)
		}
	}

	// Test simple search
	query := &SearchQuery{
		Text:  "API",
		Limit: 10,
	}

	results, err := index.Search(query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results for 'API'")
	}

	// Verify results are relevant (should find doc-1 and doc-3)
	foundDocs := make(map[string]bool)
	for _, result := range results {
		foundDocs[result.Document.ID] = true
		if result.Score <= 0 {
			t.Errorf("Expected positive score, got %f", result.Score)
		}
	}

	if !foundDocs["doc-1"] || !foundDocs["doc-3"] {
		t.Error("Expected to find both API documents")
	}
}

func TestMemoryIndex_SearchPhrase(t *testing.T) {
	index := NewMemoryIndex()

	doc := createTestDocument("test-1", "Test Document", "This is a complete REST API documentation guide")

	err := index.IndexDocument(doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}

	// Search for exact phrase
	query := &SearchQuery{
		Text:  `"REST API"`,
		Limit: 10,
	}

	results, err := index.Search(query)
	if err != nil {
		t.Fatalf("Phrase search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected results for phrase search")
	}

	// Should find the document containing the exact phrase
	found := false
	for _, result := range results {
		if result.Document.ID == "test-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find document containing exact phrase")
	}
}

func TestMemoryIndex_SearchWithHighlighting(t *testing.T) {
	index := NewMemoryIndex()

	doc := createTestDocument("test-1", "Test Document", "This document contains important information about APIs")

	err := index.IndexDocument(doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}

	query := &SearchQuery{
		Text:      "APIs",
		Highlight: true,
		Limit:     10,
	}

	results, err := index.Search(query)
	if err != nil {
		t.Fatalf("Search with highlighting failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	// Check if highlighting is applied
	for _, result := range results {
		if result.Highlighted == "" {
			t.Error("Expected highlighted content")
		}
		if !strings.Contains(result.Highlighted, "**") {
			t.Error("Expected highlighting markers in content")
		}
	}
}

func TestMemoryIndex_FuzzySearch(t *testing.T) {
	index := NewMemoryIndex()

	doc := createTestDocument("test-1", "Test Document", "This document contains authentication information")

	err := index.IndexDocument(doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}

	// Search with a typo
	query := &SearchQuery{
		Text:  "autentication", // Missing 'h'
		Fuzzy: true,
		Limit: 10,
	}

	results, err := index.Search(query)
	if err != nil {
		t.Fatalf("Fuzzy search failed: %v", err)
	}

	// Should still find results due to fuzzy matching
	if len(results) == 0 {
		t.Error("Expected fuzzy search to find results")
	}
}

func TestMemoryIndex_Tokenize(t *testing.T) {
	index := NewMemoryIndex()

	tests := []struct {
		text     string
		expected []string
	}{
		{
			text:     "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			text:     "API-based authentication system",
			expected: []string{"api", "based", "authentication", "system"},
		},
		{
			text:     "test123 with numbers",
			expected: []string{"test123", "with", "numbers"},
		},
		{
			text:     "punctuation, and symbols!",
			expected: []string{"punctuation", "and", "symbols"},
		},
		{
			text:     "  extra   spaces  ",
			expected: []string{"extra", "spaces"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := index.tokenize(tt.text)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(result))
			}
			for i, token := range result {
				if i < len(tt.expected) && token != tt.expected[i] {
					t.Errorf("Token %d: expected %q, got %q", i, tt.expected[i], token)
				}
			}
		})
	}
}

func TestMemoryIndex_CalculateTFIDF(t *testing.T) {
	index := NewMemoryIndex()

	// Add some documents to establish IDF values
	docs := []*Document{
		createTestDocument("doc-1", "Doc 1", "test document one"),
		createTestDocument("doc-2", "Doc 2", "test document two"),
		createTestDocument("doc-3", "Doc 3", "different content here"),
	}

	for _, doc := range docs {
		if err := index.IndexDocument(doc); err != nil {
			t.Fatalf("Failed to index document: %v", err)
		}
	}

	// Test TF-IDF calculation
	score := index.calculateTFIDF(2, 10, 1.5) // termFreq=2, docLength=10, idf=1.5
	if score <= 0 {
		t.Errorf("Expected positive TF-IDF score, got %f", score)
	}

	// Test edge cases
	zeroScore := index.calculateTFIDF(0, 10, 1.5)
	if zeroScore != 0 {
		t.Errorf("Expected zero score for zero term frequency, got %f", zeroScore)
	}

	zeroDocLength := index.calculateTFIDF(2, 0, 1.5)
	if zeroDocLength != 0 {
		t.Errorf("Expected zero score for zero document length, got %f", zeroDocLength)
	}
}

func TestMemoryIndex_LevenshteinDistance(t *testing.T) {
	index := NewMemoryIndex()

	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			result := index.levenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestMemoryIndex_GetStats(t *testing.T) {
	index := NewMemoryIndex()

	// Test empty index stats
	stats, err := index.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TermCount != 0 {
		t.Errorf("Expected 0 terms, got %d", stats.TermCount)
	}
	if stats.DocumentCount != 0 {
		t.Errorf("Expected 0 documents, got %d", stats.DocumentCount)
	}

	// Add documents and test stats
	docs := []*Document{
		createTestDocument("doc-1", "Short doc", "short"),
		createTestDocument("doc-2", "Long document", "this is a much longer document with many more words"),
	}

	for _, doc := range docs {
		if err := index.IndexDocument(doc); err != nil {
			t.Fatalf("Failed to index document: %v", err)
		}
	}

	stats, err = index.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats after adding documents: %v", err)
	}
	if stats.TermCount == 0 {
		t.Error("Expected positive term count")
	}
	if stats.DocumentCount != 2 {
		t.Errorf("Expected 2 documents, got %d", stats.DocumentCount)
	}
	if stats.AverageTerms <= 0 {
		t.Error("Expected positive average terms")
	}
	if stats.LongestDoc <= stats.ShortestDoc {
		t.Error("Expected longest doc to be longer than shortest doc")
	}
}

func TestMemoryIndex_Rebuild(t *testing.T) {
	index := NewMemoryIndex()

	// Add initial documents
	doc1 := createTestDocument("doc-1", "Doc 1", "initial content")
	err := index.IndexDocument(doc1)
	if err != nil {
		t.Fatalf("Failed to index initial document: %v", err)
	}

	// Rebuild with new documents
	newDocs := []*Document{
		createTestDocument("doc-2", "Doc 2", "new content"),
		createTestDocument("doc-3", "Doc 3", "more content"),
	}

	err = index.Rebuild(newDocs)
	if err != nil {
		t.Fatalf("Failed to rebuild index: %v", err)
	}

	// Verify old document is gone
	if _, exists := index.documents["doc-1"]; exists {
		t.Error("Old document should be removed after rebuild")
	}

	// Verify new documents are present
	for _, doc := range newDocs {
		if _, exists := index.documents[doc.ID]; !exists {
			t.Errorf("New document %s not found after rebuild", doc.ID)
		}
	}
}

func BenchmarkMemoryIndex_IndexDocument(b *testing.B) {
	index := NewMemoryIndex()
	doc := createTestDocument("bench-doc", "Benchmark Document",
		"This is a longer document with more content to benchmark indexing performance. "+
			"It contains multiple sentences and various terms that will be tokenized and indexed. "+
			"The purpose is to measure how fast the indexing process works with realistic content.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a new document for each iteration to avoid duplicate ID issues
		testDoc := createTestDocument("bench-doc-"+string(rune(i)), doc.Title, doc.Content)
		if err := index.IndexDocument(testDoc); err != nil {
			b.Fatalf("Failed to index document: %v", err)
		}
	}
}

func BenchmarkMemoryIndex_Search(b *testing.B) {
	index := NewMemoryIndex()

	// Prepare index with multiple documents
	for i := 0; i < 1000; i++ {
		doc := createTestDocument("doc-"+string(rune(i)), "Document "+string(rune(i)),
			"This is document content with various terms for searching and benchmarking performance")
		if err := index.IndexDocument(doc); err != nil {
			b.Fatalf("Failed to index document: %v", err)
		}
	}

	query := &SearchQuery{
		Text:  "document performance",
		Limit: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := index.Search(query); err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}
