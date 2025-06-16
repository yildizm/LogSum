package correlation

import (
	"context"
	"fmt"
	"strings"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/LogSum/internal/vectorstore"
)

// Correlator connects error patterns with documentation
type Correlator interface {
	// Correlate finds documentation relevant to error patterns
	Correlate(ctx context.Context, analysis *common.Analysis) (*CorrelationResult, error)

	// SetDocumentStore sets the document store for searching
	SetDocumentStore(store docstore.DocumentStore) error

	// SetVectorStore sets the vector store for semantic search (optional)
	SetVectorStore(store vectorstore.VectorStore, vectorizer vectorstore.Vectorizer) error

	// SetHybridSearchConfig configures hybrid search behavior
	SetHybridSearchConfig(config *HybridSearchConfig) error

	// IndexDocuments indexes documents for both keyword and vector search
	IndexDocuments(ctx context.Context) error
}

// correlator implements the Correlator interface
type correlator struct {
	docStore    docstore.DocumentStore
	vectorStore vectorstore.VectorStore
	vectorizer  vectorstore.Vectorizer
	extractor   KeywordExtractor
	config      *HybridSearchConfig
}

// NewCorrelator creates a new correlation engine
func NewCorrelator() Correlator {
	return &correlator{
		extractor: NewKeywordExtractor(),
		config:    DefaultHybridSearchConfig(),
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

// SetVectorStore sets the vector store for semantic search (optional)
func (c *correlator) SetVectorStore(store vectorstore.VectorStore, vectorizer vectorstore.Vectorizer) error {
	if store == nil {
		return fmt.Errorf("vector store cannot be nil")
	}
	if vectorizer == nil {
		return fmt.Errorf("vectorizer cannot be nil")
	}
	c.vectorStore = store
	c.vectorizer = vectorizer
	return nil
}

// SetHybridSearchConfig configures hybrid search behavior
func (c *correlator) SetHybridSearchConfig(config *HybridSearchConfig) error {
	if config == nil {
		return fmt.Errorf("hybrid search config cannot be nil")
	}
	// Validate weights sum to 1.0 (approximately)
	totalWeight := config.KeywordWeight + config.VectorWeight
	if totalWeight < 0.9 || totalWeight > 1.1 {
		return fmt.Errorf("keyword and vector weights should sum to approximately 1.0, got %.2f", totalWeight)
	}
	c.config = config
	return nil
}

// IndexDocuments indexes documents for both keyword and vector search
func (c *correlator) IndexDocuments(ctx context.Context) error {
	if c.docStore == nil {
		return fmt.Errorf("document store not configured")
	}

	// Vector indexing is optional
	if c.vectorStore != nil && c.vectorizer != nil && c.config.EnableVector {
		return c.indexDocumentsWithVectors(ctx)
	}

	return nil // Keywords are indexed automatically by docstore
}

// indexDocumentsWithVectors indexes all documents into the vector store
func (c *correlator) indexDocumentsWithVectors(ctx context.Context) error {
	// Get all documents from the document store
	allDocs, err := c.getAllDocuments(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve documents: %w", err)
	}

	if len(allDocs) == 0 {
		return nil // No documents to index
	}

	// Prepare documents for vectorization
	documents := make([]string, len(allDocs))
	for i, doc := range allDocs {
		documents[i] = doc.Content
	}

	// Train vectorizer if it's a TF-IDF vectorizer
	if tfidfVectorizer, ok := c.vectorizer.(*vectorstore.TFIDFVectorizer); ok {
		if !tfidfVectorizer.IsFitted() {
			if err := tfidfVectorizer.Fit(documents); err != nil {
				return fmt.Errorf("failed to train vectorizer: %w", err)
			}
		}
	}

	// Index each document
	for _, doc := range allDocs {
		vector, err := c.vectorizer.Vectorize(doc.Content)
		if err != nil {
			continue // Skip failed vectorizations
		}

		if err := c.vectorStore.Store(doc.ID, doc.Content, vector); err != nil {
			continue // Skip failed storage
		}
	}

	return nil
}

// getAllDocuments retrieves all documents from the document store
func (c *correlator) getAllDocuments(ctx context.Context) ([]*docstore.Document, error) {
	// Use List method to get all documents with empty filter
	filter := &docstore.FilterOptions{} // Empty filter to get all documents
	docs, err := c.docStore.List(filter)
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// Correlate finds documentation relevant to error patterns
func (c *correlator) Correlate(ctx context.Context, analysis *common.Analysis) (*CorrelationResult, error) {
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
func (c *correlator) correlatePattern(ctx context.Context, patternMatch *common.PatternMatch) (*PatternCorrelation, error) {
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

	// Search documents using hybrid approach (keywords + vectors)
	searchResults, err := c.searchDocuments(ctx, keywords, patternMatch)
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

// searchDocuments searches for documents using hybrid approach (keywords + vectors)
func (c *correlator) searchDocuments(ctx context.Context, keywords []string, patternMatch *common.PatternMatch) ([]*DocumentMatch, error) {
	var keywordResults []*DocumentMatch
	var vectorResults []*DocumentMatch

	// 1. Keyword search (always performed)
	keywordResults, err := c.performKeywordSearch(ctx, keywords)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// 2. Vector search (optional)
	if c.vectorStore != nil && c.vectorizer != nil && c.config.EnableVector {
		vectorResults, err = c.performVectorSearch(ctx, patternMatch)
		if err != nil {
			// Vector search failure shouldn't block keyword results
			vectorResults = []*DocumentMatch{}
		}
	}

	// 3. Merge and rank results
	return c.mergeAndRankResults(keywordResults, vectorResults)
}

// performKeywordSearch performs traditional keyword-based search
func (c *correlator) performKeywordSearch(ctx context.Context, keywords []string) ([]*DocumentMatch, error) {
	var allResults []*docstore.SearchResult

	// Search for each keyword
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

	// Convert to DocumentMatch
	return c.convertKeywordResults(allResults, keywords)
}

// performVectorSearch performs semantic vector search
func (c *correlator) performVectorSearch(ctx context.Context, patternMatch *common.PatternMatch) ([]*DocumentMatch, error) {
	// Create search query from pattern and log entries
	searchText := c.buildVectorSearchQuery(patternMatch)

	// Vectorize the search query
	queryVector, err := c.vectorizer.Vectorize(searchText)
	if err != nil {
		return nil, fmt.Errorf("failed to vectorize search query: %w", err)
	}

	// Search vector store
	vectorResults, err := c.vectorStore.Search(queryVector, c.config.VectorTopK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert to DocumentMatch
	return c.convertVectorResults(vectorResults)
}

// buildVectorSearchQuery creates a search query from pattern matches
func (c *correlator) buildVectorSearchQuery(patternMatch *common.PatternMatch) string {
	var queryParts []string

	// Add pattern description if available
	if patternMatch.Pattern != nil && patternMatch.Pattern.Description != "" {
		queryParts = append(queryParts, patternMatch.Pattern.Description)
	}

	// Add sample log entries (limit to avoid too long queries)
	maxSamples := 3
	for i, entry := range patternMatch.Matches {
		if i >= maxSamples {
			break
		}
		if entry.Message != "" {
			queryParts = append(queryParts, entry.Message)
		}
	}

	return strings.Join(queryParts, " ")
}

// convertKeywordResults converts keyword search results to DocumentMatch
func (c *correlator) convertKeywordResults(results []*docstore.SearchResult, keywords []string) ([]*DocumentMatch, error) {
	// Group results by document to avoid duplicates
	docMap := make(map[string]*DocumentMatch)

	for _, result := range results {
		docID := result.Document.ID

		if existing, exists := docMap[docID]; exists {
			// Update existing match
			keywordScore := maxFloat64(existing.KeywordScore, result.Score)
			existing.KeywordScore = keywordScore
			existing.Score = keywordScore // Will be recalculated in merge step
			existing.MatchedKeywords = append(existing.MatchedKeywords, extractMatchedKeywords(result, keywords)...)
		} else {
			// Create new match
			docMap[docID] = &DocumentMatch{
				Document:        result.Document,
				Score:           result.Score,
				KeywordScore:    result.Score,
				VectorScore:     0.0,
				MatchedKeywords: extractMatchedKeywords(result, keywords),
				Highlighted:     result.Highlighted,
				SearchMethod:    "keyword",
			}
		}
	}

	// Convert map to slice
	matches := make([]*DocumentMatch, 0, len(docMap))
	for _, match := range docMap {
		// Deduplicate keywords
		match.MatchedKeywords = deduplicateStrings(match.MatchedKeywords)
		matches = append(matches, match)
	}

	return matches, nil
}

// convertVectorResults converts vector search results to DocumentMatch
func (c *correlator) convertVectorResults(results []vectorstore.SearchResult) ([]*DocumentMatch, error) {
	matches := make([]*DocumentMatch, 0, len(results))

	for _, result := range results {
		// Filter by minimum score
		if result.Score < c.config.MinVectorScore {
			continue
		}

		// Create document from vector result (simplified - would need to fetch full doc)
		doc := &docstore.Document{
			ID:      result.ID,
			Content: result.Text,
		}

		match := &DocumentMatch{
			Document:        doc,
			Score:           float64(result.Score),
			KeywordScore:    0.0,
			VectorScore:     float64(result.Score),
			MatchedKeywords: []string{}, // Vector search doesn't provide specific keywords
			Highlighted:     "",         // No highlighting for vector results
			SearchMethod:    "vector",
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// mergeAndRankResults combines keyword and vector results using hybrid scoring
func (c *correlator) mergeAndRankResults(keywordResults, vectorResults []*DocumentMatch) ([]*DocumentMatch, error) {
	// Create a map to merge results by document ID
	docMap := make(map[string]*DocumentMatch)

	// Add keyword results
	for _, match := range keywordResults {
		docMap[match.Document.ID] = match
	}

	// Merge vector results
	for _, vectorMatch := range vectorResults {
		docID := vectorMatch.Document.ID

		if existing, exists := docMap[docID]; exists {
			// Document found in both searches - combine scores
			existing.VectorScore = vectorMatch.VectorScore
			existing.SearchMethod = "hybrid"
		} else {
			// Document only found in vector search
			docMap[docID] = vectorMatch
		}
	}

	// Calculate hybrid scores and convert to slice
	matches := make([]*DocumentMatch, 0, len(docMap))
	for _, match := range docMap {
		// Calculate hybrid score
		match.Score = (c.config.KeywordWeight * match.KeywordScore) +
			(c.config.VectorWeight * match.VectorScore)
		matches = append(matches, match)
	}

	// Sort by hybrid score (highest first)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].Score < matches[j].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Return top results based on configuration
	if len(matches) > c.config.MaxResults {
		matches = matches[:c.config.MaxResults]
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
