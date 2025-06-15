package correlation

import (
	"context"
	"fmt"
	"strings"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/docstore"
)

// Correlator connects error patterns with documentation
type Correlator interface {
	// Correlate finds documentation relevant to error patterns
	Correlate(ctx context.Context, analysis *analyzer.Analysis) (*CorrelationResult, error)

	// SetDocumentStore sets the document store for searching
	SetDocumentStore(store docstore.DocumentStore) error
}

// correlator implements the Correlator interface
type correlator struct {
	docStore  docstore.DocumentStore
	extractor KeywordExtractor
}

// NewCorrelator creates a new correlation engine
func NewCorrelator() Correlator {
	return &correlator{
		extractor: NewKeywordExtractor(),
	}
}

// SetDocumentStore sets the document store for searching
func (c *correlator) SetDocumentStore(store docstore.DocumentStore) error {
	if store == nil {
		return fmt.Errorf("document store cannot be nil")
	}
	c.docStore = store
	return nil
}

// Correlate finds documentation relevant to error patterns
func (c *correlator) Correlate(ctx context.Context, analysis *analyzer.Analysis) (*CorrelationResult, error) {
	if c.docStore == nil {
		return nil, fmt.Errorf("document store not configured")
	}

	if analysis == nil {
		return nil, fmt.Errorf("analysis cannot be nil")
	}

	result := &CorrelationResult{
		TotalPatterns: len(analysis.Patterns),
		Correlations:  make([]*PatternCorrelation, 0),
	}

	// Process each pattern match
	for _, patternMatch := range analysis.Patterns {
		correlation, err := c.correlatePattern(ctx, &patternMatch)
		if err != nil {
			continue // Skip failed correlations
		}

		if correlation != nil && len(correlation.DocumentMatches) > 0 {
			result.Correlations = append(result.Correlations, correlation)
		}
	}

	result.CorrelatedPatterns = len(result.Correlations)
	return result, nil
}

// correlatePattern correlates a single pattern with documentation
func (c *correlator) correlatePattern(ctx context.Context, patternMatch *analyzer.PatternMatch) (*PatternCorrelation, error) {
	// Extract keywords from the pattern and matches
	keywords := c.extractor.ExtractFromPattern(patternMatch.Pattern)

	// Add keywords from actual log entries
	for _, entry := range patternMatch.Matches {
		entryKeywords := c.extractor.ExtractFromLogEntry(entry)
		keywords = append(keywords, entryKeywords...)
	}

	// Remove duplicates and filter
	keywords = c.filterAndDeduplicateKeywords(keywords)

	if len(keywords) == 0 {
		return nil, fmt.Errorf("no keywords extracted from pattern")
	}

	// Search documents using extracted keywords
	searchResults, err := c.searchDocuments(ctx, keywords)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return &PatternCorrelation{
		Pattern:         patternMatch.Pattern,
		Keywords:        keywords,
		DocumentMatches: searchResults,
		MatchCount:      patternMatch.Count,
	}, nil
}

// searchDocuments searches for documents using keywords
func (c *correlator) searchDocuments(ctx context.Context, keywords []string) ([]*DocumentMatch, error) {
	var allResults []*docstore.SearchResult

	// Search for each keyword combination
	for _, keyword := range keywords {
		query := &docstore.SearchQuery{
			Text:      keyword,
			Highlight: true,
			Limit:     10,
		}

		results, err := c.docStore.Search(query)
		if err != nil {
			continue // Skip failed searches
		}

		allResults = append(allResults, results...)
	}

	// Convert to DocumentMatch and score
	return c.convertAndScoreResults(allResults, keywords)
}

// convertAndScoreResults converts search results to document matches
func (c *correlator) convertAndScoreResults(results []*docstore.SearchResult, keywords []string) ([]*DocumentMatch, error) {
	// Group results by document to avoid duplicates
	docMap := make(map[string]*DocumentMatch)

	for _, result := range results {
		docID := result.Document.ID

		if existing, exists := docMap[docID]; exists {
			// Update existing match
			existing.Score = maxFloat64(existing.Score, result.Score)
			existing.MatchedKeywords = append(existing.MatchedKeywords, extractMatchedKeywords(result, keywords)...)
		} else {
			// Create new match
			docMap[docID] = &DocumentMatch{
				Document:        result.Document,
				Score:           result.Score,
				MatchedKeywords: extractMatchedKeywords(result, keywords),
				Highlighted:     result.Highlighted,
			}
		}
	}

	// Convert map to slice and sort by score
	matches := make([]*DocumentMatch, 0, len(docMap))
	for _, match := range docMap {
		// Deduplicate keywords
		match.MatchedKeywords = deduplicateStrings(match.MatchedKeywords)
		matches = append(matches, match)
	}

	// Sort by score (highest first)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].Score < matches[j].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Return top 5 matches
	if len(matches) > 5 {
		matches = matches[:5]
	}

	return matches, nil
}

// extractMatchedKeywords extracts which keywords were matched in the result
func extractMatchedKeywords(result *docstore.SearchResult, keywords []string) []string {
	var matched []string
	content := strings.ToLower(result.Document.Content)

	for _, keyword := range keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			matched = append(matched, keyword)
		}
	}

	return matched
}

// filterAndDeduplicateKeywords filters and removes duplicate keywords
func (c *correlator) filterAndDeduplicateKeywords(keywords []string) []string {
	seen := make(map[string]bool)
	filtered := make([]string, 0, len(keywords))

	for _, keyword := range keywords {
		// Clean and normalize keyword
		cleaned := strings.TrimSpace(strings.ToLower(keyword))

		// Skip if empty, too short, or already seen
		if cleaned == "" || len(cleaned) < 3 || seen[cleaned] {
			continue
		}

		// Skip common words
		if isCommonWord(cleaned) {
			continue
		}

		seen[cleaned] = true
		filtered = append(filtered, keyword) // Keep original case
	}

	return filtered
}

// isCommonWord checks if a word is too common to be useful
func isCommonWord(word string) bool {
	commonWords := []string{
		"the", "and", "for", "are", "but", "not", "you", "all", "can", "had", "her", "was", "one", "our", "out", "day", "get", "has", "him", "his", "how", "man", "new", "now", "old", "see", "two", "way", "who", "boy", "did", "its", "let", "put", "say", "she", "too", "use",
		"error", "warning", "info", "debug", "log", "message", "text", "string", "value", "null", "true", "false",
	}

	for _, common := range commonWords {
		if word == common {
			return true
		}
	}

	return false
}

// Helper functions

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func deduplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
