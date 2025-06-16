package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

const testAPIKey = "test-api-key"

func TestProvider_New(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: true, // Should fail due to missing API key
		},
		{
			name: "valid config",
			config: &Config{
				APIKey:             testAPIKey,
				BaseURL:            DefaultBaseURL,
				DefaultModel:       DefaultModel,
				MaxTokens:          DefaultMaxTokens,
				DefaultTemperature: DefaultTemperature,
				Timeout:            DefaultTimeout,
			},
			wantErr: false,
		},
		{
			name: "invalid base URL",
			config: &Config{
				APIKey:  testAPIKey,
				BaseURL: "http://[::1]:namedport",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &Config{
				BaseURL: DefaultBaseURL,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("New() returned nil provider without error")
			}
			if provider != nil {
				_ = provider.Close()
			}
		})
	}
}

func TestProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected /v1/chat/completions, got %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+testAPIKey {
			t.Errorf("Expected Bearer %s, got %s", testAPIKey, auth)
		}

		body, _ := io.ReadAll(r.Body)
		var req ChatCompletionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if req.Stream {
			t.Error("Expected stream=false for Complete method")
		}

		response := ChatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "This is a test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: ChatCompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		APIKey:             testAPIKey,
		BaseURL:            server.URL,
		DefaultModel:       "gpt-3.5-turbo",
		MaxTokens:          4096,
		DefaultTemperature: 0.7,
		Timeout:            30 * time.Second,
	}

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	req := &ai.CompletionRequest{
		Prompt:    "Hello, world!",
		RequestID: "test-request",
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp == nil {
		t.Fatal("Complete() returned nil response")
	}

	if resp.Content != "This is a test response" {
		t.Errorf("Expected 'This is a test response', got %s", resp.Content)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("Expected 'stop', got %s", resp.FinishReason)
	}

	if resp.Usage == nil {
		t.Error("Expected usage information")
	} else if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestProvider_CompleteStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var req ChatCompletionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream=true for CompleteStream method")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		chunks := []string{
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	config := &Config{
		APIKey:             testAPIKey,
		BaseURL:            server.URL,
		DefaultModel:       "gpt-3.5-turbo",
		MaxTokens:          4096,
		DefaultTemperature: 0.7,
		Timeout:            30 * time.Second,
	}

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	req := &ai.CompletionRequest{
		Prompt:    "Hello",
		RequestID: "test-request",
	}

	ch, err := provider.CompleteStream(context.Background(), req)
	if err != nil {
		t.Fatalf("CompleteStream() error = %v", err)
	}

	var content strings.Builder
	chunks := make([]ai.StreamChunk, 0, 4)

	for chunk := range ch {
		chunks = append(chunks, chunk)
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
		}
		if chunk.Done {
			break
		}
	}

	expectedContent := "Hello world!"
	if content.String() != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, content.String())
	}

	if len(chunks) < 3 {
		t.Errorf("Expected at least 3 chunks, got %d", len(chunks))
	}

	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Done {
		t.Error("Expected last chunk to be marked as done")
	}
}

func TestProvider_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		expectedHealth bool
	}{
		{
			name: "healthy response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/models" {
					t.Errorf("Expected /v1/models, got %s", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(ModelListResponse{
					Object: "list",
					Data:   []Model{},
				})
			},
			wantErr:        false,
			expectedHealth: true,
		},
		{
			name: "unauthorized response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(ErrorResponse{
					Error: ErrorDetail{
						Message: "Invalid API key",
						Type:    "invalid_request_error",
					},
				})
			},
			wantErr:        true,
			expectedHealth: false,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:        true,
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			config := &Config{
				APIKey:             testAPIKey,
				BaseURL:            server.URL,
				DefaultModel:       "gpt-3.5-turbo",
				MaxTokens:          4096,
				DefaultTemperature: 0.7,
				Timeout:            30 * time.Second,
			}

			provider, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}
			defer func() { _ = provider.Close() }()

			err = provider.HealthCheck(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}

			if provider.IsHealthy() != tt.expectedHealth {
				t.Errorf("Expected health %v, got %v", tt.expectedHealth, provider.IsHealthy())
			}
		})
	}
}

func TestProvider_GetModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("Expected /v1/models, got %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+testAPIKey {
			t.Errorf("Expected Bearer %s, got %s", testAPIKey, auth)
		}

		response := ModelListResponse{
			Object: "list",
			Data: []Model{
				{
					ID:      "gpt-3.5-turbo",
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: "openai",
				},
				{
					ID:      "gpt-4",
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: "openai",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		APIKey:             testAPIKey,
		BaseURL:            server.URL,
		DefaultModel:       "gpt-3.5-turbo",
		MaxTokens:          4096,
		DefaultTemperature: 0.7,
		Timeout:            30 * time.Second,
	}

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	models, err := provider.GetModels()
	if err != nil {
		t.Fatalf("GetModels() error = %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	expectedModels := map[string]int{
		"gpt-3.5-turbo": 4096,
		"gpt-4":         8192,
	}

	for _, model := range models {
		expectedTokens, exists := expectedModels[model.ID]
		if !exists {
			t.Errorf("Unexpected model: %s", model.ID)
			continue
		}

		if model.MaxTokens != expectedTokens {
			t.Errorf("Model %s: expected %d tokens, got %d", model.ID, expectedTokens, model.MaxTokens)
		}

		if model.Provider != "openai" {
			t.Errorf("Model %s: expected provider 'openai', got '%s'", model.ID, model.Provider)
		}
	}
}

func TestProvider_RateLimitHandling(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
				},
			})
			return
		}

		response := ChatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-3.5-turbo",
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Success after retries",
					},
					FinishReason: "stop",
				},
			},
			Usage: ChatCompletionUsage{
				PromptTokens:     5,
				CompletionTokens: 3,
				TotalTokens:      8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		APIKey:             testAPIKey,
		BaseURL:            server.URL,
		DefaultModel:       "gpt-3.5-turbo",
		MaxTokens:          4096,
		DefaultTemperature: 0.7,
		Timeout:            30 * time.Second,
	}

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	req := &ai.CompletionRequest{
		Prompt:    "Test retry",
		RequestID: "test-request",
	}

	start := time.Now()
	resp, err := provider.Complete(context.Background(), req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "Success after retries" {
		t.Errorf("Expected 'Success after retries', got %s", resp.Content)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if duration < 2*time.Second {
		t.Errorf("Expected retry delays, but request completed too quickly: %v", duration)
	}
}

func TestProvider_TokenEstimation(t *testing.T) {
	config := &Config{
		APIKey:             testAPIKey,
		BaseURL:            DefaultBaseURL,
		DefaultModel:       "gpt-3.5-turbo",
		MaxTokens:          4096,
		DefaultTemperature: 0.7,
		Timeout:            30 * time.Second,
	}

	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	tests := []struct {
		text      string
		minTokens int
		maxTokens int
	}{
		{"hello", 1, 5},
		{"hello world", 2, 10},
		{"The quick brown fox jumps over the lazy dog", 5, 20},
		{"", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			tokens, err := provider.CountTokens(tt.text)
			if err != nil {
				t.Errorf("CountTokens() error = %v", err)
			}

			if tokens < tt.minTokens || tokens > tt.maxTokens {
				t.Errorf("CountTokens(%q) = %d, want between %d and %d", tt.text, tokens, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIKey:             testAPIKey,
				BaseURL:            "https://api.openai.com",
				DefaultModel:       "gpt-3.5-turbo",
				MaxTokens:          4096,
				DefaultTemperature: 0.7,
				Timeout:            30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				BaseURL:            "https://api.openai.com",
				DefaultModel:       "gpt-3.5-turbo",
				MaxTokens:          4096,
				DefaultTemperature: 0.7,
				Timeout:            30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid base URL",
			config: &Config{
				APIKey:             testAPIKey,
				BaseURL:            "http://[::1]:namedport",
				DefaultModel:       "gpt-3.5-turbo",
				MaxTokens:          4096,
				DefaultTemperature: 0.7,
				Timeout:            30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: &Config{
				APIKey:             testAPIKey,
				BaseURL:            "https://api.openai.com",
				DefaultModel:       "gpt-3.5-turbo",
				MaxTokens:          4096,
				DefaultTemperature: 3.0, // Invalid: > 2
				Timeout:            30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero max tokens",
			config: &Config{
				APIKey:             testAPIKey,
				BaseURL:            "https://api.openai.com",
				DefaultModel:       "gpt-3.5-turbo",
				MaxTokens:          0, // Invalid
				DefaultTemperature: 0.7,
				Timeout:            30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
