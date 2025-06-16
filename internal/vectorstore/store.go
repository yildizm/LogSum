package vectorstore

// VectorStore defines the interface for vector storage operations
type VectorStore interface {
	Store(id string, text string, vector []float32) error
	Search(vector []float32, topK int) ([]SearchResult, error)
	Delete(id string) error
	Close() error
}

// Vectorizer defines the interface for text vectorization
type Vectorizer interface {
	Vectorize(text string) ([]float32, error)
	Dimension() int
}

// SearchResult represents a search result from vector store
type SearchResult struct {
	ID     string
	Score  float32
	Text   string
	Vector []float32
}
