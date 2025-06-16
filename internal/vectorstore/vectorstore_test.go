package vectorstore

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestCosineSimilarity tests the cosine similarity function
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 3},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0},
			b:        []float32{0, 1},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0},
			b:        []float32{-1, 0},
			expected: -1.0,
		},
		{
			name:     "different length vectors",
			a:        []float32{1, 2},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > 1e-6 {
				t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestEuclideanDistance tests the Euclidean distance function
func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "simple distance",
			a:        []float32{0, 0},
			b:        []float32{3, 4},
			expected: 5.0,
		},
		{
			name:     "different length vectors",
			a:        []float32{1, 2},
			b:        []float32{1, 2, 3},
			expected: float32(math.Inf(1)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EuclideanDistance(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > 1e-6 {
				t.Errorf("EuclideanDistance() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeVector tests vector normalization
func TestNormalizeVector(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected []float32
	}{
		{
			name:     "unit vector",
			input:    []float32{1, 0, 0},
			expected: []float32{1, 0, 0},
		},
		{
			name:     "simple vector",
			input:    []float32{3, 4},
			expected: []float32{0.6, 0.8},
		},
		{
			name:     "zero vector",
			input:    []float32{0, 0, 0},
			expected: []float32{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeVector(tt.input)
			for i, val := range result {
				if math.Abs(float64(val-tt.expected[i])) > 1e-6 {
					t.Errorf("NormalizeVector()[%d] = %v, want %v", i, val, tt.expected[i])
				}
			}
		})
	}
}

// TestTFIDFVectorizer tests the TF-IDF vectorizer
func TestTFIDFVectorizer(t *testing.T) {
	documents := []string{
		"the cat sat on the mat",
		"the dog ran in the park",
		"cats and dogs are pets",
	}

	vectorizer := NewTFIDFVectorizer(10)

	// Test that vectorizer needs to be fitted first
	_, err := vectorizer.Vectorize("test")
	if err == nil {
		t.Error("Expected error when vectorizing without fitting")
	}

	// Test fitting
	err = vectorizer.Fit(documents)
	if err != nil {
		t.Fatalf("Failed to fit vectorizer: %v", err)
	}

	if !vectorizer.IsFitted() {
		t.Error("Vectorizer should be fitted")
	}

	// Test vectorization
	vector, err := vectorizer.Vectorize("the cat")
	if err != nil {
		t.Fatalf("Failed to vectorize: %v", err)
	}

	if len(vector) != vectorizer.Dimension() {
		t.Errorf("Vector dimension = %d, want %d", len(vector), vectorizer.Dimension())
	}

	// Test FitTransform
	vectors, err := vectorizer.FitTransform(documents)
	if err != nil {
		t.Fatalf("Failed to fit and transform: %v", err)
	}

	if len(vectors) != len(documents) {
		t.Errorf("Number of vectors = %d, want %d", len(vectors), len(documents))
	}

	// Test empty corpus
	emptyVectorizer := NewTFIDFVectorizer(10)
	err = emptyVectorizer.Fit([]string{})
	if err == nil {
		t.Error("Expected error when fitting empty corpus")
	}
}

// TestMemoryStore tests the in-memory vector store
func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	// Test initial state
	if store.Size() != 0 {
		t.Errorf("New store size = %d, want 0", store.Size())
	}

	// Test storing vectors
	vector1 := []float32{1, 2, 3}
	err := store.Store("doc1", "test document 1", vector1)
	if err != nil {
		t.Fatalf("Failed to store vector: %v", err)
	}

	if store.Size() != 1 {
		t.Errorf("Store size after storing = %d, want 1", store.Size())
	}

	// Test retrieving vector
	entry, exists := store.Get("doc1")
	if !exists {
		t.Error("Vector should exist")
	}
	if entry.Text != "test document 1" {
		t.Errorf("Entry text = %s, want 'test document 1'", entry.Text)
	}

	// Test search
	vector2 := []float32{1, 2, 4}
	if err := store.Store("doc2", "test document 2", vector2); err != nil {
		t.Fatalf("Failed to store second vector: %v", err)
	}

	results, err := store.Search([]float32{1, 2, 3}, 2)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search results length = %d, want 2", len(results))
	}

	// First result should be exact match
	if results[0].ID != "doc1" {
		t.Errorf("First result ID = %s, want 'doc1'", results[0].ID)
	}
	if math.Abs(float64(results[0].Score-1.0)) > 1e-6 {
		t.Errorf("First result score = %f, want 1.0", results[0].Score)
	}

	// Test deletion
	err = store.Delete("doc1")
	if err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}

	if store.Size() != 1 {
		t.Errorf("Store size after deletion = %d, want 1", store.Size())
	}

	// Test deleting non-existent vector
	err = store.Delete("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent vector")
	}

	// Test clearing
	store.Clear()
	if store.Size() != 0 {
		t.Errorf("Store size after clearing = %d, want 0", store.Size())
	}
}

// TestMemoryStoreWithOptions tests memory store with various options
func TestMemoryStoreWithOptions(t *testing.T) {
	// Test with max vectors limit
	store := NewMemoryStore(WithMaxVectors(2))

	if err := store.Store("doc1", "text1", []float32{1, 2, 3}); err != nil {
		t.Fatalf("Failed to store doc1: %v", err)
	}
	if err := store.Store("doc2", "text2", []float32{2, 3, 4}); err != nil {
		t.Fatalf("Failed to store doc2: %v", err)
	}

	// This should fail due to max vectors limit
	err := store.Store("doc3", "text3", []float32{3, 4, 5})
	if err == nil {
		t.Error("Expected error when exceeding max vectors")
	}

	// Test with normalization
	normalizeStore := NewMemoryStore(WithNormalization())
	if err := normalizeStore.Store("doc1", "text1", []float32{3, 4, 0}); err != nil {
		t.Fatalf("Failed to store vector: %v", err)
	}

	entry, _ := normalizeStore.Get("doc1")
	magnitude := float32(0)
	for _, val := range entry.Vector {
		magnitude += val * val
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))

	if math.Abs(float64(magnitude-1.0)) > 1e-6 {
		t.Errorf("Normalized vector magnitude = %f, want 1.0", magnitude)
	}
}

// TestMemoryStorePersistence tests file persistence
func TestMemoryStorePersistence(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "vectors.json")

	// Create store and add some vectors
	store := NewMemoryStore(WithPersistence(filename))
	if err := store.Store("doc1", "text1", []float32{1, 2, 3}); err != nil {
		t.Fatalf("Failed to store doc1: %v", err)
	}
	if err := store.Store("doc2", "text2", []float32{4, 5, 6}); err != nil {
		t.Fatalf("Failed to store doc2: %v", err)
	}

	// Save to file
	err := store.SaveToFile(filename)
	if err != nil {
		t.Fatalf("Failed to save to file: %v", err)
	}

	// Create new store and load from file
	newStore := NewMemoryStore()
	err = newStore.LoadFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	if newStore.Size() != 2 {
		t.Errorf("Loaded store size = %d, want 2", newStore.Size())
	}

	entry, exists := newStore.Get("doc1")
	if !exists {
		t.Error("doc1 should exist in loaded store")
	}
	if entry.Text != "text1" {
		t.Errorf("Loaded entry text = %s, want 'text1'", entry.Text)
	}
}

// TestMemoryStoreExportImport tests export/import functionality
func TestMemoryStoreExportImport(t *testing.T) {
	store := NewMemoryStore()
	if err := store.Store("doc1", "text1", []float32{1, 2, 3}); err != nil {
		t.Fatalf("Failed to store doc1: %v", err)
	}
	if err := store.Store("doc2", "text2", []float32{4, 5, 6}); err != nil {
		t.Fatalf("Failed to store doc2: %v", err)
	}

	// Export to buffer
	var buf bytes.Buffer
	err := store.ExportToWriter(&buf)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	// Import to new store
	newStore := NewMemoryStore()
	err = newStore.ImportFromReader(&buf)
	if err != nil {
		t.Fatalf("Failed to import: %v", err)
	}

	if newStore.Size() != 2 {
		t.Errorf("Imported store size = %d, want 2", newStore.Size())
	}
}

// TestMemoryStoreConcurrency tests thread safety
func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore()
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				docID := fmt.Sprintf("doc_%d_%d", id, j)
				vector := []float32{float32(id), float32(j), float32(id + j)}
				if err := store.Store(docID, fmt.Sprintf("text_%d_%d", id, j), vector); err != nil {
					t.Errorf("Failed to store vector: %v", err)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				if _, err := store.Search([]float32{1, 2, 3}, 5); err != nil {
					t.Errorf("Search failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	expectedSize := numGoroutines * numOperations
	if store.Size() != expectedSize {
		t.Errorf("Store size after concurrent operations = %d, want %d", store.Size(), expectedSize)
	}
}

// TestMemoryStoreWithFilter tests search with custom filter
func TestMemoryStoreWithFilter(t *testing.T) {
	store := NewMemoryStore()

	// Add vectors with metadata
	if err := store.Store("doc1", "text1", []float32{1, 2, 3}); err != nil {
		t.Fatalf("Failed to store doc1: %v", err)
	}
	if err := store.Store("doc2", "text2", []float32{2, 3, 4}); err != nil {
		t.Fatalf("Failed to store doc2: %v", err)
	}
	if err := store.Store("doc3", "text3", []float32{3, 4, 5}); err != nil {
		t.Fatalf("Failed to store doc3: %v", err)
	}

	// Add metadata
	if err := store.UpdateMetadata("doc1", map[string]interface{}{"category": "A"}); err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}
	if err := store.UpdateMetadata("doc2", map[string]interface{}{"category": "B"}); err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}
	if err := store.UpdateMetadata("doc3", map[string]interface{}{"category": "A"}); err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	// Search with filter for category A
	results, err := store.SearchWithFilter([]float32{1, 2, 3}, 10, func(entry VectorEntry) bool {
		category, exists := entry.Metadata["category"]
		return exists && category == "A"
	})

	if err != nil {
		t.Fatalf("Failed to search with filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Filtered search results length = %d, want 2", len(results))
	}

	// Verify only category A results
	for _, result := range results {
		if result.ID != "doc1" && result.ID != "doc3" {
			t.Errorf("Unexpected result ID: %s", result.ID)
		}
	}
}

// TestMemoryStoreAutoSave tests auto-save functionality
func TestMemoryStoreAutoSave(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "autosave.json")

	// Create store with short auto-save interval
	store := NewMemoryStore(
		WithPersistence(filename),
		WithAutoSave(100*time.Millisecond),
	)
	defer func() { _ = store.Close() }()

	// Add a vector
	if err := store.Store("doc1", "text1", []float32{1, 2, 3}); err != nil {
		t.Fatalf("Failed to store vector: %v", err)
	}

	// Wait for auto-save and verify content multiple times
	var loaded bool
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)

		// Check if file exists and has content
		if info, err := os.Stat(filename); err == nil && info.Size() > 0 {
			// Try to load and verify content
			newStore := NewMemoryStore()
			if loadErr := newStore.LoadFromFile(filename); loadErr == nil && newStore.Size() == 1 {
				loaded = true
				break
			}
		}
	}

	if !loaded {
		t.Error("Auto-save should have created a valid file with content")
	}
}

// TestVectorizerConfiguration tests vectorizer configuration options
func TestVectorizerConfiguration(t *testing.T) {
	vectorizer := NewTFIDFVectorizer(100)

	// Test default configuration
	if vectorizer.GetVocabularySize() != 0 {
		t.Errorf("Initial vocabulary size = %d, want 0", vectorizer.GetVocabularySize())
	}

	// Test configuration methods
	vectorizer.SetMinWordLength(3)
	vectorizer.SetMaxWordLength(20)
	vectorizer.AddStopWords([]string{"custom", "stop", "words"})

	// Test with documents
	documents := []string{
		"this is a test document with custom words",
		"another test document for vectorization",
		"stop words should be filtered out",
	}

	err := vectorizer.Fit(documents)
	if err != nil {
		t.Fatalf("Failed to fit vectorizer: %v", err)
	}

	// Test vectorization with a word that should be in vocabulary
	vector, err := vectorizer.Vectorize("test document vectorization")
	if err != nil {
		t.Fatalf("Failed to vectorize: %v", err)
	}

	// Should have some non-zero values for words in vocabulary
	hasNonZero := false
	for _, val := range vector {
		if val != 0 {
			hasNonZero = true
			break
		}
	}

	if !hasNonZero {
		t.Error("Vector should have non-zero values for words in vocabulary")
	}
}

// BenchmarkCosineSimilarity benchmarks cosine similarity calculation
func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float32, 1000)
	vec := make([]float32, 1000)
	for i := range a {
		a[i] = float32(i)
		vec[i] = float32(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, vec)
	}
}

// BenchmarkMemoryStoreSearch benchmarks vector search
func BenchmarkMemoryStoreSearch(b *testing.B) {
	store := NewMemoryStore()

	// Add 1000 vectors
	for i := 0; i < 1000; i++ {
		vector := make([]float32, 100)
		for j := range vector {
			vector[j] = float32(i*j) / 100.0
		}
		if err := store.Store(fmt.Sprintf("doc%d", i), fmt.Sprintf("document %d", i), vector); err != nil {
			b.Fatalf("Failed to store vector: %v", err)
		}
	}

	queryVector := make([]float32, 100)
	for i := range queryVector {
		queryVector[i] = float32(i) / 100.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Search(queryVector, 10); err != nil {
			b.Errorf("Search failed: %v", err)
		}
	}
}

// BenchmarkTFIDFVectorizer benchmarks TF-IDF vectorization
func BenchmarkTFIDFVectorizer(b *testing.B) {
	documents := make([]string, 100)
	for i := range documents {
		documents[i] = fmt.Sprintf("document %d with some text content for testing vectorization performance", i)
	}

	vectorizer := NewTFIDFVectorizer(1000)
	if err := vectorizer.Fit(documents); err != nil {
		b.Fatalf("Failed to fit vectorizer: %v", err)
	}

	testText := "sample text for vectorization benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := vectorizer.Vectorize(testText); err != nil {
			b.Errorf("Vectorize failed: %v", err)
		}
	}
}
