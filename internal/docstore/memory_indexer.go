package docstore

// MemoryIndexer wraps MemoryIndex to implement the Indexer interface
type MemoryIndexer struct {
	*MemoryIndex
}

// NewMemoryIndexer creates a new memory-based indexer
func NewMemoryIndexer() *MemoryIndexer {
	return &MemoryIndexer{
		MemoryIndex: NewMemoryIndex(),
	}
}

// IndexDocument implements the Indexer interface
func (mi *MemoryIndexer) IndexDocument(doc *Document) error {
	return mi.MemoryIndex.IndexDocument(doc)
}

// RemoveDocument implements the Indexer interface
func (mi *MemoryIndexer) RemoveDocument(docID string) error {
	return mi.MemoryIndex.RemoveDocument(docID)
}

// Search implements the Indexer interface
func (mi *MemoryIndexer) Search(query *SearchQuery) ([]*SearchResult, error) {
	return mi.MemoryIndex.Search(query)
}

// GetStats implements the Indexer interface
func (mi *MemoryIndexer) GetStats() (*IndexStats, error) {
	return mi.MemoryIndex.GetStats()
}

// Rebuild implements the Indexer interface
func (mi *MemoryIndexer) Rebuild(docs []*Document) error {
	return mi.MemoryIndex.Rebuild(docs)
}

// UpdateIndex implements the Indexer interface
func (mi *MemoryIndexer) UpdateIndex(changes []*DocumentChange) error {
	return mi.MemoryIndex.UpdateIndex(changes)
}
