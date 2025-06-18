package vectorstore

import (
	"fmt"
	"math/rand"
	"testing"
)

// BenchmarkVectorStoreOperations benchmarks various vector store operations
func BenchmarkVectorStoreOperations(b *testing.B) {
	// Setup data
	vectorDim := 384
	numVectors := 1000

	// Generate test vectors
	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = generateRandomVector(vectorDim)
	}

	b.Run("Store_Operations", func(b *testing.B) {
		store := NewMemoryStore()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			idx := i % numVectors
			_ = store.Store(fmt.Sprintf("vec_%d", idx), fmt.Sprintf("text_%d", idx), vectors[idx])
		}
	})

	b.Run("Search_Without_Cache", func(b *testing.B) {
		store := NewMemoryStore()

		// Pre-populate store
		for i := 0; i < numVectors; i++ {
			_ = store.Store(fmt.Sprintf("vec_%d", i), fmt.Sprintf("text_%d", i), vectors[i])
		}

		queryVector := generateRandomVector(vectorDim)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := store.Search(queryVector, 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Search_With_Cache", func(b *testing.B) {
		store := NewMemoryStore(WithCache(500))

		// Pre-populate store
		for i := 0; i < numVectors; i++ {
			_ = store.Store(fmt.Sprintf("vec_%d", i), fmt.Sprintf("text_%d", i), vectors[i])
		}

		queryVector := generateRandomVector(vectorDim)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := store.Search(queryVector, 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Cosine_Similarity", func(b *testing.B) {
		vec1 := generateRandomVector(vectorDim)
		vec2 := generateRandomVector(vectorDim)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			CosineSimilarity(vec1, vec2)
		}
	})

	b.Run("Vector_Normalization", func(b *testing.B) {
		vec := generateRandomVector(vectorDim)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			NormalizeVector(vec)
		}
	})
}

// BenchmarkTFIDFVectorizerOperations benchmarks the TF-IDF vectorizer
func BenchmarkTFIDFVectorizerOperations(b *testing.B) {
	texts := []string{
		"database connection timeout error",
		"high memory usage detected",
		"failed to authenticate user",
		"network request timeout",
		"disk space running low",
		"cache miss rate high",
		"application startup failed",
		"user login successful",
		"data processing complete",
		"backup operation started",
	}

	b.Run("Fit_Vectorize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vectorizer := NewTFIDFVectorizer(100)
			_ = vectorizer.Fit(texts)
			for _, text := range texts {
				_, _ = vectorizer.Vectorize(text)
			}
		}
	})

	b.Run("Vectorize_Only", func(b *testing.B) {
		vectorizer := NewTFIDFVectorizer(100)
		_ = vectorizer.Fit(texts)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % len(texts)
			_, _ = vectorizer.Vectorize(texts[idx])
		}
	})
}

// BenchmarkSearchScaling tests search performance with different store sizes
func BenchmarkSearchScaling(b *testing.B) {
	vectorDim := 384
	storeSizes := []int{100, 500, 1000, 5000, 10000}

	for _, size := range storeSizes {
		b.Run(fmt.Sprintf("Search_%d_vectors", size), func(b *testing.B) {
			store := NewMemoryStore()

			// Pre-populate store
			for i := 0; i < size; i++ {
				vec := generateRandomVector(vectorDim)
				_ = store.Store(fmt.Sprintf("vec_%d", i), fmt.Sprintf("text_%d", i), vec)
			}

			queryVector := generateRandomVector(vectorDim)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := store.Search(queryVector, 10)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCacheEffectiveness compares cached vs non-cached performance
func BenchmarkCacheEffectiveness(b *testing.B) {
	vectorDim := 384
	numVectors := 1000

	// Generate test data
	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = generateRandomVector(vectorDim)
	}

	// Test with repeated queries (should benefit from cache)
	queryVectors := make([][]float32, 10)
	for i := 0; i < 10; i++ {
		queryVectors[i] = generateRandomVector(vectorDim)
	}

	b.Run("No_Cache_Repeated_Queries", func(b *testing.B) {
		store := NewMemoryStore()

		// Pre-populate store
		for i := 0; i < numVectors; i++ {
			_ = store.Store(fmt.Sprintf("vec_%d", i), fmt.Sprintf("text_%d", i), vectors[i])
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryIdx := i % len(queryVectors)
			_, err := store.Search(queryVectors[queryIdx], 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("With_Cache_Repeated_Queries", func(b *testing.B) {
		store := NewMemoryStore(WithCache(500))

		// Pre-populate store
		for i := 0; i < numVectors; i++ {
			_ = store.Store(fmt.Sprintf("vec_%d", i), fmt.Sprintf("text_%d", i), vectors[i])
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryIdx := i % len(queryVectors)
			_, err := store.Search(queryVectors[queryIdx], 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryUsage tests memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
	vectorDim := 384

	b.Run("Memory_Store_Growth", func(b *testing.B) {
		store := NewMemoryStore()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vec := generateRandomVector(vectorDim)
			_ = store.Store(fmt.Sprintf("vec_%d", i), fmt.Sprintf("text_%d", i), vec)
		}

		b.StopTimer()
		b.ReportMetric(float64(store.Size()), "vectors_stored")
	})
}

// generateRandomVector creates a random vector for testing
func generateRandomVector(dim int) []float32 {
	vec := make([]float32, dim)
	for i := range vec {
		//nolint:gosec // Using weak random for benchmark testing is acceptable
		vec[i] = rand.Float32()*2 - 1 // Range: -1 to 1
	}
	return vec
}
