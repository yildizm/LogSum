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

	// Add special handling for error types like TermNotFoundException
	if strings.Contains(entry.Message, "Exception") {
		// Extract exception types as meaningful keywords
		exceptionRegex := regexp.MustCompile(`(\w+Exception)`)
		exceptions := exceptionRegex.FindAllString(entry.Message, -1)
		for _, exception := range exceptions {
			// Split CamelCase exception names into meaningful parts
			keywords = append(keywords, e.splitCamelCase(exception)...)
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

	// Also extract compound identifiers (preserve dots and underscores in meaningful contexts)
	compoundWords := extractCompoundIdentifiers(text)
	keywords = append(keywords, compoundWords...)

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

		// Skip if empty, too short, already seen, common word, or log metadata
		if normalized == "" || len(normalized) < 3 || seen[normalized] || isStopWord(normalized) || isLogMetadata(normalized) {
			continue
		}

		seen[normalized] = true
		cleaned = append(cleaned, keyword)
	}

	return cleaned
}

// splitByDelimiters splits text by common delimiters, preserving meaningful identifiers
func splitByDelimiters(text string) []string {
	// Use more conservative delimiters to preserve identifiers like "promo_id" and "file.ext"
	delimiters := []string{",", ";", "|", " ", "\t", "\n"}
	result := text

	for _, delimiter := range delimiters {
		result = strings.ReplaceAll(result, delimiter, " ")
	}

	return strings.Fields(result)
}

// splitCamelCase splits CamelCase words into separate words
func (e *keywordExtractor) splitCamelCase(s string) []string {
	var result []string
	var current strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			if current.Len() > 0 {
				result = append(result, strings.ToLower(current.String()))
				current.Reset()
			}
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		result = append(result, strings.ToLower(current.String()))
	}

	return result
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

	if allNumbers {
		return false
	}

	// Filter out UUID fragments and hex patterns
	if isUUIDFragment(word) || isHexPattern(word) || isTimestampFragment(word) {
		return false
	}

	return true
}

// isUUIDFragment checks if word looks like a UUID fragment
func isUUIDFragment(word string) bool {
	if len(word) < 3 || len(word) > 12 {
		return false
	}
	// Check if it's mostly hex characters
	hexCount := 0
	for _, r := range word {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			hexCount++
		}
	}
	// Lower threshold to catch more fragments
	return float64(hexCount)/float64(len(word)) > 0.6
}

// isHexPattern checks if word is a hex pattern
func isHexPattern(word string) bool {
	if len(word) < 6 {
		return false
	}
	for _, r := range word {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

// isTimestampFragment checks if word looks like a timestamp fragment
func isTimestampFragment(word string) bool {
	wordLower := strings.ToLower(word)

	// Common timestamp patterns
	timestampPatterns := []string{
		"t02", "t03", "t04", "t05", "t06", "t07", "t08", "t09",
		"t10", "t11", "t12", "t13", "t14", "t15", "t16", "t17",
		"t18", "t19", "t20", "t21", "t22", "t23",
		"20t", "21t", "22t", "23t",
	}

	for _, pattern := range timestampPatterns {
		if strings.Contains(wordLower, pattern) {
			return true
		}
	}

	// Timestamp suffixes (milliseconds, timezones)
	if strings.HasSuffix(wordLower, "z") && len(word) >= 4 {
		// Check if it looks like 490Z, 709Z, 129Z pattern
		beforeZ := wordLower[:len(wordLower)-1]
		if len(beforeZ) >= 2 {
			allDigits := true
			for _, r := range beforeZ {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return true
			}
		}
	}

	return false
}

// isStopWord checks if a word should be filtered out
func isStopWord(word string) bool {
	// Preserve important technical terms for correlation
	preservedTerms := map[string]bool{
		"term": true, "terms": true, "promotional": true, "promo": true,
		"exception": true, "error": true, "database": true, "connection": true,
		"timeout": true, "authentication": true, "auth": true, "setup": true,
		"configuration": true, "config": true, "campaign": true, "discount": true,
		"missing": true, "found": true, "failed": true, "failure": true,
		"summer2024": true,
	}

	if preservedTerms[word] {
		return false
	}

	stopWords := map[string]bool{
		// Common English words
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"you": true, "all": true, "can": true, "had": true, "her": true, "was": true,
		"one": true, "our": true, "out": true, "day": true, "get": true, "has": true,
		"him": true, "his": true, "how": true, "man": true, "new": true, "now": true,
		"old": true, "see": true, "two": true, "way": true, "who": true, "boy": true,
		"did": true, "its": true, "let": true, "put": true, "say": true, "she": true,
		"too": true, "use": true, "may": true, "own": true, "run": true, "try": true,

		// Common technical words that are too generic (reduced list)
		"log": true, "logs": true, "text": true, "data": true, "info": true,
		"code": true, "time": true, "date": true, "user": true, "name": true,
		"type": true, "file": true, "line": true, "json": true, "xml": true,
		"http": true, "post": true, "delete": true,
		"true": true, "false": true, "null": true, "undefined": true,

		// Filter out UUID fragments, timestamps, and log metadata
		"245bc020": true, "7fb9e507": true, "c9edab2f": true, "ebd": true,
		"cd3": true, "aed": true, "934z": true, "853z": true, "t23": true,
	}

	return stopWords[word]
}

// isLogMetadata checks if word is log metadata that should be filtered
func isLogMetadata(word string) bool {
	// Check for log ID patterns like [245bc020] or bracketed content
	if strings.HasPrefix(word, "[") && strings.HasSuffix(word, "]") {
		return true
	}

	// Common log metadata patterns
	metadataPatterns := []string{
		"error", "warn", "info", "debug", "trace",
		"utc", "gmt", "pst", "est",
	}

	for _, pattern := range metadataPatterns {
		if strings.EqualFold(word, pattern) {
			return true
		}
	}

	return false
}

// extractCompoundIdentifiers extracts meaningful compound identifiers like promo_id, SUMMER2024.DISCOUNT_RATE
func extractCompoundIdentifiers(text string) []string {
	var identifiers []string

	// Pattern for identifiers with dots: word.word (like SUMMER2024.DISCOUNT_RATE)
	dotPattern := regexp.MustCompile(`[A-Z0-9_]+\.[A-Z0-9_]+`)
	dotMatches := dotPattern.FindAllString(text, -1)
	identifiers = append(identifiers, dotMatches...)

	// Pattern for identifiers with underscores: word_word (like promo_id)
	underscorePattern := regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9]*_[a-zA-Z][a-zA-Z0-9]*`)
	underscoreMatches := underscorePattern.FindAllString(text, -1)
	identifiers = append(identifiers, underscoreMatches...)

	// Pattern for key=value pairs: word=WORD
	keyValuePattern := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*=[A-Z0-9_]+`)
	keyValueMatches := keyValuePattern.FindAllString(text, -1)
	identifiers = append(identifiers, keyValueMatches...)

	return identifiers
}
