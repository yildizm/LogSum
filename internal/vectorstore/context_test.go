package vectorstore

import (
	"context"
	"testing"
	"time"
)

func TestSearchWithContext_Cancellation(t *testing.T) {
	// Create a memory store with many vectors to ensure we hit the cancellation check
	store := NewMemoryStore()

	// Add enough vectors to guarantee we cross the 100-iteration check threshold
	for i := 0; i < 500; i++ {
		vector := []float32{float32(i), float32(i * 2)}
		err := store.Store(string(rune(i)), "text", vector)
		if err != nil {
			t.Fatalf("Failed to store vector: %v", err)
		}
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// This should be cancelled because context is already cancelled
	queryVector := []float32{1.0, 2.0}
	_, err := store.SearchWithContext(ctx, queryVector, 10)

	// Should return context cancelled error
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestSearchWithContext_Success(t *testing.T) {
	// Create a memory store with a few vectors
	store := NewMemoryStore()

	// Add a few vectors
	for i := 0; i < 5; i++ {
		vector := []float32{float32(i), float32(i * 2)}
		err := store.Store(string(rune(i)), "text", vector)
		if err != nil {
			t.Fatalf("Failed to store vector: %v", err)
		}
	}

	// Create a context with plenty of time
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This should succeed
	queryVector := []float32{1.0, 2.0}
	results, err := store.SearchWithContext(ctx, queryVector, 3)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected some results, got none")
	}
}
