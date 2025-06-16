package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/vectorstore"
)

// VectorIntegrationExample demonstrates how to use the vector store
// for semantic search capabilities in LogSum
//
//nolint:gocyclo // Demo function intentionally shows multiple features
func main() {
	// Create vectorizer and store
	vectorizer := vectorstore.NewTFIDFVectorizer(1000)
	store := vectorstore.NewMemoryStore(
		vectorstore.WithMaxVectors(10000),
		vectorstore.WithNormalization(),
	)
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Failed to close store: %v", err)
		}
	}()

	// Sample documentation corpus (similar to what would be indexed)
	corpus := []string{
		"Database connection timeout error - check network configuration",
		"Failed to connect to PostgreSQL database server",
		"Network timeout during HTTP request to external API",
		"Authentication failed - invalid credentials provided",
		"File not found error when accessing configuration file",
		"Memory allocation failure in application startup",
		"SSL certificate verification failed during HTTPS connection",
		"Disk space full error - unable to write log files",
		"Service unavailable - external dependency not responding",
		"Invalid JSON format in API response payload",
	}

	// Train vectorizer on corpus
	fmt.Println("Training vectorizer on document corpus...")
	if err := vectorizer.Fit(corpus); err != nil {
		log.Printf("Failed to train vectorizer: %v", err)
		return
	}

	fmt.Printf("Vectorizer trained with %d vocabulary terms\n", vectorizer.GetVocabularySize())

	// Index documents
	fmt.Println("\nIndexing documents...")
	for i, doc := range corpus {
		vector, err := vectorizer.Vectorize(doc)
		if err != nil {
			log.Printf("Failed to vectorize document %d: %v", i, err)
			continue
		}

		docID := fmt.Sprintf("doc_%d", i)
		if err := store.Store(docID, doc, vector); err != nil {
			log.Printf("Failed to store vector for document %d: %v", i, err)
			continue
		}
	}

	fmt.Printf("Indexed %d documents\n", store.Size())

	// Demonstrate semantic search queries
	queries := []string{
		"database connection error",
		"network timeout problem",
		"authentication issue",
		"storage space problem",
		"certificate validation error",
	}

	fmt.Println("\n=== Semantic Search Results ===")

	for _, query := range queries {
		fmt.Printf("\nQuery: %s\n", query)
		fmt.Println(strings.Repeat("-", 50))

		// Vectorize query
		queryVector, err := vectorizer.Vectorize(query)
		if err != nil {
			log.Printf("Failed to vectorize query '%s': %v", query, err)
			continue
		}

		// Search for similar documents
		results, err := store.Search(queryVector, 3)
		if err != nil {
			log.Printf("Failed to search for query '%s': %v", query, err)
			continue
		}

		// Display results
		for i, result := range results {
			fmt.Printf("%d. [Score: %.3f] %s\n", i+1, result.Score, result.Text)
		}
	}

	// Demonstrate filtered search
	fmt.Println("\n=== Filtered Search Example ===")

	// Add metadata to documents
	if err := store.UpdateMetadata("doc_0", map[string]interface{}{"category": "database"}); err != nil {
		log.Printf("Failed to update metadata for doc_0: %v", err)
	}
	if err := store.UpdateMetadata("doc_1", map[string]interface{}{"category": "database"}); err != nil {
		log.Printf("Failed to update metadata for doc_1: %v", err)
	}
	if err := store.UpdateMetadata("doc_2", map[string]interface{}{"category": "network"}); err != nil {
		log.Printf("Failed to update metadata for doc_2: %v", err)
	}
	if err := store.UpdateMetadata("doc_6", map[string]interface{}{"category": "security"}); err != nil {
		log.Printf("Failed to update metadata for doc_6: %v", err)
	}

	// Search only in database category
	queryVector, _ := vectorizer.Vectorize("database connection problem")
	filteredResults, err := store.SearchWithFilter(queryVector, 5, func(entry vectorstore.VectorEntry) bool {
		category, exists := entry.Metadata["category"]
		return exists && category == "database"
	})

	if err != nil {
		log.Printf("Failed to perform filtered search: %v", err)
	} else {
		fmt.Println("\nDatabase-only search results:")
		for i, result := range filteredResults {
			fmt.Printf("%d. [Score: %.3f] %s\n", i+1, result.Score, result.Text)
		}
	}

	// Demonstrate persistence
	fmt.Println("\n=== Persistence Example ===")
	persistentStore := vectorstore.NewMemoryStore(
		vectorstore.WithPersistence("vector_store.json"),
		vectorstore.WithAutoSave(5*time.Second),
	)
	defer func() {
		if err := persistentStore.Close(); err != nil {
			log.Printf("Failed to close persistent store: %v", err)
		}
	}()

	// Add some vectors
	sampleDoc := "This is a sample document for persistence testing"
	sampleVector, _ := vectorizer.Vectorize(sampleDoc)
	if err := persistentStore.Store("sample_doc", sampleDoc, sampleVector); err != nil {
		log.Printf("Failed to store sample document: %v", err)
	}

	fmt.Println("Created persistent store with auto-save enabled")
	fmt.Println("Vector store will automatically save to 'vector_store.json'")

	// Performance demonstration
	fmt.Println("\n=== Performance Information ===")
	fmt.Printf("Vector dimension: %d\n", vectorizer.Dimension())
	fmt.Printf("Total vectors stored: %d\n", store.Size())
	fmt.Printf("Memory usage: ~%.2f MB (estimated)\n",
		float64(store.Size()*vectorizer.Dimension()*4)/1024/1024) // 4 bytes per float32
}
