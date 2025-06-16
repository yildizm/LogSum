package openai

import (
	"time"

	"github.com/yildizm/LogSum/internal/ai"
)

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	User        string        `json:"user,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   ChatCompletionUsage    `json:"usage"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionStreamResponse struct {
	ID      string                       `json:"id"`
	Object  string                       `json:"object"`
	Created int64                        `json:"created"`
	Model   string                       `json:"model"`
	Choices []ChatCompletionStreamChoice `json:"choices"`
}

type ChatCompletionStreamChoice struct {
	Index        int                 `json:"index"`
	Delta        ChatCompletionDelta `json:"delta"`
	FinishReason *string             `json:"finish_reason"`
}

type ChatCompletionDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type ModelListResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

func (r *ChatCompletionRequest) ToMessages(systemPrompt, prompt, context string) {
	r.Messages = []ChatMessage{}

	if systemPrompt != "" {
		r.Messages = append(r.Messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	if context != "" {
		r.Messages = append(r.Messages, ChatMessage{
			Role:    "user",
			Content: "Context: " + context,
		})
	}

	r.Messages = append(r.Messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})
}

func (r *ChatCompletionResponse) ToAIResponse(requestID string) *ai.CompletionResponse {
	response := &ai.CompletionResponse{
		RequestID: requestID,
		Model:     r.Model,
		CreatedAt: time.Unix(r.Created, 0),
		Usage: &ai.TokenUsage{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		},
	}

	if len(r.Choices) > 0 {
		choice := r.Choices[0]
		response.Content = choice.Message.Content
		response.FinishReason = choice.FinishReason
	}

	return response
}
