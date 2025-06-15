package docstore

import (
	"time"
)

// Document represents a full document in the store
type Document struct {
	ID           string     `json:"id"`
	Path         string     `json:"path"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Metadata     *Metadata  `json:"metadata"`
	Sections     []*Section `json:"sections"`
	LastModified time.Time  `json:"last_modified"`
	Size         int64      `json:"size"`
	Hash         string     `json:"hash"`
}

// Section represents a document chunk/section
type Section struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	Heading    string `json:"heading"`
	Content    string `json:"content"`
	Level      int    `json:"level"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	WordCount  int    `json:"word_count"`
}

// Metadata holds flexible metadata for documents
type Metadata struct {
	Tags     []string               `json:"tags"`
	Author   string                 `json:"author"`
	Date     *time.Time             `json:"date"`
	Custom   map[string]interface{} `json:"custom"`
	Language string                 `json:"language"`
	Format   string                 `json:"format"`
}

// SearchResult represents a search result with relevance scoring
type SearchResult struct {
	Document    *Document `json:"document"`
	Section     *Section  `json:"section,omitempty"`
	Score       float64   `json:"score"`
	Matches     []Match   `json:"matches"`
	Highlighted string    `json:"highlighted"`
}

// Match represents a specific match within content
type Match struct {
	Field   string `json:"field"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Text    string `json:"text"`
	Context string `json:"context"`
}

// FilterOptions for listing and filtering documents
type FilterOptions struct {
	Tags       []string  `json:"tags,omitempty"`
	Author     string    `json:"author,omitempty"`
	DateAfter  time.Time `json:"date_after,omitempty"`
	DateBefore time.Time `json:"date_before,omitempty"`
	PathPrefix string    `json:"path_prefix,omitempty"`
	Language   string    `json:"language,omitempty"`
	Format     string    `json:"format,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

// StoreStats provides statistics about the document store
type StoreStats struct {
	DocumentCount  int64     `json:"document_count"`
	SectionCount   int64     `json:"section_count"`
	TotalSize      int64     `json:"total_size"`
	IndexSize      int64     `json:"index_size"`
	LastIndexed    time.Time `json:"last_indexed"`
	MemoryUsage    int64     `json:"memory_usage"`
	IndexedTerms   int64     `json:"indexed_terms"`
	AverageDocSize float64   `json:"average_doc_size"`
}

// SearchQuery represents a structured search query
type SearchQuery struct {
	Text      string        `json:"text"`
	Fields    []string      `json:"fields,omitempty"`
	Filters   FilterOptions `json:"filters,omitempty"`
	Fuzzy     bool          `json:"fuzzy,omitempty"`
	Highlight bool          `json:"highlight,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Offset    int           `json:"offset,omitempty"`
	SortBy    string        `json:"sort_by,omitempty"`
	SortOrder string        `json:"sort_order,omitempty"`
}

// IndexTerm represents a term in the inverted index
type IndexTerm struct {
	Term      string             `json:"term"`
	Documents map[string]*TermOc `json:"documents"`
	IDF       float64            `json:"idf"`
}

// TermOccurrence tracks term occurrence in a document
type TermOc struct {
	Frequency int      `json:"frequency"`
	Positions []int    `json:"positions"`
	Fields    []string `json:"fields"`
}

// DocumentChange represents a change to track for incremental updates
type DocumentChange struct {
	Type       ChangeType `json:"type"`
	DocumentID string     `json:"document_id"`
	Path       string     `json:"path"`
	Timestamp  time.Time  `json:"timestamp"`
}

// ChangeType represents the type of document change
type ChangeType int

const (
	ChangeAdded ChangeType = iota
	ChangeModified
	ChangeDeleted
)

func (ct ChangeType) String() string {
	switch ct {
	case ChangeAdded:
		return "added"
	case ChangeModified:
		return "modified"
	case ChangeDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}
