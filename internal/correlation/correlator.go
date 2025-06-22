package correlation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/LogSum/internal/logger"
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
	logger      *logger.Logger
}

// NewCorrelator creates a new correlation engine
func NewCorrelator() Correlator {
	return &correlator{
		extractor: NewKeywordExtractor(),
		config:    DefaultHybridSearchConfig(),
		logger:    logger.NewWithCallback("correlator", func() bool { return false }), // Default no verbose
	}
}

// NewCorrelatorWithLogger creates a new correlation engine with logger
func NewCorrelatorWithLogger(log *logger.Logger) Correlator {
	return &correlator{
		extractor: NewKeywordExtractor(),
		config:    DefaultHybridSearchConfig(),
		logger:    log.WithComponent("correlator"),
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

// Correlate finds documentation relevant to error patterns and direct errors
func (c *correlator) Correlate(ctx context.Context, analysis *common.Analysis) (*CorrelationResult, error) {
	startTime := time.Now()

	if analysis == nil {
		c.logger.Error("analysis cannot be nil")
		return nil, fmt.Errorf("analysis cannot be nil")
	}

	c.logger.InfoWithFields("starting correlation analysis", []logger.Field{
		logger.F("patterns", len(analysis.Patterns)),
		logger.F("raw_entries", len(analysis.RawEntries)),
	})

	if c.docStore == nil {
		c.logger.Error("document store not configured")
		return nil, fmt.Errorf("document store not configured")
	}

	result := &CorrelationResult{
		TotalPatterns:      len(analysis.Patterns),
		Correlations:       make([]*PatternCorrelation, 0),
		TotalErrors:        0, // Will be calculated when we extract error entries
		DirectCorrelations: make([]*ErrorCorrelation, 0),
	}

	c.logger.DebugWithFields("initialized correlation result", []logger.Field{
		logger.F("patterns", len(analysis.Patterns)),
		logger.F("raw_entries", len(analysis.RawEntries)),
	})

	// Process pattern-based correlations (existing logic)
	c.logger.Debug("processing pattern-based correlations (%d patterns)", len(analysis.Patterns))
	for i, patternMatch := range analysis.Patterns {
		c.logger.Debug("correlating pattern %d: %s", i+1, patternMatch.Pattern.Name)

		correlation, err := c.correlatePattern(ctx, &patternMatch)
		if err != nil {
			c.logger.Debug("failed to correlate pattern %s: %v", patternMatch.Pattern.Name, err)
			continue // Skip failed correlations
		}

		if correlation != nil && len(correlation.DocumentMatches) > 0 {
			c.logger.DebugWithFields("pattern correlation successful", []logger.Field{
				logger.F("pattern", patternMatch.Pattern.Name),
				logger.F("matches", len(correlation.DocumentMatches)),
			})
			result.Correlations = append(result.Correlations, correlation)
		} else {
			c.logger.Debug("no documents matched pattern: %s", patternMatch.Pattern.Name)
		}
	}

	result.CorrelatedPatterns = len(result.Correlations)
	c.logger.InfoWithFields("pattern correlation completed", []logger.Field{
		logger.F("total_patterns", len(analysis.Patterns)),
		logger.F("correlated_patterns", result.CorrelatedPatterns),
	})

	// Process direct error correlations (new logic)
	c.logger.Debug("extracting error entries from raw logs")
	var errorEntries []*common.LogEntry
	if analysis.RawEntries != nil {
		for _, entry := range analysis.RawEntries {
			// Only include entries that appear to be errors
			if c.isErrorEntry(entry) {
				errorEntries = append(errorEntries, entry)
			}
		}
	}

	result.TotalErrors = len(errorEntries)
	c.logger.DebugWithFields("error extraction completed", []logger.Field{
		logger.F("raw_entries", len(analysis.RawEntries)),
		logger.F("error_entries", len(errorEntries)),
	})

	// Perform direct error correlation
	switch {
	case len(errorEntries) == 0:
		c.logger.Debug("no error entries found for direct correlation")
	default:
		c.logger.Info("starting direct error correlation (%d errors)", len(errorEntries))
		directCorrelations, err := c.CorrelateDirectErrors(ctx, errorEntries)
		switch {
		case err == nil && len(directCorrelations) > 0:
			result.DirectCorrelations = directCorrelations
			result.CorrelatedErrors = len(directCorrelations)
			c.logger.InfoWithFields("direct error correlation successful", []logger.Field{
				logger.F("correlated_errors", len(directCorrelations)),
			})
		case err != nil:
			c.logger.Debug("direct error correlation failed: %v", err)
		default:
			c.logger.Debug("no direct error correlations found")
		}
	}

	duration := time.Since(startTime)
	c.logger.InfoWithFields("correlation analysis completed", []logger.Field{
		logger.Duration(duration),
		logger.F("total_patterns", result.TotalPatterns),
		logger.F("correlated_patterns", result.CorrelatedPatterns),
		logger.F("total_errors", result.TotalErrors),
		logger.F("correlated_errors", result.CorrelatedErrors),
	})

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

	// Handle nil pattern match (used for direct error correlation)
	if patternMatch == nil {
		return ""
	}

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

// CorrelateDirectErrors performs correlation on raw log entries without requiring patterns
func (c *correlator) CorrelateDirectErrors(ctx context.Context, entries []*common.LogEntry) ([]*ErrorCorrelation, error) {
	if c.docStore == nil {
		return nil, fmt.Errorf("document store not configured")
	}

	var correlations []*ErrorCorrelation

	// Group similar errors to avoid duplicate correlations
	errorGroups := c.groupSimilarErrors(entries)

	for errorType, errorGroup := range errorGroups {
		// Extract keywords from the representative error
		representative := errorGroup[0]
		keywords := c.extractor.ExtractFromLogEntry(representative)

		// Filter and clean keywords
		keywords = c.filterAndDeduplicateKeywords(keywords)

		if len(keywords) == 0 {
			continue // Skip if no meaningful keywords extracted
		}

		// Search documents using the same hybrid approach as patterns
		searchResults, err := c.searchDocuments(ctx, keywords, nil)
		if err != nil {
			continue // Skip on error, don't fail entire correlation
		}

		if len(searchResults) > 0 {
			correlation := &ErrorCorrelation{
				Error:           representative,
				ErrorType:       errorType,
				Keywords:        keywords,
				DocumentMatches: searchResults,
				MatchCount:      len(errorGroup),
				Confidence:      c.calculateErrorConfidence(keywords, searchResults),
			}

			correlations = append(correlations, correlation)
		}
	}

	return correlations, nil
}

// groupSimilarErrors groups log entries by error signature for intelligent deduplication
func (c *correlator) groupSimilarErrors(entries []*common.LogEntry) map[string][]*common.LogEntry {
	groups := make(map[string][]*common.LogEntry)

	for _, entry := range entries {
		signature := c.generateErrorSignature(entry)
		groups[signature] = append(groups[signature], entry)
	}

	return groups
}

// generateErrorSignature creates a unique signature for error grouping based on multiple factors
func (c *correlator) generateErrorSignature(entry *common.LogEntry) string {
	var signatureParts []string

	// 1. Error type (primary classifier)
	errorType := c.extractErrorType(entry)
	signatureParts = append(signatureParts, errorType)

	// 2. Normalize the error message by removing variable data
	normalizedMessage := c.normalizeErrorMessage(entry.Message)
	signatureParts = append(signatureParts, normalizedMessage)

	// 3. Add context-sensitive information
	if entry.Source != "" {
		// Include source/component if available
		signatureParts = append(signatureParts, "src:"+entry.Source)
	}

	if entry.Service != "" {
		// Include service if available
		signatureParts = append(signatureParts, "service:"+entry.Service)
	}

	if entry.TraceID != "" {
		// Include trace context (but not the actual ID as it's too specific)
		signatureParts = append(signatureParts, "traced")
	}

	// 4. Include log level for additional context
	signatureParts = append(signatureParts, "level:"+entry.LogLevel.String())

	return strings.Join(signatureParts, "|")
}

// normalizeErrorMessage removes variable data to enable better grouping
func (c *correlator) normalizeErrorMessage(message string) string {
	normalized := message

	// Define patterns to normalize (replace with placeholders)
	normalizationPatterns := []struct {
		pattern     string
		placeholder string
	}{
		// IDs and numeric values
		{`\b\d{4,}\b`, "<ID>"}, // Long numbers (IDs, timestamps)
		{`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`, "<UUID>"}, // UUIDs
		{`\b[0-9a-fA-F]{32}\b`, "<HASH>"}, // MD5 hashes
		{`\b[0-9a-fA-F]{40}\b`, "<HASH>"}, // SHA1 hashes
		{`\b[0-9a-fA-F]{64}\b`, "<HASH>"}, // SHA256 hashes

		// File paths and URLs
		{`/[\w\-/\.]+/[\w\-\.]+`, "<PATH>"}, // File paths
		{`https?://[^\s]+`, "<URL>"},        // URLs
		{`file://[^\s]+`, "<FILE_URL>"},     // File URLs

		// Network addresses
		{`\b(?:\d{1,3}\.){3}\d{1,3}:\d+\b`, "<IP:PORT>"}, // IP:port
		{`\b(?:\d{1,3}\.){3}\d{1,3}\b`, "<IP>"},          // IP addresses
		{`localhost:\d+`, "<LOCALHOST>"},                 // localhost with port

		// Timestamps and durations
		{`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`, "<TIMESTAMP>"}, // ISO timestamps
		{`\d+ms\b`, "<DURATION>"},                                 // Milliseconds
		{`\d+s\b`, "<DURATION>"},                                  // Seconds
		{`\d+\.\d+s\b`, "<DURATION>"},                             // Float seconds

		// Database specific
		{`'[^']*'`, "<STRING>"}, // Single quoted strings (SQL values)
		{`"[^"]*"`, "<STRING>"}, // Double quoted strings
		{`=\s*\w+`, "=<VALUE>"}, // Assignment values

		// Memory addresses and references
		{`0x[0-9a-fA-F]+`, "<MEMADDR>"}, // Memory addresses
		{`@[0-9a-fA-F]+`, "<OBJREF>"},   // Object references

		// User/session specific
		{`user_id[=:]\w+`, "user_id=<ID>"}, // User IDs
		{`session[=:]\w+`, "session=<ID>"}, // Session IDs
		{`token[=:]\w+`, "token=<TOKEN>"},  // Tokens

		// Business domain specific (terms example)
		{`promo_id[=:]['"]?[A-Z0-9]+['"]?`, "promo_id=<PROMO_ID>"},      // Promo IDs like SUMMER2024
		{`campaign[=:]['"]?[A-Za-z0-9_-]+['"]?`, "campaign=<CAMPAIGN>"}, // Campaign names

		// Generic parameter patterns
		{`\w+=[^\s,)]+`, "<PARAM>"}, // Generic key=value pairs
	}

	// Apply normalizations
	for _, np := range normalizationPatterns {
		re := regexp.MustCompile(np.pattern)
		normalized = re.ReplaceAllString(normalized, np.placeholder)
	}

	// Additional cleanup
	// Remove excessive whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Trim and limit length to prevent extremely long signatures
	normalized = strings.TrimSpace(normalized)
	if len(normalized) > 200 {
		normalized = normalized[:200] + "..."
	}

	return normalized
}

// extractErrorType attempts to extract the error type from a log entry using enhanced classification
func (c *correlator) extractErrorType(entry *common.LogEntry) string {
	message := entry.Message

	// Try classification methods in priority order
	if errorType := c.extractExceptionPatterns(message); errorType != "" {
		return errorType
	}

	if errorType := c.extractDomainPatterns(message); errorType != "" {
		return errorType
	}

	if errorType := c.extractHTTPPatterns(message); errorType != "" {
		return errorType
	}

	if errorType := c.extractLogLevelError(entry.LogLevel); errorType != "" {
		return errorType
	}

	if errorType := c.extractKeywordPatterns(message); errorType != "" {
		return errorType
	}

	return c.extractFallbackError(message)
}

// extractExceptionPatterns looks for specific exception/error class patterns (highest priority)
func (c *correlator) extractExceptionPatterns(message string) string {
	patterns := []struct {
		pattern  string
		category string
	}{
		{`(\w+Exception)`, "exception"},
		{`(\w+Error)`, "error"},
		{`(\w+Fault)`, "fault"},
		{`(\w+Timeout)`, "timeout"},
		{`(\w+Failure)`, "failure"},
		{`ERROR:\s*(\w+)`, "error"},
		{`FATAL:\s*(\w+)`, "fatal"},
		{`PANIC:\s*(\w+)`, "panic"},
	}

	for _, ep := range patterns {
		if match := regexp.MustCompile(ep.pattern).FindStringSubmatch(message); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// extractDomainPatterns performs business domain error classification
func (c *correlator) extractDomainPatterns(message string) string {
	patterns := []struct {
		pattern   string
		errorType string
	}{
		{`(?i)(database|db|sql|connection|query).*?(error|fail|timeout)`, "DatabaseError"},
		{`(?i)(auth|login|credential|permission|access).*?(error|fail|denied)`, "AuthenticationError"},
		{`(?i)(network|connection|socket|http|api).*?(error|fail|timeout)`, "NetworkError"},
		{`(?i)(file|disk|storage|io).*?(error|fail|full|not found)`, "IOError"},
		{`(?i)(memory|heap|stack|oom).*?(error|fail|overflow)`, "MemoryError"},
		{`(?i)(validation|invalid|bad|malformed).*?(data|input|format)`, "ValidationError"},
		{`(?i)(timeout|deadline|expired)`, "TimeoutError"},
		{`(?i)(not found|missing|absent|404)`, "NotFoundError"},
		{`(?i)(conflict|duplicate|already exists|409)`, "ConflictError"},
		{`(?i)(rate.?limit|throttle|too many)`, "RateLimitError"},
		{`(?i)(config|configuration|setting).*?(error|invalid|missing)`, "ConfigurationError"},
		{`(?i)(term|promo|promotion|campaign).*?(not found|missing|invalid)`, "TermsError"},
	}

	for _, dp := range patterns {
		if matched, _ := regexp.MatchString(dp.pattern, message); matched {
			return dp.errorType
		}
	}
	return ""
}

// extractHTTPPatterns classifies HTTP status code errors
func (c *correlator) extractHTTPPatterns(message string) string {
	patterns := []struct {
		pattern   string
		errorType string
	}{
		{`\b(400|Bad Request)\b`, "BadRequestError"},
		{`\b(401|Unauthorized)\b`, "UnauthorizedError"},
		{`\b(403|Forbidden)\b`, "ForbiddenError"},
		{`\b(404|Not Found)\b`, "NotFoundError"},
		{`\b(409|Conflict)\b`, "ConflictError"},
		{`\b(429|Too Many Requests)\b`, "RateLimitError"},
		{`\b(500|Internal Server Error)\b`, "InternalServerError"},
		{`\b(502|Bad Gateway)\b`, "BadGatewayError"},
		{`\b(503|Service Unavailable)\b`, "ServiceUnavailableError"},
		{`\b(504|Gateway Timeout)\b`, "GatewayTimeoutError"},
	}

	for _, hp := range patterns {
		if matched, _ := regexp.MatchString(hp.pattern, message); matched {
			return hp.errorType
		}
	}
	return ""
}

// extractLogLevelError performs log level based classification
func (c *correlator) extractLogLevelError(level common.LogLevel) string {
	switch level {
	case common.LevelFatal:
		return "FatalError"
	case common.LevelError:
		return "ApplicationError"
	default:
		return ""
	}
}

// extractKeywordPatterns performs keyword-based classification (lower priority)
func (c *correlator) extractKeywordPatterns(message string) string {
	patterns := []struct {
		keywords  []string
		errorType string
	}{
		{[]string{"connection", "refused", "timeout"}, "ConnectionError"},
		{[]string{"permission", "denied", "access"}, "PermissionError"},
		{[]string{"parse", "parsing", "syntax"}, "ParseError"},
		{[]string{"serialize", "deserialize", "marshal"}, "SerializationError"},
		{[]string{"lock", "deadlock", "blocked"}, "ConcurrencyError"},
		{[]string{"queue", "buffer", "overflow"}, "BufferError"},
		{[]string{"certificate", "ssl", "tls"}, "SecurityError"},
	}

	messageLower := strings.ToLower(message)
	for _, kp := range patterns {
		matchCount := 0
		for _, keyword := range kp.keywords {
			if strings.Contains(messageLower, keyword) {
				matchCount++
			}
		}
		// Require at least 2 keyword matches for classification
		if matchCount >= 2 {
			return kp.errorType
		}
	}
	return ""
}

// extractFallbackError extracts meaningful prefix from message as fallback
func (c *correlator) extractFallbackError(message string) string {
	words := strings.Fields(message)
	if len(words) >= 2 {
		// Try to identify if first words contain error context
		firstTwo := strings.Join(words[:2], "")
		// Remove special characters and check if it looks like an error type
		cleaned := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(firstTwo, "")
		if len(cleaned) >= 5 && len(cleaned) <= 30 {
			return cleaned + "Error"
		}
		return strings.Join(words[:2], " ")
	} else if len(words) == 1 {
		return words[0] + "Error"
	}

	return "UnknownError"
}

// calculateErrorConfidence calculates confidence score for error correlation using enhanced scoring
func (c *correlator) calculateErrorConfidence(keywords []string, matches []*DocumentMatch) float64 {
	if len(matches) == 0 {
		return 0.0
	}

	if len(keywords) == 0 {
		return 0.0
	}

	// Enhanced confidence calculation with multiple factors
	var weightedConfidence float64
	totalWeight := 0.0

	for i, match := range matches {
		// 1. Keyword coverage score (how many keywords are matched)
		keywordCoverage := float64(len(match.MatchedKeywords)) / float64(len(keywords))

		// 2. Document relevance score (from search)
		documentScore := match.Score

		// 3. Position weight (earlier results are more relevant)
		positionWeight := 1.0 / (1.0 + float64(i)*0.1)

		// 4. Keyword density bonus (high-quality matches have more keywords)
		keywordDensity := float64(len(match.MatchedKeywords)) / 10.0 // Normalize to 0-1 range
		if keywordDensity > 1.0 {
			keywordDensity = 1.0
		}

		// 5. Search method bonus (hybrid searches are more reliable)
		methodBonus := 1.0
		switch match.SearchMethod {
		case "hybrid":
			methodBonus = 1.2
		case "vector":
			methodBonus = 1.1
		}

		// Combine factors with weights
		matchConfidence := (keywordCoverage * 0.4) + // 40% weight on keyword coverage
			(documentScore * 0.3) + // 30% weight on document relevance
			(keywordDensity * 0.2) + // 20% weight on keyword density
			(positionWeight * 0.1) // 10% weight on result position

		// Apply search method bonus
		matchConfidence *= methodBonus

		// Weight this match by its position (top results matter more)
		currentWeight := positionWeight
		weightedConfidence += matchConfidence * currentWeight
		totalWeight += currentWeight
	}

	// Calculate weighted average
	finalConfidence := weightedConfidence / totalWeight

	// Apply confidence boosts for strong indicators
	if len(matches) >= 3 {
		finalConfidence *= 1.1 // Multiple matches boost confidence
	}

	// Apply semantic similarity boost if we have high keyword coverage
	avgKeywordCoverage := 0.0
	for _, match := range matches {
		avgKeywordCoverage += float64(len(match.MatchedKeywords)) / float64(len(keywords))
	}
	avgKeywordCoverage /= float64(len(matches))

	if avgKeywordCoverage >= 0.7 { // High keyword coverage
		finalConfidence *= 1.15
	} else if avgKeywordCoverage >= 0.5 { // Medium keyword coverage
		finalConfidence *= 1.05
	}

	// Cap confidence at 0.95 for direct error correlation
	if finalConfidence > 0.95 {
		finalConfidence = 0.95
	}

	// Ensure minimum threshold for very weak matches
	if finalConfidence < 0.05 {
		finalConfidence = 0.05
	}

	return finalConfidence
}

// IsErrorEntry determines if a log entry represents an error (exported for testing)
func (c *correlator) IsErrorEntry(entry *common.LogEntry) bool {
	return c.isErrorEntry(entry)
}

// ExtractErrorType extracts error type from log entry (exported for testing)
func (c *correlator) ExtractErrorType(entry *common.LogEntry) string {
	return c.extractErrorType(entry)
}

// GenerateErrorSignature generates error signature for grouping (exported for testing)
func (c *correlator) GenerateErrorSignature(entry *common.LogEntry) string {
	return c.generateErrorSignature(entry)
}

// CalculateErrorConfidence calculates confidence for error correlation (exported for testing)
func (c *correlator) CalculateErrorConfidence(keywords []string, matches []*DocumentMatch) float64 {
	return c.calculateErrorConfidence(keywords, matches)
}

// TestCorrelateDirectErrors performs direct error correlation (exported for testing)
func (c *correlator) TestCorrelateDirectErrors(ctx context.Context, entries []*common.LogEntry) ([]*ErrorCorrelation, error) {
	return c.CorrelateDirectErrors(ctx, entries)
}

// isErrorEntry determines if a log entry represents an error
func (c *correlator) isErrorEntry(entry *common.LogEntry) bool {
	// Check log level
	if entry.LogLevel == common.LevelError || entry.LogLevel == common.LevelFatal {
		return true
	}

	// Check for error keywords in the message
	message := strings.ToLower(entry.Message)
	errorIndicators := []string{
		"error", "exception", "failed", "failure", "fault",
		"panic", "fatal", "abort", "crash", "timeout",
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(message, indicator) {
			return true
		}
	}

	return false
}
