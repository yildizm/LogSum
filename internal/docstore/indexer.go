package docstore

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// MemoryIndex implements an in-memory full-text search index
type MemoryIndex struct {
	terms     map[string]*IndexTerm
	documents map[string]*Document
	stopWords map[string]bool
}

// NewMemoryIndex creates a new in-memory search index
func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		terms:     make(map[string]*IndexTerm),
		documents: make(map[string]*Document),
		stopWords: getDefaultStopWords(),
	}
}

// IndexDocument adds a document to the search index
func (mi *MemoryIndex) IndexDocument(doc *Document) error {
	if doc == nil || doc.ID == "" {
		return fmt.Errorf("invalid document")
	}

	// Remove existing document if it exists
	if err := mi.RemoveDocument(doc.ID); err != nil {
		return fmt.Errorf("failed to remove existing document: %w", err)
	}

	// Store document reference
	mi.documents[doc.ID] = doc

	// Index document content
	mi.indexText(doc.ID, "title", doc.Title)
	mi.indexText(doc.ID, "content", doc.Content)

	// Index metadata
	if doc.Metadata != nil {
		mi.indexText(doc.ID, "author", doc.Metadata.Author)
		for _, tag := range doc.Metadata.Tags {
			mi.indexText(doc.ID, "tags", tag)
		}
	}

	// Index sections
	for _, section := range doc.Sections {
		mi.indexText(doc.ID, "heading", section.Heading)
		mi.indexText(doc.ID, "section_content", section.Content)
	}

	return nil
}

// RemoveDocument removes a document from the index
func (mi *MemoryIndex) RemoveDocument(docID string) error {
	// Remove document reference
	delete(mi.documents, docID)

	// Remove document from all terms
	for termText, term := range mi.terms {
		delete(term.Documents, docID)

		// Remove term if no documents reference it
		if len(term.Documents) == 0 {
			delete(mi.terms, termText)
		} else {
			// Recalculate IDF
			term.IDF = mi.calculateIDF(len(term.Documents))
		}
	}

	return nil
}

// Search performs a search query against the index
func (mi *MemoryIndex) Search(query *SearchQuery) ([]*SearchResult, error) {
	if query == nil || query.Text == "" {
		return []*SearchResult{}, nil
	}

	// Parse query terms
	queryTerms := mi.tokenize(query.Text)
	if len(queryTerms) == 0 {
		return []*SearchResult{}, nil
	}

	// Handle phrase search
	if strings.Contains(query.Text, "\"") {
		return mi.searchPhrase(query)
	}

	// Handle boolean search
	if strings.Contains(strings.ToUpper(query.Text), " AND ") ||
		strings.Contains(strings.ToUpper(query.Text), " OR ") {
		return mi.searchBoolean(query)
	}

	// Regular keyword search
	return mi.searchKeywords(query, queryTerms)
}

// GetStats returns statistics about the index
func (mi *MemoryIndex) GetStats() (*IndexStats, error) {
	termCount := int64(len(mi.terms))
	docCount := int64(len(mi.documents))

	var totalTerms int64
	var longestDoc, shortestDoc int

	for _, doc := range mi.documents {
		docTerms := len(mi.tokenize(doc.Content))
		totalTerms += int64(docTerms)

		if longestDoc == 0 || docTerms > longestDoc {
			longestDoc = docTerms
		}
		if shortestDoc == 0 || docTerms < shortestDoc {
			shortestDoc = docTerms
		}
	}

	avgTerms := float64(0)
	if docCount > 0 {
		avgTerms = float64(totalTerms) / float64(docCount)
	}

	// Estimate index size (rough calculation)
	indexSize := int64(len(mi.terms)*50 + len(mi.documents)*200)

	return &IndexStats{
		TermCount:     termCount,
		DocumentCount: docCount,
		IndexSize:     indexSize,
		AverageTerms:  avgTerms,
		LongestDoc:    longestDoc,
		ShortestDoc:   shortestDoc,
	}, nil
}

// Rebuild rebuilds the entire index with the given documents
func (mi *MemoryIndex) Rebuild(docs []*Document) error {
	// Clear existing index
	mi.terms = make(map[string]*IndexTerm)
	mi.documents = make(map[string]*Document)

	// Index all documents
	for _, doc := range docs {
		if err := mi.IndexDocument(doc); err != nil {
			return fmt.Errorf("failed to index document %s: %w", doc.ID, err)
		}
	}

	return nil
}

// UpdateIndex updates the index with document changes
func (mi *MemoryIndex) UpdateIndex(changes []*DocumentChange) error {
	for _, change := range changes {
		switch change.Type {
		case ChangeAdded, ChangeModified:
			// Document should be reindexed externally
			continue
		case ChangeDeleted:
			if err := mi.RemoveDocument(change.DocumentID); err != nil {
				return fmt.Errorf("failed to remove document %s: %w", change.DocumentID, err)
			}
		}
	}
	return nil
}

// Helper methods

func (mi *MemoryIndex) indexText(docID, field, text string) {
	if text == "" {
		return
	}

	terms := mi.tokenize(text)
	positions := make(map[string][]int)

	// Track term positions
	for pos, term := range terms {
		if mi.stopWords[term] {
			continue
		}

		positions[term] = append(positions[term], pos)
	}

	// Update index
	for term, termPositions := range positions {
		indexTerm, exists := mi.terms[term]
		if !exists {
			indexTerm = &IndexTerm{
				Term:      term,
				Documents: make(map[string]*TermOc),
			}
			mi.terms[term] = indexTerm
		}

		termOcc, exists := indexTerm.Documents[docID]
		if !exists {
			termOcc = &TermOc{
				Frequency: 0,
				Positions: []int{},
				Fields:    []string{},
			}
			indexTerm.Documents[docID] = termOcc
		}

		termOcc.Frequency += len(termPositions)
		termOcc.Positions = append(termOcc.Positions, termPositions...)

		// Add field if not already present
		fieldExists := false
		for _, f := range termOcc.Fields {
			if f == field {
				fieldExists = true
				break
			}
		}
		if !fieldExists {
			termOcc.Fields = append(termOcc.Fields, field)
		}

		// Recalculate IDF
		indexTerm.IDF = mi.calculateIDF(len(indexTerm.Documents))
	}
}

func (mi *MemoryIndex) tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split on word boundaries
	words := regexp.MustCompile(`\b\w+\b`).FindAllString(text, -1)

	var terms []string
	for _, word := range words {
		// Remove punctuation and whitespace
		cleaned := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})

		if len(cleaned) > 2 { // Ignore very short terms
			terms = append(terms, cleaned)
		}
	}

	return terms
}

func (mi *MemoryIndex) calculateIDF(docFreq int) float64 {
	totalDocs := len(mi.documents)
	if totalDocs == 0 || docFreq == 0 {
		return 0
	}
	return math.Log(float64(totalDocs) / float64(docFreq))
}

func (mi *MemoryIndex) calculateTFIDF(termFreq, docLength int, idf float64) float64 {
	if termFreq == 0 || docLength == 0 {
		return 0
	}
	tf := float64(termFreq) / float64(docLength)
	return tf * idf
}

func (mi *MemoryIndex) searchKeywords(query *SearchQuery, terms []string) ([]*SearchResult, error) {
	docScores := make(map[string]float64)
	docMatches := make(map[string][]Match)

	// Score documents for each term
	for _, term := range terms {
		if mi.stopWords[term] {
			continue
		}

		indexTerm, exists := mi.terms[term]
		if !exists {
			// Handle fuzzy search if enabled
			if query.Fuzzy {
				fuzzyTerms := mi.findFuzzyMatches(term)
				for _, fuzzyTerm := range fuzzyTerms {
					if fuzzyIndexTerm, fuzzyExists := mi.terms[fuzzyTerm]; fuzzyExists {
						mi.scoreDocumentsForTerm(fuzzyIndexTerm, docScores, docMatches, term)
					}
				}
			}
			continue
		}

		mi.scoreDocumentsForTerm(indexTerm, docScores, docMatches, term)
	}

	// Convert to search results
	results := make([]*SearchResult, 0, len(docScores))
	for docID, score := range docScores {
		doc, exists := mi.documents[docID]
		if !exists {
			continue
		}

		result := &SearchResult{
			Document: doc,
			Score:    score,
			Matches:  docMatches[docID],
		}

		// Add highlighting if requested
		if query.Highlight {
			result.Highlighted = mi.highlightMatches(doc.Content, terms)
		}

		results = append(results, result)
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func (mi *MemoryIndex) scoreDocumentsForTerm(indexTerm *IndexTerm, docScores map[string]float64, docMatches map[string][]Match, originalTerm string) {
	for docID, termOcc := range indexTerm.Documents {
		doc, exists := mi.documents[docID]
		if !exists {
			continue
		}

		// Calculate TF-IDF score
		docLength := len(mi.tokenize(doc.Content))
		score := mi.calculateTFIDF(termOcc.Frequency, docLength, indexTerm.IDF)
		docScores[docID] += score

		// Add matches
		for _, field := range termOcc.Fields {
			match := Match{
				Field: field,
				Text:  originalTerm,
			}
			docMatches[docID] = append(docMatches[docID], match)
		}
	}
}

func (mi *MemoryIndex) searchPhrase(query *SearchQuery) ([]*SearchResult, error) {
	// Extract phrases from query
	phrases := mi.extractPhrases(query.Text)

	var results []*SearchResult
	for _, phrase := range phrases {
		phraseResults := mi.searchExactPhrase(phrase)
		results = append(results, phraseResults...)
	}

	// Remove duplicates and sort by score
	results = mi.dedupAndSort(results)

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func (mi *MemoryIndex) searchBoolean(query *SearchQuery) ([]*SearchResult, error) {
	// Simple boolean search implementation
	// For now, just treat as regular search
	terms := mi.tokenize(query.Text)
	return mi.searchKeywords(query, terms)
}

func (mi *MemoryIndex) extractPhrases(text string) []string {
	var phrases []string
	inQuotes := false
	var currentPhrase strings.Builder

	for _, char := range text {
		if char == '"' {
			if inQuotes {
				phrase := strings.TrimSpace(currentPhrase.String())
				if phrase != "" {
					phrases = append(phrases, phrase)
				}
				currentPhrase.Reset()
			}
			inQuotes = !inQuotes
		} else if inQuotes {
			currentPhrase.WriteRune(char)
		}
	}

	return phrases
}

func (mi *MemoryIndex) searchExactPhrase(phrase string) []*SearchResult {
	var results []*SearchResult

	for _, doc := range mi.documents {
		if strings.Contains(strings.ToLower(doc.Content), strings.ToLower(phrase)) ||
			strings.Contains(strings.ToLower(doc.Title), strings.ToLower(phrase)) {

			result := &SearchResult{
				Document: doc,
				Score:    1.0, // Fixed score for exact matches
				Matches: []Match{{
					Field: "content",
					Text:  phrase,
				}},
			}
			results = append(results, result)
		}
	}

	return results
}

func (mi *MemoryIndex) findFuzzyMatches(term string) []string {
	var matches []string

	for indexTerm := range mi.terms {
		if mi.levenshteinDistance(term, indexTerm) <= 2 {
			matches = append(matches, indexTerm)
		}
	}

	return matches
}

func (mi *MemoryIndex) levenshteinDistance(s1, s2 string) int {
	if s1 == "" {
		return len(s2)
	}
	if s2 == "" {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func (mi *MemoryIndex) highlightMatches(content string, terms []string) string {
	highlighted := content

	for _, term := range terms {
		if mi.stopWords[term] {
			continue
		}

		// Simple highlighting - wrap matches in **
		re := regexp.MustCompile("(?i)\\b" + regexp.QuoteMeta(term) + "\\b")
		highlighted = re.ReplaceAllStringFunc(highlighted, func(match string) string {
			return "**" + match + "**"
		})
	}

	return highlighted
}

func (mi *MemoryIndex) dedupAndSort(results []*SearchResult) []*SearchResult {
	seen := make(map[string]bool)
	var deduped []*SearchResult

	for _, result := range results {
		if !seen[result.Document.ID] {
			seen[result.Document.ID] = true
			deduped = append(deduped, result)
		}
	}

	// Sort by score (descending)
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Score > deduped[j].Score
	})

	return deduped
}

func getDefaultStopWords() map[string]bool {
	stopWords := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
		"has", "he", "in", "is", "it", "its", "of", "on", "that", "the",
		"to", "was", "will", "with", "would", "you", "your", "yours",
		"i", "me", "my", "we", "us", "our", "ours", "they", "them",
		"their", "theirs", "this", "these", "those", "what", "which",
		"who", "whom", "whose", "where", "when", "why", "how",
	}

	stopWordMap := make(map[string]bool)
	for _, word := range stopWords {
		stopWordMap[word] = true
	}

	return stopWordMap
}
