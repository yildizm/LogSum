package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

func TestProvider_New(t *testing.T) {
	config := DefaultConfig()

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Name() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", provider.Name())
	}

	if !provider.SupportsStreaming() {
		t.Error("Expected provider to support streaming")
	}

	if provider.MaxTokens() != config.MaxTokens {
		t.Errorf("Expected max tokens %d, got %d", config.MaxTokens, provider.MaxTokens())
	}
}

func TestProvider_Complete(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected path '/api/generate', got '%s'", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}

		// Verify request body
		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Model == "" {
			t.Error("Expected model to be set")
		}

		if req.Prompt == "" {
			t.Error("Expected prompt to be set")
		}

		// Mock response
		resp := GenerateResponse{
			Model:              req.Model,
			Response:           "This is a test response.",
			Done:               true,
			CreatedAt:          time.Now(),
			PromptEvalCount:    10,
			EvalCount:          5,
			TotalDuration:      1000000,
			LoadDuration:       100000,
			PromptEvalDuration: 500000,
			EvalDuration:       400000,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server URL
	config := DefaultConfig()
	config.BaseURL = server.URL

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test completion
	ctx := context.Background()
	req := &ai.CompletionRequest{
		Prompt:      "Test prompt",
		Model:       "llama2",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Failed to complete: %v", err)
	}

	if resp.Content != "This is a test response." {
		t.Errorf("Expected content 'This is a test response.', got '%s'", resp.Content)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected prompt tokens 10, got %d", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("Expected completion tokens 5, got %d", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", resp.Usage.TotalTokens)
	}
}

func TestProvider_CompleteStream(t *testing.T) {
	// Create mock server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected path '/api/generate', got '%s'", r.URL.Path)
		}

		// Verify streaming request
		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream to be true")
		}

		w.Header().Set("Content-Type", "application/json")

		// Send streaming responses
		responses := []GenerateResponse{
			{Model: req.Model, Response: "Hello", Done: false},
			{Model: req.Model, Response: " world", Done: false},
			{Model: req.Model, Response: "!", Done: true},
		}

		for _, resp := range responses {
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	// Create provider with mock server URL
	config := DefaultConfig()
	config.BaseURL = server.URL

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test streaming completion
	ctx := context.Background()
	req := &ai.CompletionRequest{
		Prompt:      "Test prompt",
		Model:       "llama2",
		Temperature: 0.7,
		Stream:      true,
	}

	ch, err := provider.CompleteStream(ctx, req)
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}

	var content strings.Builder
	var chunks int
	var done bool

	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}

		content.WriteString(chunk.Content)
		chunks++

		if chunk.Done {
			done = true
		}
	}

	if chunks != 3 {
		t.Errorf("Expected 3 chunks, got %d", chunks)
	}

	if !done {
		t.Error("Expected stream to be done")
	}

	if content.String() != "Hello world!" {
		t.Errorf("Expected content 'Hello world!', got '%s'", content.String())
	}
}

func TestProvider_HealthCheck(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected path '/api/tags', got '%s'", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got '%s'", r.Method)
		}

		// Mock tags response
		resp := TagsResponse{
			Models: []Model{
				{Name: "llama2", Size: 1000000, ModifiedAt: time.Now()},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server URL
	config := DefaultConfig()
	config.BaseURL = server.URL

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test health check
	ctx := context.Background()
	err = provider.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if !provider.IsHealthy() {
		t.Error("Expected provider to be healthy")
	}
}

func TestProvider_ListModels(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TagsResponse{
			Models: []Model{
				{Name: "llama2", Size: 1000000, ModifiedAt: time.Now()},
				{Name: "mistral", Size: 2000000, ModifiedAt: time.Now()},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server URL
	config := DefaultConfig()
	config.BaseURL = server.URL

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test list models
	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].Name != "llama2" {
		t.Errorf("Expected first model 'llama2', got '%s'", models[0].Name)
	}

	if models[1].Name != "mistral" {
		t.Errorf("Expected second model 'mistral', got '%s'", models[1].Name)
	}
}

func TestProvider_CountTokens(t *testing.T) {
	config := DefaultConfig()
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	text := "This is a test text with some words."
	tokens, err := provider.CountTokens(text)
	if err != nil {
		t.Fatalf("Failed to count tokens: %v", err)
	}

	// Simple estimation should be around text length / 4
	expectedTokens := len(text) / 4
	if tokens != expectedTokens {
		t.Errorf("Expected approximately %d tokens, got %d", expectedTokens, tokens)
	}
}

func TestProvider_TruncateToFit(t *testing.T) {
	config := DefaultConfig()
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	text := "This is a long text that needs to be truncated to fit within token limits."
	maxTokens := 5 // Very small limit to force truncation

	truncated, err := provider.TruncateToFit(text, maxTokens)
	if err != nil {
		t.Fatalf("Failed to truncate text: %v", err)
	}

	if len(truncated) >= len(text) {
		t.Error("Expected text to be truncated")
	}

	// Verify truncated text token count is within limit
	tokens, _ := provider.CountTokens(truncated)
	if tokens > maxTokens {
		t.Errorf("Truncated text exceeds token limit: %d > %d", tokens, maxTokens)
	}
}

func TestProvider_ErrorHandling(t *testing.T) {
	// Test with server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "Internal server error"})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	req := &ai.CompletionRequest{
		Prompt: "Test prompt",
	}

	_, err = provider.Complete(ctx, req)
	if err == nil {
		t.Error("Expected error from server, got nil")
	}

	// Verify error type
	if providerErr, ok := err.(*ai.ProviderError); ok {
		if providerErr.Provider != "ollama" {
			t.Errorf("Expected provider 'ollama', got '%s'", providerErr.Provider)
		}
	} else {
		t.Errorf("Expected ProviderError, got %T", err)
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "valid config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "empty base URL",
			config: &Config{
				BaseURL:      "",
				DefaultModel: "llama2",
				Timeout:      30 * time.Second,
				MaxTokens:    4096,
			},
			expectError: true,
		},
		{
			name: "empty default model",
			config: &Config{
				BaseURL:      "http://localhost:11434",
				DefaultModel: "",
				Timeout:      30 * time.Second,
				MaxTokens:    4096,
			},
			expectError: true,
		},
		{
			name: "negative timeout",
			config: &Config{
				BaseURL:      "http://localhost:11434",
				DefaultModel: "llama2",
				Timeout:      -1 * time.Second,
				MaxTokens:    4096,
			},
			expectError: true,
		},
		{
			name: "invalid temperature",
			config: &Config{
				BaseURL:            "http://localhost:11434",
				DefaultModel:       "llama2",
				Timeout:            30 * time.Second,
				MaxTokens:          4096,
				DefaultTemperature: 1.5,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}
