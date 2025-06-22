package correlation

import (
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/docstore"
)

// CorrelationResult represents the result of correlating patterns and errors with documentation
type CorrelationResult struct {
	TotalPatterns      int                   `json:"total_patterns"`
	CorrelatedPatterns int                   `json:"correlated_patterns"`
	Correlations       []*PatternCorrelation `json:"correlations"`
	TotalErrors        int                   `json:"total_errors"`
	CorrelatedErrors   int                   `json:"correlated_errors"`
	DirectCorrelations []*ErrorCorrelation   `json:"direct_correlations"`
}

// PatternCorrelation represents the correlation between a pattern and documentation
type PatternCorrelation struct {
	Pattern         *common.Pattern  `json:"pattern"`
	Keywords        []string         `json:"keywords"`
	DocumentMatches []*DocumentMatch `json:"document_matches"`
	MatchCount      int              `json:"match_count"`
}

// ErrorCorrelation represents the correlation between a direct error and documentation
type ErrorCorrelation struct {
	Error           *common.LogEntry `json:"error"`
	ErrorType       string           `json:"error_type"`
	Keywords        []string         `json:"keywords"`
	DocumentMatches []*DocumentMatch `json:"document_matches"`
	MatchCount      int              `json:"match_count"`
	Confidence      float64          `json:"confidence"`
}

// DocumentMatch represents a document that matches pattern keywords
type DocumentMatch struct {
	Document        *docstore.Document `json:"document"`
	Score           float64            `json:"score"`
	MatchedKeywords []string           `json:"matched_keywords"`
	Highlighted     string             `json:"highlighted"`
	KeywordScore    float64            `json:"keyword_score"`
	VectorScore     float64            `json:"vector_score"`
	SearchMethod    string             `json:"search_method"` // "keyword", "vector", or "hybrid"
}

// KeywordExtractionResult holds extracted keywords with metadata
type KeywordExtractionResult struct {
	Keywords   []string `json:"keywords"`
	Source     string   `json:"source"`
	Confidence float64  `json:"confidence"`
}

// HybridSearchConfig configures the hybrid search behavior
type HybridSearchConfig struct {
	KeywordWeight  float64 `json:"keyword_weight"`   // Weight for keyword search results (0.0-1.0)
	VectorWeight   float64 `json:"vector_weight"`    // Weight for vector search results (0.0-1.0)
	MaxResults     int     `json:"max_results"`      // Maximum number of results to return
	VectorTopK     int     `json:"vector_top_k"`     // Number of top vector results to retrieve
	MinVectorScore float32 `json:"min_vector_score"` // Minimum vector similarity score
	EnableVector   bool    `json:"enable_vector"`    // Whether to use vector search
}

// DefaultHybridSearchConfig returns default configuration for hybrid search
func DefaultHybridSearchConfig() *HybridSearchConfig {
	return &HybridSearchConfig{
		KeywordWeight:  0.6,
		VectorWeight:   0.4,
		MaxResults:     5,
		VectorTopK:     10,
		MinVectorScore: 0.01,
		EnableVector:   true,
	}
}
