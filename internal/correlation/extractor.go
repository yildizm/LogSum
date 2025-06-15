package correlation

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/yildizm/LogSum/internal/common"
)

// KeywordExtractor extracts meaningful keywords from patterns and log entries
type KeywordExtractor interface {
	// ExtractFromPattern extracts keywords from a pattern definition
	ExtractFromPattern(pattern *common.Pattern) []string

	// ExtractFromLogEntry extracts keywords from a log entry
	ExtractFromLogEntry(entry *common.LogEntry) []string
}

// keywordExtractor implements the KeywordExtractor interface
type keywordExtractor struct {
	// Compiled regex patterns for extraction
	identifierRegex *regexp.Regexp
	errorRegex      *regexp.Regexp
	quotedRegex     *regexp.Regexp
	camelCaseRegex  *regexp.Regexp
}

// NewKeywordExtractor creates a new keyword extractor
func NewKeywordExtractor() KeywordExtractor {
	return &keywordExtractor{
		identifierRegex: regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]{2,}`),
		errorRegex:      regexp.MustCompile(`(?i)(error|exception|failure|timeout|connection|authentication|authorization|validation|null|undefined|invalid|missing|denied|forbidden|conflict|overflow|underflow)`),
		quotedRegex:     regexp.MustCompile(`["']([^"']{3,})["']`),
		camelCaseRegex:  regexp.MustCompile(`[A-Z][a-z]+(?:[A-Z][a-z]+)*`),
	}
}

// ExtractFromPattern extracts keywords from a pattern definition
func (e *keywordExtractor) ExtractFromPattern(pattern *common.Pattern) []string {
	if pattern == nil {
		return nil
	}

	var keywords []string

	// Extract from pattern name
	keywords = append(keywords, e.extractFromText(pattern.Name)...)

	// Extract from pattern description
	keywords = append(keywords, e.extractFromText(pattern.Description)...)

	// Extract from regex pattern (literal parts)
	keywords = append(keywords, e.extractFromRegex(pattern.Regex)...)

	return e.cleanKeywords(keywords)
}

// ExtractFromLogEntry extracts keywords from a log entry
func (e *keywordExtractor) ExtractFromLogEntry(entry *common.LogEntry) []string {
	if entry == nil {
		return nil
	}

	var keywords []string

	// Extract from message
	keywords = append(keywords, e.extractFromText(entry.Message)...)

	// Extract from fields
	for key, value := range entry.Fields {
		keywords = append(keywords, e.extractFromText(key)...)
		if strValue, ok := value.(string); ok {
			keywords = append(keywords, e.extractFromText(strValue)...)
		}
	}

	return e.cleanKeywords(keywords)
}

// extractFromText extracts keywords from general text
func (e *keywordExtractor) extractFromText(text string) []string {
	if text == "" {
		return nil
	}

	var keywords []string

	// Extract quoted strings
	quotedMatches := e.quotedRegex.FindAllStringSubmatch(text, -1)
	for _, match := range quotedMatches {
		if len(match) > 1 {
			keywords = append(keywords, match[1])
		}
	}

	// Extract error-related terms
	errorMatches := e.errorRegex.FindAllString(text, -1)
	keywords = append(keywords, errorMatches...)

	// Extract identifiers (camelCase, snake_case, etc.)
	identifierMatches := e.identifierRegex.FindAllString(text, -1)
	keywords = append(keywords, identifierMatches...)

	// Extract CamelCase words
	camelMatches := e.camelCaseRegex.FindAllString(text, -1)
	keywords = append(keywords, camelMatches...)

	// Extract words separated by common delimiters
	words := splitByDelimiters(text)
	for _, word := range words {
		if len(word) > 2 && isValidKeyword(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// extractFromRegex extracts literal keywords from regex patterns
func (e *keywordExtractor) extractFromRegex(pattern string) []string {
	if pattern == "" {
		return nil
	}

	var keywords []string

	// Remove regex metacharacters and extract literal parts
	cleaned := regexp.MustCompile(`[()[\]{}*+?^$|\\.]`).ReplaceAllString(pattern, " ")

	// Extract words from cleaned pattern
	words := strings.Fields(cleaned)
	for _, word := range words {
		if len(word) > 2 && isValidKeyword(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// cleanKeywords removes duplicates and filters keywords
func (e *keywordExtractor) cleanKeywords(keywords []string) []string {
	seen := make(map[string]bool)
	cleaned := make([]string, 0, len(keywords))

	for _, keyword := range keywords {
		// Normalize keyword
		normalized := strings.ToLower(strings.TrimSpace(keyword))

		// Skip if empty, too short, already seen, or common word
		if normalized == "" || len(normalized) < 3 || seen[normalized] || isStopWord(normalized) {
			continue
		}

		seen[normalized] = true
		cleaned = append(cleaned, keyword)
	}

	return cleaned
}

// splitByDelimiters splits text by common delimiters
func splitByDelimiters(text string) []string {
	// Replace common delimiters with spaces
	delimiters := []string{".", "_", "-", ":", "/", "\\", "|", ",", ";", " ", "\t", "\n"}
	result := text

	for _, delimiter := range delimiters {
		result = strings.ReplaceAll(result, delimiter, " ")
	}

	return strings.Fields(result)
}

// isValidKeyword checks if a word is a valid keyword
func isValidKeyword(word string) bool {
	// Must contain at least one letter
	hasLetter := false
	for _, r := range word {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}

	if !hasLetter {
		return false
	}

	// Shouldn't be all numbers
	allNumbers := true
	for _, r := range word {
		if !unicode.IsDigit(r) {
			allNumbers = false
			break
		}
	}

	return !allNumbers
}

// isStopWord checks if a word should be filtered out
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		// Common English words
		"the": true, "and": true, "for": true, "are": true, "but": true, "not": true,
		"you": true, "all": true, "can": true, "had": true, "her": true, "was": true,
		"one": true, "our": true, "out": true, "day": true, "get": true, "has": true,
		"him": true, "his": true, "how": true, "man": true, "new": true, "now": true,
		"old": true, "see": true, "two": true, "way": true, "who": true, "boy": true,
		"did": true, "its": true, "let": true, "put": true, "say": true, "she": true,
		"too": true, "use": true, "may": true, "own": true, "run": true, "try": true,

		// Common technical words that are too generic
		"log": true, "logs": true, "text": true, "data": true, "info": true,
		"code": true, "time": true, "date": true, "user": true, "name": true,
		"type": true, "file": true, "line": true, "json": true, "xml": true,
		"http": true, "post": true, "delete": true,
		"true": true, "false": true, "null": true, "undefined": true,
	}

	return stopWords[word]
}
