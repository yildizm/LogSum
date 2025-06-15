package ollama

import "time"

// GenerateRequest represents an Ollama generate API request
type GenerateRequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	System  string   `json:"system,omitempty"`
	Stream  bool     `json:"stream"`
	Options *Options `json:"options,omitempty"`
}

// GenerateResponse represents an Ollama generate API response
type GenerateResponse struct {
	Model     string    `json:"model"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	Context   []int     `json:"context,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Evaluation metrics
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

// EmbeddingsRequest represents an Ollama embeddings API request
type EmbeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingsResponse represents an Ollama embeddings API response
type EmbeddingsResponse struct {
	Embedding []float64 `json:"embedding"`
}

// TagsResponse represents the response from /api/tags
type TagsResponse struct {
	Models []Model `json:"models"`
}

// Model represents an Ollama model
type Model struct {
	Name       string        `json:"name"`
	ModifiedAt time.Time     `json:"modified_at"`
	Size       int64         `json:"size"`
	Digest     string        `json:"digest"`
	Details    *ModelDetails `json:"details,omitempty"`
}

// ModelDetails contains detailed model information
type ModelDetails struct {
	ParentModel       string   `json:"parent_model,omitempty"`
	Format            string   `json:"format,omitempty"`
	Family            string   `json:"family,omitempty"`
	Families          []string `json:"families,omitempty"`
	ParameterSize     string   `json:"parameter_size,omitempty"`
	QuantizationLevel string   `json:"quantization_level,omitempty"`
}

// PullRequest represents a model pull request
type PullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

// PullResponse represents a model pull response
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// Options contains generation options
type Options struct {
	Temperature   float64  `json:"temperature,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	NumCtx        int      `json:"num_ctx,omitempty"`     // Context window size
	NumPredict    int      `json:"num_predict,omitempty"` // Maximum tokens to generate
	RepeatPenalty float64  `json:"repeat_penalty,omitempty"`
	Seed          int      `json:"seed,omitempty"`
	Stop          []string `json:"stop,omitempty"`
}

// ErrorResponse represents an Ollama API error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthStatus represents the health status of Ollama
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
	Models    []string  `json:"models,omitempty"`
}
