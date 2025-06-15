//go:build integration
// +build integration

package ollama

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

// TestOllamaIntegration tests against a real Ollama instance
// Run with: go test -tags=integration
func TestOllamaIntegration(t *testing.T) {
	// Skip if Ollama is not available
	if os.Getenv("OLLAMA_HOST") == "" && !isOllamaRunning() {
		t.Skip("Ollama not available - skipping integration test")
	}

	config := DefaultConfig()
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		config.BaseURL = host
	}

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("HealthCheck", func(t *testing.T) {
		err := provider.HealthCheck(ctx)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		if !provider.IsHealthy() {
			t.Error("Provider should be healthy")
		}
	})

	t.Run("ListModels", func(t *testing.T) {
		models, err := provider.ListModels(ctx)
		if err != nil {
			t.Fatalf("Failed to list models: %v", err)
		}

		if len(models) == 0 {
			t.Skip("No models available - skipping completion tests")
		}

		t.Logf("Available models: %d", len(models))
		for _, model := range models {
			t.Logf("  - %s (size: %d bytes)", model.Name, model.Size)
		}
	})

	t.Run("Complete", func(t *testing.T) {
		// First ensure we have a model available
		models, err := provider.ListModels(ctx)
		if err != nil {
			t.Fatalf("Failed to list models: %v", err)
		}

		if len(models) == 0 {
			// Try to pull a small model for testing
			t.Log("No models available, attempting to pull llama3.2:1b")
			err = provider.PullModel(ctx, "llama3.2:1b")
			if err != nil {
				t.Skipf("Failed to pull model for testing: %v", err)
			}
		}

		req := &ai.CompletionRequest{
			Prompt:      "What is 2+2? Give a very short answer.",
			Temperature: 0.1,
			MaxTokens:   50,
		}

		resp, err := provider.Complete(ctx, req)
		if err != nil {
			t.Fatalf("Completion failed: %v", err)
		}

		if resp.Content == "" {
			t.Error("Expected non-empty response content")
		}

		if resp.Usage == nil {
			t.Error("Expected usage information")
		} else {
			t.Logf("Tokens used - Prompt: %d, Completion: %d, Total: %d",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}

		t.Logf("Response: %s", resp.Content)
	})

	t.Run("CompleteStream", func(t *testing.T) {
		// Check if we have models available
		models, err := provider.ListModels(ctx)
		if err != nil {
			t.Fatalf("Failed to list models: %v", err)
		}

		if len(models) == 0 {
			t.Skip("No models available - skipping streaming test")
		}

		req := &ai.CompletionRequest{
			Prompt:      "Count from 1 to 3, one number per line.",
			Temperature: 0.1,
			MaxTokens:   20,
			Stream:      true,
		}

		ch, err := provider.CompleteStream(ctx, req)
		if err != nil {
			t.Fatalf("Streaming failed: %v", err)
		}

		var chunks []string
		var done bool

		for chunk := range ch {
			if chunk.Error != nil {
				t.Fatalf("Stream error: %v", chunk.Error)
			}

			if chunk.Content != "" {
				chunks = append(chunks, chunk.Content)
				t.Logf("Chunk: %q", chunk.Content)
			}

			if chunk.Done {
				done = true
			}
		}

		if len(chunks) == 0 {
			t.Error("Expected at least one content chunk")
		}

		if !done {
			t.Error("Expected stream to be marked as done")
		}
	})

	t.Run("ModelAvailability", func(t *testing.T) {
		// Test with a model that likely exists
		available, err := provider.IsModelAvailable(ctx, "llama3.2")
		if err != nil {
			t.Fatalf("Failed to check model availability: %v", err)
		}

		t.Logf("llama3.2 available: %v", available)

		// Test with a model that likely doesn't exist
		available, err = provider.IsModelAvailable(ctx, "nonexistent-model-12345")
		if err != nil {
			t.Fatalf("Failed to check model availability: %v", err)
		}

		if available {
			t.Error("Expected nonexistent model to be unavailable")
		}
	})
}

// isOllamaRunning checks if Ollama is running on the default port
func isOllamaRunning() bool {
	config := DefaultConfig()
	provider, err := New(config)
	if err != nil {
		return false
	}
	defer provider.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = provider.HealthCheck(ctx)
	return err == nil
}

// BenchmarkOllamaCompletion benchmarks completion performance
func BenchmarkOllamaCompletion(b *testing.B) {
	if os.Getenv("OLLAMA_HOST") == "" && !isOllamaRunning() {
		b.Skip("Ollama not available - skipping benchmark")
	}

	config := DefaultConfig()
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		config.BaseURL = host
	}

	provider, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Check if models are available
	models, err := provider.ListModels(ctx)
	if err != nil {
		b.Fatalf("Failed to list models: %v", err)
	}

	if len(models) == 0 {
		b.Skip("No models available - skipping benchmark")
	}

	req := &ai.CompletionRequest{
		Prompt:      "What is the capital of France?",
		Temperature: 0.1,
		MaxTokens:   10,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := provider.Complete(ctx, req)
		if err != nil {
			b.Fatalf("Completion failed: %v", err)
		}
	}
}
