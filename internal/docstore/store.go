package docstore

import (
	"context"
	"io"
)

// DocumentStore defines the interface for document storage and search
type DocumentStore interface {
	// Document operations
	Add(doc *Document) error
	AddBatch(docs []*Document) error
	Get(id string) (*Document, error)
	Update(doc *Document) error
	Delete(id string) error
	List(filter *FilterOptions) ([]*Document, error)

	// Search operations
	Search(query *SearchQuery) ([]*SearchResult, error)
	SearchSections(query *SearchQuery) ([]*Section, error)
	SearchSimple(text string) ([]*SearchResult, error)

	// Index operations
	Reindex() error
	IndexDocument(doc *Document) error
	RemoveFromIndex(docID string) error

	// Utility operations
	Clear() error
	Stats() (*StoreStats, error)
	Close() error

	// Incremental updates
	AddChangeListener(listener ChangeListener)
	ProcessChanges(changes []*DocumentChange) error
}

// ChangeListener is called when documents change
type ChangeListener interface {
	OnDocumentChanged(change *DocumentChange)
}

// Scanner defines the interface for scanning and parsing documents
type Scanner interface {
	// Scan a single file
	ScanFile(path string) (*Document, error)

	// Scan a directory recursively
	ScanDirectory(path string, patterns []string) ([]*Document, error)

	// Parse content from reader
	ParseContent(reader io.Reader, path string) (*Document, error)

	// Extract metadata from frontmatter
	ExtractMetadata(content string) (*Metadata, string, error)

	// Split document into sections
	SplitSections(content string) ([]*Section, error)
}

// Indexer defines the interface for building and maintaining search indexes
type Indexer interface {
	// Build index for a document
	IndexDocument(doc *Document) error

	// Remove document from index
	RemoveDocument(docID string) error

	// Search the index
	Search(query *SearchQuery) ([]*SearchResult, error)

	// Get index statistics
	GetStats() (*IndexStats, error)

	// Rebuild the entire index
	Rebuild(docs []*Document) error

	// Update index incrementally
	UpdateIndex(changes []*DocumentChange) error
}

// IndexStats provides statistics about the search index
type IndexStats struct {
	TermCount     int64   `json:"term_count"`
	DocumentCount int64   `json:"document_count"`
	IndexSize     int64   `json:"index_size"`
	AverageTerms  float64 `json:"average_terms"`
	LongestDoc    int     `json:"longest_doc"`
	ShortestDoc   int     `json:"shortest_doc"`
}

// Store represents the main document store implementation
type Store struct {
	store   DocumentStore
	scanner Scanner
	indexer Indexer
}

// NewStore creates a new document store with the given implementations
func NewStore(store DocumentStore, scanner Scanner, indexer Indexer) *Store {
	return &Store{
		store:   store,
		scanner: scanner,
		indexer: indexer,
	}
}

// Close closes the store and releases resources
func (s *Store) Close() error {
	return s.store.Close()
}

// ScanAndAdd scans a file and adds it to the store
func (s *Store) ScanAndAdd(path string) error {
	doc, err := s.scanner.ScanFile(path)
	if err != nil {
		return err
	}

	return s.store.Add(doc)
}

// ScanAndAddDirectory scans a directory and adds all matching files to the store
func (s *Store) ScanAndAddDirectory(path string, patterns []string) error {
	docs, err := s.scanner.ScanDirectory(path, patterns)
	if err != nil {
		return err
	}

	return s.store.AddBatch(docs)
}

// Search performs a search using the configured indexer and store
func (s *Store) Search(query *SearchQuery) ([]*SearchResult, error) {
	return s.store.Search(query)
}

// Get retrieves a document by ID
func (s *Store) Get(id string) (*Document, error) {
	return s.store.Get(id)
}

// List lists documents with optional filtering
func (s *Store) List(filter *FilterOptions) ([]*Document, error) {
	return s.store.List(filter)
}

// Stats returns statistics about the store
func (s *Store) Stats() (*StoreStats, error) {
	return s.store.Stats()
}

// ProcessDirectoryWithContext scans and indexes a directory with context support
func (s *Store) ProcessDirectoryWithContext(ctx context.Context, path string, patterns []string, progressCallback func(processed, total int)) error {
	docs, err := s.scanner.ScanDirectory(path, patterns)
	if err != nil {
		return err
	}

	total := len(docs)
	for i, doc := range docs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.store.Add(doc); err != nil {
				return err
			}
			if progressCallback != nil {
				progressCallback(i+1, total)
			}
		}
	}

	return nil
}
