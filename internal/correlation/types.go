package correlation

import (
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/docstore"
)

// CorrelationResult represents the result of correlating patterns with documentation
type CorrelationResult struct {
	TotalPatterns      int                   `json:"total_patterns"`
	CorrelatedPatterns int                   `json:"correlated_patterns"`
	Correlations       []*PatternCorrelation `json:"correlations"`
}

// PatternCorrelation represents the correlation between a pattern and documentation
type PatternCorrelation struct {
	Pattern         *common.Pattern  `json:"pattern"`
	Keywords        []string         `json:"keywords"`
	DocumentMatches []*DocumentMatch `json:"document_matches"`
	MatchCount      int              `json:"match_count"`
}

// DocumentMatch represents a document that matches pattern keywords
type DocumentMatch struct {
	Document        *docstore.Document `json:"document"`
	Score           float64            `json:"score"`
	MatchedKeywords []string           `json:"matched_keywords"`
	Highlighted     string             `json:"highlighted"`
}

// KeywordExtractionResult holds extracted keywords with metadata
type KeywordExtractionResult struct {
	Keywords   []string `json:"keywords"`
	Source     string   `json:"source"`
	Confidence float64  `json:"confidence"`
}
