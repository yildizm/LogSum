package docstore

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of DocumentStore
type MemoryStore struct {
	mu        sync.RWMutex
	documents map[string]*Document
	index     *MemoryIndex
	listeners []ChangeListener
}

// NewMemoryStore creates a new in-memory document store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		documents: make(map[string]*Document),
		index:     NewMemoryIndex(),
		listeners: make([]ChangeListener, 0),
	}
}

// Add adds a document to the store
func (ms *MemoryStore) Add(doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}
	if doc.ID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if document already exists
	existing, exists := ms.documents[doc.ID]
	changeType := ChangeAdded
	if exists {
		changeType = ChangeModified
	}

	// Store the document
	ms.documents[doc.ID] = doc

	// Index the document
	if err := ms.index.IndexDocument(doc); err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	// Notify listeners
	change := &DocumentChange{
		Type:       changeType,
		DocumentID: doc.ID,
		Path:       doc.Path,
		Timestamp:  time.Now(),
	}
	ms.notifyListeners(change)

	// If we're updating, remove old document from index
	if exists && existing.Hash != doc.Hash {
		if err := ms.index.RemoveDocument(existing.ID); err != nil {
			return fmt.Errorf("failed to remove old document from index: %w", err)
		}
		if err := ms.index.IndexDocument(doc); err != nil {
			return fmt.Errorf("failed to reindex document: %w", err)
		}
	}

	return nil
}

// AddBatch adds multiple documents to the store
func (ms *MemoryStore) AddBatch(docs []*Document) error {
	if len(docs) == 0 {
		return nil
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, doc := range docs {
		if doc == nil || doc.ID == "" {
			continue
		}

		// Check if document already exists
		_, exists := ms.documents[doc.ID]
		changeType := ChangeAdded
		if exists {
			changeType = ChangeModified
		}

		// Store the document
		ms.documents[doc.ID] = doc

		// Index the document
		if err := ms.index.IndexDocument(doc); err != nil {
			return fmt.Errorf("failed to index document %s: %w", doc.ID, err)
		}

		// Notify listeners
		change := &DocumentChange{
			Type:       changeType,
			DocumentID: doc.ID,
			Path:       doc.Path,
			Timestamp:  time.Now(),
		}
		ms.notifyListeners(change)
	}

	return nil
}

// Get retrieves a document by ID
func (ms *MemoryStore) Get(id string) (*Document, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	doc, exists := ms.documents[id]
	if !exists {
		return nil, fmt.Errorf("document with ID %s not found", id)
	}

	return doc, nil
}

// Update updates an existing document
func (ms *MemoryStore) Update(doc *Document) error {
	if doc == nil || doc.ID == "" {
		return fmt.Errorf("invalid document")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	_, exists := ms.documents[doc.ID]
	if !exists {
		return fmt.Errorf("document with ID %s not found", doc.ID)
	}

	// Remove old document from index
	if err := ms.index.RemoveDocument(doc.ID); err != nil {
		return fmt.Errorf("failed to remove document from index: %w", err)
	}

	// Store updated document
	ms.documents[doc.ID] = doc

	// Reindex
	if err := ms.index.IndexDocument(doc); err != nil {
		return fmt.Errorf("failed to reindex document: %w", err)
	}

	// Notify listeners
	change := &DocumentChange{
		Type:       ChangeModified,
		DocumentID: doc.ID,
		Path:       doc.Path,
		Timestamp:  time.Now(),
	}
	ms.notifyListeners(change)

	return nil
}

// Delete removes a document from the store
func (ms *MemoryStore) Delete(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	doc, exists := ms.documents[id]
	if !exists {
		return fmt.Errorf("document with ID %s not found", id)
	}

	// Remove from store
	delete(ms.documents, id)

	// Remove from index
	if err := ms.index.RemoveDocument(id); err != nil {
		return fmt.Errorf("failed to remove document from index: %w", err)
	}

	// Notify listeners
	change := &DocumentChange{
		Type:       ChangeDeleted,
		DocumentID: id,
		Path:       doc.Path,
		Timestamp:  time.Now(),
	}
	ms.notifyListeners(change)

	return nil
}

// List returns documents matching the filter
func (ms *MemoryStore) List(filter *FilterOptions) ([]*Document, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var result []*Document
	for _, doc := range ms.documents {
		if ms.matchesFilter(doc, filter) {
			result = append(result, doc)
		}
	}

	// Apply pagination
	start := filter.Offset
	if start > len(result) {
		start = len(result)
	}

	end := start + filter.Limit
	if filter.Limit <= 0 || end > len(result) {
		end = len(result)
	}

	if start > 0 || end < len(result) {
		result = result[start:end]
	}

	return result, nil
}

// Search performs a search query
func (ms *MemoryStore) Search(query *SearchQuery) ([]*SearchResult, error) {
	if query == nil || query.Text == "" {
		return []*SearchResult{}, nil
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return ms.index.Search(query)
}

// SearchSections searches for sections matching the query
func (ms *MemoryStore) SearchSections(query *SearchQuery) ([]*Section, error) {
	results, err := ms.Search(query)
	if err != nil {
		return nil, err
	}

	var sections []*Section
	for _, result := range results {
		if result.Section != nil {
			sections = append(sections, result.Section)
		} else if result.Document != nil {
			sections = append(sections, result.Document.Sections...)
		}
	}

	return sections, nil
}

// SearchSimple performs a simple text search
func (ms *MemoryStore) SearchSimple(text string) ([]*SearchResult, error) {
	query := &SearchQuery{
		Text:      text,
		Highlight: true,
		Limit:     100,
	}
	return ms.Search(query)
}

// Reindex rebuilds the entire search index
func (ms *MemoryStore) Reindex() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.index = NewMemoryIndex()

	for _, doc := range ms.documents {
		if err := ms.index.IndexDocument(doc); err != nil {
			return fmt.Errorf("failed to reindex document %s: %w", doc.ID, err)
		}
	}

	return nil
}

// IndexDocument indexes a single document
func (ms *MemoryStore) IndexDocument(doc *Document) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.index.IndexDocument(doc)
}

// RemoveFromIndex removes a document from the search index
func (ms *MemoryStore) RemoveFromIndex(docID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.index.RemoveDocument(docID)
}

// Clear removes all documents from the store
func (ms *MemoryStore) Clear() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.documents = make(map[string]*Document)
	ms.index = NewMemoryIndex()

	return nil
}

// Stats returns statistics about the store
func (ms *MemoryStore) Stats() (*StoreStats, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var totalSize int64
	var sectionCount int64
	var lastIndexed time.Time

	for _, doc := range ms.documents {
		totalSize += doc.Size
		sectionCount += int64(len(doc.Sections))
		if doc.LastModified.After(lastIndexed) {
			lastIndexed = doc.LastModified
		}
	}

	indexStats, err := ms.index.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}

	avgDocSize := float64(0)
	if len(ms.documents) > 0 {
		avgDocSize = float64(totalSize) / float64(len(ms.documents))
	}

	return &StoreStats{
		DocumentCount:  int64(len(ms.documents)),
		SectionCount:   sectionCount,
		TotalSize:      totalSize,
		IndexSize:      indexStats.IndexSize,
		LastIndexed:    lastIndexed,
		MemoryUsage:    totalSize + indexStats.IndexSize,
		IndexedTerms:   indexStats.TermCount,
		AverageDocSize: avgDocSize,
	}, nil
}

// Close closes the store and releases resources
func (ms *MemoryStore) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.documents = nil
	ms.index = nil
	ms.listeners = nil

	return nil
}

// AddChangeListener adds a change listener
func (ms *MemoryStore) AddChangeListener(listener ChangeListener) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.listeners = append(ms.listeners, listener)
}

// ProcessChanges processes a batch of document changes
func (ms *MemoryStore) ProcessChanges(changes []*DocumentChange) error {
	for _, change := range changes {
		ms.notifyListeners(change)
	}
	return nil
}

// Helper methods

func (ms *MemoryStore) matchesFilter(doc *Document, filter *FilterOptions) bool {
	return ms.matchesTagFilter(doc, filter) &&
		ms.matchesAuthorFilter(doc, filter) &&
		ms.matchesDateFilter(doc, filter) &&
		ms.matchesPathFilter(doc, filter) &&
		ms.matchesLanguageFilter(doc, filter) &&
		ms.matchesFormatFilter(doc, filter)
}

func (ms *MemoryStore) matchesTagFilter(doc *Document, filter *FilterOptions) bool {
	if len(filter.Tags) == 0 {
		return true
	}
	if doc.Metadata == nil {
		return false
	}
	return ms.containsAnyTag(doc.Metadata.Tags, filter.Tags)
}

func (ms *MemoryStore) matchesAuthorFilter(doc *Document, filter *FilterOptions) bool {
	if filter.Author == "" {
		return true
	}
	if doc.Metadata == nil {
		return false
	}
	return doc.Metadata.Author == filter.Author
}

func (ms *MemoryStore) matchesDateFilter(doc *Document, filter *FilterOptions) bool {
	if filter.DateAfter.IsZero() && filter.DateBefore.IsZero() {
		return true
	}
	if doc.Metadata == nil || doc.Metadata.Date == nil {
		return true
	}

	if !filter.DateAfter.IsZero() && doc.Metadata.Date.Before(filter.DateAfter) {
		return false
	}
	if !filter.DateBefore.IsZero() && doc.Metadata.Date.After(filter.DateBefore) {
		return false
	}
	return true
}

func (ms *MemoryStore) matchesPathFilter(doc *Document, filter *FilterOptions) bool {
	if filter.PathPrefix == "" {
		return true
	}
	return strings.HasPrefix(doc.Path, filter.PathPrefix)
}

func (ms *MemoryStore) matchesLanguageFilter(doc *Document, filter *FilterOptions) bool {
	if filter.Language == "" {
		return true
	}
	if doc.Metadata == nil {
		return false
	}
	return doc.Metadata.Language == filter.Language
}

func (ms *MemoryStore) matchesFormatFilter(doc *Document, filter *FilterOptions) bool {
	if filter.Format == "" {
		return true
	}
	if doc.Metadata == nil {
		return false
	}
	return doc.Metadata.Format == filter.Format
}

func (ms *MemoryStore) containsAnyTag(docTags, filterTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range docTags {
		tagSet[strings.ToLower(tag)] = true
	}

	for _, filterTag := range filterTags {
		if tagSet[strings.ToLower(filterTag)] {
			return true
		}
	}

	return false
}

func (ms *MemoryStore) notifyListeners(change *DocumentChange) {
	for _, listener := range ms.listeners {
		listener.OnDocumentChanged(change)
	}
}
