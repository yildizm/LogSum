package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

// Provider implements the AI provider interface for Ollama
type Provider struct {
	config     *Config
	client     *http.Client
	baseURL    *url.URL
	healthy    bool
	healthMu   sync.RWMutex
	lastHealth time.Time
}

// New creates a new Ollama provider instance
func New(config *Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, ai.NewConfigurationError("ollama", "base_url", "invalid base URL: "+err.Error())
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	p := &Provider{
		config:  config,
		client:  client,
		baseURL: baseURL,
	}

	return p, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "ollama"
}

// Complete performs text completion
func (p *Provider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
	startTime := time.Now()

	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = p.config.DefaultTemperature
	}

	options := &Options{
		Temperature: temperature,
	}

	if req.MaxTokens > 0 {
		options.NumPredict = req.MaxTokens
	}

	ollamaReq := &GenerateRequest{
		Model:   model,
		Prompt:  req.Prompt,
		System:  req.SystemPrompt,
		Stream:  false,
		Options: options,
	}

	resp, err := p.generate(ctx, ollamaReq)
	if err != nil {
		return nil, err
	}

	usage := &ai.TokenUsage{
		PromptTokens:     resp.PromptEvalCount,
		CompletionTokens: resp.EvalCount,
		TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
	}

	return &ai.CompletionResponse{
		Content:      resp.Response,
		FinishReason: "stop",
		Usage:        usage,
		Model:        resp.Model,
		RequestID:    req.RequestID,
		CreatedAt:    startTime,
		Metadata: map[string]interface{}{
			"total_duration":       resp.TotalDuration,
			"load_duration":        resp.LoadDuration,
			"prompt_eval_duration": resp.PromptEvalDuration,
			"eval_duration":        resp.EvalDuration,
		},
	}, nil
}

// CompleteStream performs streaming text completion
func (p *Provider) CompleteStream(ctx context.Context, req *ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = p.config.DefaultTemperature
	}

	options := &Options{
		Temperature: temperature,
	}

	if req.MaxTokens > 0 {
		options.NumPredict = req.MaxTokens
	}

	ollamaReq := &GenerateRequest{
		Model:   model,
		Prompt:  req.Prompt,
		System:  req.SystemPrompt,
		Stream:  true,
		Options: options,
	}

	return p.generateStream(ctx, ollamaReq)
}

// CountTokens estimates token count for the given text
func (p *Provider) CountTokens(text string) (int, error) {
	// Simple token estimation (roughly 4 characters per token)
	return len(text) / 4, nil
}

// MaxTokens returns the maximum context window size
func (p *Provider) MaxTokens() int {
	return p.config.MaxTokens
}

// SupportsStreaming indicates if the provider supports streaming
func (p *Provider) SupportsStreaming() bool {
	return true
}

// ValidateConfig validates the provider configuration
func (p *Provider) ValidateConfig() error {
	return p.config.Validate()
}

// Close cleans up provider resources
func (p *Provider) Close() error {
	// No persistent connections to close for HTTP client
	return nil
}

// TruncateToFit truncates text to fit within token limits
func (p *Provider) TruncateToFit(text string, maxTokens int) (string, error) {
	estimatedTokens, err := p.CountTokens(text)
	if err != nil {
		return "", err
	}

	if estimatedTokens <= maxTokens {
		return text, nil
	}

	// Simple truncation based on character count
	ratio := float64(maxTokens) / float64(estimatedTokens)
	targetLength := int(float64(len(text)) * ratio)

	if targetLength >= len(text) {
		return text, nil
	}

	return text[:targetLength], nil
}

// SplitByTokens splits text into chunks by token count
func (p *Provider) SplitByTokens(text string, chunkSize int) ([]string, error) {
	estimatedTokens, err := p.CountTokens(text)
	if err != nil {
		return nil, err
	}

	if estimatedTokens <= chunkSize {
		return []string{text}, nil
	}

	// Calculate approximate character length per chunk
	charsPerToken := len(text) / estimatedTokens
	charsPerChunk := chunkSize * charsPerToken

	var chunks []string
	for i := 0; i < len(text); i += charsPerChunk {
		end := i + charsPerChunk
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}

	return chunks, nil
}

// EstimateTokens provides a rough token count estimate
func (p *Provider) EstimateTokens(text string) int {
	tokens, _ := p.CountTokens(text)
	return tokens
}

// HealthCheck verifies provider connectivity and status
func (p *Provider) HealthCheck(ctx context.Context) error {
	endpoint := p.baseURL.JoinPath("/api/tags")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), http.NoBody)
	if err != nil {
		return ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "failed to create health check request", "ollama", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.setHealthy(false)
		return ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "health check failed", "ollama", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		p.setHealthy(false)
		return ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("health check failed with status %d", resp.StatusCode), "ollama")
	}

	p.setHealthy(true)
	return nil
}

// IsHealthy returns current health status
func (p *Provider) IsHealthy() bool {
	p.healthMu.RLock()
	defer p.healthMu.RUnlock()
	return p.healthy
}

// generate performs a single generation request
func (p *Provider) generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	endpoint := p.baseURL.JoinPath("/api/generate")

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to marshal request", "ollama", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to create request", "ollama", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "request failed", "ollama", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp ErrorResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			return nil, ai.NewProviderError(ai.ErrTypeProvider, errorResp.Error, "ollama")
		}
		return nil, ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("request failed with status %d", resp.StatusCode), "ollama")
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to decode response", "ollama", err)
	}

	return &result, nil
}

// generateStream performs a streaming generation request
//
//nolint:gocyclo // Complex streaming logic is necessary
func (p *Provider) generateStream(ctx context.Context, req *GenerateRequest) (<-chan ai.StreamChunk, error) {
	endpoint := p.baseURL.JoinPath("/api/generate")

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to marshal request", "ollama", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to create request", "ollama", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "request failed", "ollama", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var errorResp ErrorResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			return nil, ai.NewProviderError(ai.ErrTypeProvider, errorResp.Error, "ollama")
		}
		return nil, ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("request failed with status %d", resp.StatusCode), "ollama")
	}

	ch := make(chan ai.StreamChunk)

	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var genResp GenerateResponse
			if err := json.Unmarshal([]byte(line), &genResp); err != nil {
				select {
				case ch <- ai.StreamChunk{Error: ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to decode stream response", "ollama", err)}:
				case <-ctx.Done():
				}
				return
			}

			select {
			case ch <- ai.StreamChunk{Content: genResp.Response, Done: genResp.Done}:
			case <-ctx.Done():
				return
			}

			if genResp.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case ch <- ai.StreamChunk{Error: ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "stream scanning error", "ollama", err)}:
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}

// setHealthy updates the health status
func (p *Provider) setHealthy(healthy bool) {
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	p.healthy = healthy
	p.lastHealth = time.Now()
}

// ListModels returns available models
func (p *Provider) ListModels(ctx context.Context) ([]Model, error) {
	endpoint := p.baseURL.JoinPath("/api/tags")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), http.NoBody)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "failed to create request", "ollama", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "request failed", "ollama", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("list models failed with status %d", resp.StatusCode), "ollama")
	}

	var tagsResp TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to decode response", "ollama", err)
	}

	return tagsResp.Models, nil
}

// PullModel downloads a model
func (p *Provider) PullModel(ctx context.Context, modelName string) error {
	endpoint := p.baseURL.JoinPath("/api/pull")

	pullReq := &PullRequest{
		Name:   modelName,
		Stream: false,
	}

	jsonData, err := json.Marshal(pullReq)
	if err != nil {
		return ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to marshal request", "ollama", err)
	}

	// Use pull timeout for this operation
	ctx, cancel := context.WithTimeout(ctx, p.config.PullTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to create request", "ollama", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "pull request failed", "ollama", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp ErrorResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			return ai.NewProviderError(ai.ErrTypeProvider, errorResp.Error, "ollama")
		}
		return ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("pull failed with status %d", resp.StatusCode), "ollama")
	}

	return nil
}

// IsModelAvailable checks if a model is available locally
func (p *Provider) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model.Name == modelName || strings.HasPrefix(model.Name, modelName+":") {
			return true, nil
		}
	}

	return false, nil
}
