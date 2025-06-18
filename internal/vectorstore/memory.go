package vectorstore

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

// NewMemoryStore creates a new in-memory vector store
func NewMemoryStore(options ...MemoryStoreOption) *MemoryStore {
	opts := MemoryStoreOptions{
		AutoSaveInterval: 5 * time.Minute,
		MaxVectors:       10000,
		NormalizeVectors: false,
		EnableCache:      false,
		CacheSize:        1000,
	}

	for _, option := range options {
		option(&opts)
	}

	store := &MemoryStore{
		vectors: make(map[string]VectorEntry),
		options: opts,
		done:    make(chan bool),
	}

	// Initialize cache if enabled
	if opts.EnableCache {
		store.cache = make(map[string]CacheEntry)
		store.cacheKeys = make([]string, 0, opts.CacheSize)
	}

	// Start auto-save routine if enabled
	if opts.AutoSave && opts.AutoSaveInterval > 0 {
		store.ticker = time.NewTicker(opts.AutoSaveInterval)
		go store.autoSaveRoutine()
	}

	return store
}

// Store adds a vector to the store
func (ms *MemoryStore) Store(id, text string, vector []float32) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if we've reached the maximum number of vectors
	if ms.options.MaxVectors > 0 && len(ms.vectors) >= ms.options.MaxVectors {
		if _, exists := ms.vectors[id]; !exists {
			return fmt.Errorf("vector store is full (max %d vectors)", ms.options.MaxVectors)
		}
	}

	// Normalize vector if requested
	if ms.options.NormalizeVectors {
		vector = NormalizeVector(vector)
	}

	entry := VectorEntry{
		ID:        id,
		Text:      text,
		Vector:    vector,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	ms.vectors[id] = entry
	return nil
}

// Search finds the most similar vectors using cosine similarity
func (ms *MemoryStore) Search(vector []float32, topK int) ([]SearchResult, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if len(ms.vectors) == 0 {
		return []SearchResult{}, nil
	}

	// Normalize query vector if store normalizes vectors
	queryVector := vector
	if ms.options.NormalizeVectors {
		queryVector = NormalizeVector(vector)
	}

	// Calculate similarities for all vectors
	type scoredResult struct {
		entry SearchResult
		score float32
	}

	results := make([]scoredResult, 0, len(ms.vectors))
	for _, entry := range ms.vectors {
		similarity := ms.getCachedSimilarity(queryVector, entry.Vector, entry.ID)

		result := SearchResult{
			ID:     entry.ID,
			Score:  similarity,
			Text:   entry.Text,
			Vector: entry.Vector,
		}

		results = append(results, scoredResult{
			entry: result,
			score: similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Return top K results
	if topK > len(results) {
		topK = len(results)
	}

	searchResults := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		searchResults[i] = results[i].entry
	}

	return searchResults, nil
}

// Delete removes a vector from the store
func (ms *MemoryStore) Delete(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.vectors[id]; !exists {
		return fmt.Errorf("vector with ID %s not found", id)
	}

	delete(ms.vectors, id)
	return nil
}

// Close shuts down the vector store and saves if persistence is enabled
func (ms *MemoryStore) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Stop auto-save routine
	if ms.ticker != nil {
		ms.ticker.Stop()
		ms.done <- true
	}

	// Save to file if persistence is enabled
	if ms.options.PersistenceFile != "" {
		return ms.saveToFileUnsafe(ms.options.PersistenceFile)
	}

	return nil
}

// SaveToFile saves the vector store to a JSON file
func (ms *MemoryStore) SaveToFile(filename string) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return ms.saveToFileUnsafe(filename)
}

// saveToFileUnsafe saves without acquiring locks (internal use)
func (ms *MemoryStore) saveToFileUnsafe(filename string) error {
	file, err := os.Create(filename) // #nosec G304 -- filename is controlled by caller
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(ms.vectors); err != nil {
		return fmt.Errorf("failed to encode vectors: %w", err)
	}

	return nil
}

// LoadFromFile loads the vector store from a JSON file
func (ms *MemoryStore) LoadFromFile(filename string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	file, err := os.Open(filename) // #nosec G304 -- filename is controlled by caller
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	decoder := json.NewDecoder(file)
	vectors := make(map[string]VectorEntry)

	if err := decoder.Decode(&vectors); err != nil {
		return fmt.Errorf("failed to decode vectors: %w", err)
	}

	ms.vectors = vectors
	return nil
}

// autoSaveRoutine runs the automatic save routine
func (ms *MemoryStore) autoSaveRoutine() {
	for {
		select {
		case <-ms.ticker.C:
			if ms.options.PersistenceFile != "" {
				if err := ms.SaveToFile(ms.options.PersistenceFile); err != nil {
					// In a real implementation, you might want to log this error
					continue
				}
			}
		case <-ms.done:
			return
		}
	}
}

// Size returns the number of vectors in the store
func (ms *MemoryStore) Size() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.vectors)
}

// Get retrieves a vector entry by ID
func (ms *MemoryStore) Get(id string) (VectorEntry, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entry, exists := ms.vectors[id]
	return entry, exists
}

// List returns all vector IDs in the store
func (ms *MemoryStore) List() []string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	ids := make([]string, 0, len(ms.vectors))
	for id := range ms.vectors {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids
}

// Clear removes all vectors from the store
func (ms *MemoryStore) Clear() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.vectors = make(map[string]VectorEntry)
}

// UpdateMetadata updates the metadata for a vector entry
func (ms *MemoryStore) UpdateMetadata(id string, metadata map[string]interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	entry, exists := ms.vectors[id]
	if !exists {
		return fmt.Errorf("vector with ID %s not found", id)
	}

	entry.Metadata = metadata
	ms.vectors[id] = entry
	return nil
}

// SearchWithFilter searches vectors with a custom filter function
func (ms *MemoryStore) SearchWithFilter(vector []float32, topK int, filter func(VectorEntry) bool) ([]SearchResult, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if len(ms.vectors) == 0 {
		return []SearchResult{}, nil
	}

	// Normalize query vector if store normalizes vectors
	queryVector := vector
	if ms.options.NormalizeVectors {
		queryVector = NormalizeVector(vector)
	}

	// Calculate similarities for filtered vectors
	type scoredResult struct {
		entry SearchResult
		score float32
	}

	results := make([]scoredResult, 0, len(ms.vectors))
	for _, entry := range ms.vectors {
		// Apply filter
		if filter != nil && !filter(entry) {
			continue
		}

		similarity := ms.getCachedSimilarity(queryVector, entry.Vector, entry.ID)

		result := SearchResult{
			ID:     entry.ID,
			Score:  similarity,
			Text:   entry.Text,
			Vector: entry.Vector,
		}

		results = append(results, scoredResult{
			entry: result,
			score: similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Return top K results
	if topK > len(results) {
		topK = len(results)
	}

	searchResults := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		searchResults[i] = results[i].entry
	}

	return searchResults, nil
}

// ExportToWriter exports vectors to a writer in JSON format
func (ms *MemoryStore) ExportToWriter(writer io.Writer) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	return encoder.Encode(ms.vectors)
}

// ImportFromReader imports vectors from a reader in JSON format
func (ms *MemoryStore) ImportFromReader(reader io.Reader) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	decoder := json.NewDecoder(reader)
	vectors := make(map[string]VectorEntry)

	if err := decoder.Decode(&vectors); err != nil {
		return fmt.Errorf("failed to decode vectors: %w", err)
	}

	// Merge with existing vectors
	for id, entry := range vectors {
		ms.vectors[id] = entry
	}

	return nil
}

// getCachedSimilarity returns cached similarity or calculates and caches it
func (ms *MemoryStore) getCachedSimilarity(queryVector, targetVector []float32, targetID string) float32 {
	if !ms.options.EnableCache {
		return CosineSimilarity(queryVector, targetVector)
	}

	// Generate cache key
	cacheKey := ms.generateCacheKey(queryVector, targetID)

	// Check cache first
	ms.cacheMu.RLock()
	if entry, exists := ms.cache[cacheKey]; exists {
		ms.cacheMu.RUnlock()
		return entry.Similarity
	}
	ms.cacheMu.RUnlock()

	// Calculate similarity
	similarity := CosineSimilarity(queryVector, targetVector)

	// Cache the result
	ms.addToCache(cacheKey, similarity)

	return similarity
}

// generateCacheKey creates a unique key for the vector pair
func (ms *MemoryStore) generateCacheKey(queryVector []float32, targetID string) string {
	// Use a simple hash based on first few vector elements and target ID
	// This is much faster than MD5 for our use case
	var hash uint64 = 5381

	// Hash first 8 elements of query vector (or all if less than 8)
	limit := len(queryVector)
	if limit > 8 {
		limit = 8
	}

	for i := 0; i < limit; i++ {
		val := uint64(queryVector[i] * 1000000) // Convert to fixed-point for hashing
		hash = ((hash << 5) + hash) + val
	}

	// Hash target ID
	for _, b := range []byte(targetID) {
		hash = ((hash << 5) + hash) + uint64(b)
	}

	return fmt.Sprintf("%x", hash)
}

// addToCache adds a similarity calculation to the cache with LRU eviction
func (ms *MemoryStore) addToCache(key string, similarity float32) {
	ms.cacheMu.Lock()
	defer ms.cacheMu.Unlock()

	// If cache is full, remove oldest entry
	if len(ms.cache) >= ms.options.CacheSize {
		if len(ms.cacheKeys) > 0 {
			oldestKey := ms.cacheKeys[0]
			delete(ms.cache, oldestKey)
			ms.cacheKeys = ms.cacheKeys[1:]
		}
	}

	// Add new entry
	ms.cache[key] = CacheEntry{
		Similarity: similarity,
		Timestamp:  time.Now(),
	}
	ms.cacheKeys = append(ms.cacheKeys, key)
}

// ClearCache clears the similarity cache
func (ms *MemoryStore) ClearCache() {
	if !ms.options.EnableCache {
		return
	}

	ms.cacheMu.Lock()
	defer ms.cacheMu.Unlock()

	ms.cache = make(map[string]CacheEntry)
	ms.cacheKeys = make([]string, 0, ms.options.CacheSize)
}
