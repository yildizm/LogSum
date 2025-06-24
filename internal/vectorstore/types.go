package vectorstore

import (
	"context"
	"sync"
	"time"
)

// VectorEntry represents a stored vector with metadata
type VectorEntry struct {
	ID        string
	Text      string
	Vector    []float32
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// MemoryStoreOptions configures the in-memory vector store
type MemoryStoreOptions struct {
	PersistenceFile   string
	AutoSave          bool
	AutoSaveInterval  time.Duration
	MaxVectors        int
	NormalizeVectors  bool
	EnableCache       bool
	CacheSize         int
	CancelCheckPeriod int // Iterations between cancellation checks (default: 100)
}

// CacheEntry represents a cached similarity calculation
type CacheEntry struct {
	Similarity float32
	Timestamp  time.Time
}

// MemoryStoreOption is a function type for configuring MemoryStore
type MemoryStoreOption func(*MemoryStoreOptions)

// WithPersistence enables disk persistence for the memory store
func WithPersistence(filename string) MemoryStoreOption {
	return func(opts *MemoryStoreOptions) {
		opts.PersistenceFile = filename
	}
}

// WithAutoSave enables automatic saving to disk
func WithAutoSave(interval time.Duration) MemoryStoreOption {
	return func(opts *MemoryStoreOptions) {
		opts.AutoSave = true
		opts.AutoSaveInterval = interval
	}
}

// WithMaxVectors limits the number of vectors stored
func WithMaxVectors(maxVectors int) MemoryStoreOption {
	return func(opts *MemoryStoreOptions) {
		opts.MaxVectors = maxVectors
	}
}

// WithNormalization enables automatic vector normalization
func WithNormalization() MemoryStoreOption {
	return func(opts *MemoryStoreOptions) {
		opts.NormalizeVectors = true
	}
}

// WithCache enables similarity calculation caching
func WithCache(cacheSize int) MemoryStoreOption {
	return func(opts *MemoryStoreOptions) {
		opts.EnableCache = true
		opts.CacheSize = cacheSize
	}
}

// WithCancelCheckPeriod sets how often to check for context cancellation
func WithCancelCheckPeriod(period int) MemoryStoreOption {
	return func(opts *MemoryStoreOptions) {
		opts.CancelCheckPeriod = period
	}
}

// MemoryStore implements VectorStore using in-memory storage
type MemoryStore struct {
	mu        sync.RWMutex
	vectors   map[string]VectorEntry
	options   MemoryStoreOptions
	ticker    *time.Ticker
	done      chan bool
	cache     map[string]CacheEntry
	cacheKeys []string
	cacheMu   sync.RWMutex

	// Auto-save routine management
	autoSaveCtx    context.Context
	autoSaveCancel context.CancelFunc
	autoSaveWg     sync.WaitGroup
}

// TFIDFVectorizer implements text vectorization using TF-IDF
type TFIDFVectorizer struct {
	mu            sync.RWMutex
	dimensions    int
	vocabulary    map[string]int
	idf           []float32
	documentCount int
	fitted        bool
	minWordLength int
	maxWordLength int
	stopWords     map[string]bool
}
