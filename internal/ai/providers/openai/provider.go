package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

type Provider struct {
	config  *Config
	client  *http.Client
	baseURL *url.URL
	healthy bool
	mu      sync.RWMutex
}

func New(config *Config) (*Provider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, ai.NewConfigurationError("openai", "base_url", fmt.Sprintf("invalid base URL: %v", err))
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	p := &Provider{
		config:  config,
		client:  client,
		baseURL: baseURL,
		healthy: true,
	}

	return p, nil
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if req == nil {
		return nil, ai.NewValidationError("request", "nil", "completion request is required")
	}

	chatReq := p.buildChatRequest(req)
	chatReq.Stream = false

	response, err := p.sendChatRequest(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return response.ToAIResponse(req.RequestID), nil
}

func (p *Provider) CompleteStream(ctx context.Context, req *ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	if req == nil {
		return nil, ai.NewValidationError("request", "nil", "completion request is required")
	}

	chatReq := p.buildChatRequest(req)
	chatReq.Stream = true

	ch := make(chan ai.StreamChunk)

	go func() {
		defer close(ch)

		if err := p.sendChatRequestStream(ctx, chatReq, ch); err != nil {
			select {
			case ch <- ai.StreamChunk{Error: err}:
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}

func (p *Provider) CountTokens(text string) (int, error) {
	return p.estimateTokens(text), nil
}

func (p *Provider) MaxTokens() int {
	return p.config.MaxTokens
}

func (p *Provider) SupportsStreaming() bool {
	return true
}

func (p *Provider) ValidateConfig() error {
	return p.config.Validate()
}

func (p *Provider) Close() error {
	return nil
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	endpoint := p.baseURL.JoinPath("/v1/models")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), http.NoBody)
	if err != nil {
		p.setHealthy(false)
		return ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "failed to create health check request", "openai", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.OrganizationID != "" {
		req.Header.Set("OpenAI-Organization", p.config.OrganizationID)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.setHealthy(false)
		return ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "health check request failed", "openai", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		p.setHealthy(true)
		return nil
	}

	p.setHealthy(false)

	if resp.StatusCode == http.StatusUnauthorized {
		return ai.NewProviderError(ai.ErrTypeAuthentication, "invalid API key", "openai")
	}

	return ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("health check failed with status %d", resp.StatusCode), "openai")
}

func (p *Provider) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthy
}

func (p *Provider) TruncateToFit(text string, maxTokens int) (string, error) {
	tokens := p.estimateTokens(text)
	if tokens <= maxTokens {
		return text, nil
	}

	ratio := float64(maxTokens) / float64(tokens)
	targetLength := int(float64(len(text)) * ratio * 0.9)

	if targetLength >= len(text) {
		return text, nil
	}

	truncated := text[:targetLength]

	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		return truncated[:lastSpace], nil
	}

	return truncated, nil
}

func (p *Provider) SplitByTokens(text string, maxTokens int) ([]string, error) {
	if p.estimateTokens(text) <= maxTokens {
		return []string{text}, nil
	}

	var chunks []string
	words := strings.Fields(text)

	var currentChunk strings.Builder
	for _, word := range words {
		testChunk := currentChunk.String()
		if testChunk != "" {
			testChunk += " "
		}
		testChunk += word

		if p.estimateTokens(testChunk) > maxTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(word)
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(word)
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks, nil
}

func (p *Provider) EstimateTokens(text string) int {
	return p.estimateTokens(text)
}

func (p *Provider) buildChatRequest(req *ai.CompletionRequest) *ChatCompletionRequest {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = p.config.DefaultTemperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens / 2
	}

	chatReq := &ChatCompletionRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		User:        req.RequestID,
	}

	chatReq.ToMessages(req.SystemPrompt, req.Prompt, req.Context)

	return chatReq
}

func (p *Provider) sendChatRequest(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	endpoint := p.baseURL.JoinPath("/v1/chat/completions")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to marshal request", "openai", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "failed to create request", "openai", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.doRequestWithRetry(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp)
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to decode response", "openai", err)
	}

	return &chatResp, nil
}

func (p *Provider) sendChatRequestStream(ctx context.Context, req *ChatCompletionRequest, ch chan<- ai.StreamChunk) error {
	resp, err := p.createStreamRequest(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return p.handleErrorResponse(resp)
	}

	return p.processStreamResponse(ctx, resp, ch)
}

func (p *Provider) createStreamRequest(ctx context.Context, req *ChatCompletionRequest) (*http.Response, error) {
	endpoint := p.baseURL.JoinPath("/v1/chat/completions")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to marshal request", "openai", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "failed to create request", "openai", err)
	}

	p.setHeaders(httpReq)
	return p.doRequestWithRetry(httpReq)
}

func (p *Provider) processStreamResponse(ctx context.Context, resp *http.Response, ch chan<- ai.StreamChunk) error {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if line == "data: [DONE]" {
			return p.sendStreamChunk(ctx, ch, ai.StreamChunk{Done: true})
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		if err := p.processStreamLine(ctx, ch, line); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "error reading stream", "openai", err)
	}

	return nil
}

func (p *Provider) processStreamLine(ctx context.Context, ch chan<- ai.StreamChunk, line string) error {
	data := strings.TrimPrefix(line, "data: ")

	var streamResp ChatCompletionStreamResponse
	if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
		return nil // Skip malformed lines
	}

	if len(streamResp.Choices) == 0 {
		return nil
	}

	choice := streamResp.Choices[0]
	content := choice.Delta.Content
	done := choice.FinishReason != nil && *choice.FinishReason != ""

	if err := p.sendStreamChunk(ctx, ch, ai.StreamChunk{Content: content, Done: done}); err != nil {
		return err
	}

	if done {
		return io.EOF // Signal completion
	}

	return nil
}

func (p *Provider) sendStreamChunk(ctx context.Context, ch chan<- ai.StreamChunk, chunk ai.StreamChunk) error {
	select {
	case ch <- chunk:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	if p.config.OrganizationID != "" {
		req.Header.Set("OpenAI-Organization", p.config.OrganizationID)
	}
}

func (p *Provider) doRequestWithRetry(originalReq *http.Request) (*http.Response, error) {
	maxRetries := 3
	baseDelay := time.Second

	var body []byte
	if originalReq.Body != nil {
		var err error
		body, err = io.ReadAll(originalReq.Body)
		if err != nil {
			return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to read request body", "openai", err)
		}
		_ = originalReq.Body.Close()
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(originalReq.Context(), originalReq.Method, originalReq.URL.String(), reqBody)
		if err != nil {
			return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to create retry request", "openai", err)
		}

		for key, values := range originalReq.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := p.client.Do(req)
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "request failed after retries", "openai", err)
			}
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * baseDelay)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()

			if attempt == maxRetries-1 {
				return nil, ai.NewRateLimitError("openai", 0, "requests")
			}

			retryAfter := 1
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				if seconds, err := strconv.Atoi(retryHeader); err == nil {
					retryAfter = seconds
				}
			}

			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}

		return resp, nil
	}

	return nil, ai.NewProviderError(ai.ErrTypeNetwork, "max retries exceeded", "openai")
}

func (p *Provider) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("request failed with status %d", resp.StatusCode), "openai")
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return ai.NewProviderError(ai.ErrTypeProvider, fmt.Sprintf("request failed with status %d", resp.StatusCode), "openai")
	}

	message := errorResp.Error.Message
	if message == "" {
		message = fmt.Sprintf("request failed with status %d", resp.StatusCode)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ai.NewProviderError(ai.ErrTypeAuthentication, message, "openai")
	case http.StatusTooManyRequests:
		return ai.NewRateLimitError("openai", 0, "requests")
	case http.StatusBadRequest:
		return ai.NewValidationError("request", "invalid", message)
	default:
		return ai.NewProviderError(ai.ErrTypeProvider, message, "openai")
	}
}

func (p *Provider) setHealthy(healthy bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthy = healthy
}

func (p *Provider) GetModels() ([]ai.Model, error) {
	endpoint := p.baseURL.JoinPath("/v1/models")

	req, err := http.NewRequest("GET", endpoint.String(), http.NoBody)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "failed to create models request", "openai", err)
	}

	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeNetwork, "models request failed", "openai", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp)
	}

	var modelResp ModelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelResp); err != nil {
		return nil, ai.NewProviderErrorWithCause(ai.ErrTypeInternal, "failed to decode models response", "openai", err)
	}

	models := make([]ai.Model, 0, len(modelResp.Data))
	for _, model := range modelResp.Data {
		models = append(models, ai.Model{
			ID:          model.ID,
			Name:        model.ID,
			Description: fmt.Sprintf("OpenAI model %s owned by %s", model.ID, model.OwnedBy),
			Provider:    "openai",
			MaxTokens:   p.getModelMaxTokens(model.ID),
			CreatedAt:   time.Unix(model.Created, 0),
			OwnedBy:     model.OwnedBy,
		})
	}

	return models, nil
}

func (p *Provider) getModelMaxTokens(modelID string) int {
	switch {
	case strings.Contains(modelID, "gpt-4"):
		if strings.Contains(modelID, "32k") {
			return 32768
		}
		return 8192
	case strings.Contains(modelID, "gpt-3.5-turbo"):
		if strings.Contains(modelID, "16k") {
			return 16384
		}
		return 4096
	case strings.Contains(modelID, "text-davinci"):
		return 4097
	case strings.Contains(modelID, "text-curie"):
		return 2049
	case strings.Contains(modelID, "text-babbage"):
		return 2049
	case strings.Contains(modelID, "text-ada"):
		return 2049
	default:
		return p.config.MaxTokens
	}
}

func (p *Provider) estimateTokens(text string) int {
	return len(strings.Fields(text)) + len(text)/4
}
