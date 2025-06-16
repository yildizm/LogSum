package vectorstore

import (
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
	PersistenceFile  string
	AutoSave         bool
	AutoSaveInterval time.Duration
	MaxVectors       int
	NormalizeVectors bool
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

// MemoryStore implements VectorStore using in-memory storage
type MemoryStore struct {
	mu      sync.RWMutex
	vectors map[string]VectorEntry
	options MemoryStoreOptions
	ticker  *time.Ticker
	done    chan bool
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
