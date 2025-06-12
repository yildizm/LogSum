package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/yildizm/LogSum/internal/parser"
)

// PatternMatcher handles efficient pattern matching against log entries
type PatternMatcher struct {
	compiledPatterns []*compiledPattern
	mu               sync.RWMutex
}

type compiledPattern struct {
	pattern       *parser.Pattern
	regex         *regexp.Regexp
	keywords      []string
	keywordsLower []string // Pre-computed lowercase keywords
}

// searchableEntry pre-computes search text
type searchableEntry struct {
	entry      *parser.LogEntry
	searchText string // Pre-computed lowercase search text
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		compiledPatterns: []*compiledPattern{},
	}
}

// AddPattern adds a single pattern to the matcher
func (m *PatternMatcher) AddPattern(pattern *parser.Pattern) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	compiled, err := m.compilePattern(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile pattern %s: %w", pattern.ID, err)
	}

	m.compiledPatterns = append(m.compiledPatterns, compiled)
	return nil
}

// SetPatterns sets all patterns for the matcher
func (m *PatternMatcher) SetPatterns(patterns []*parser.Pattern) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.compiledPatterns = make([]*compiledPattern, 0, len(patterns))

	for _, pattern := range patterns {
		compiled, err := m.compilePattern(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile pattern %s: %w", pattern.ID, err)
		}
		m.compiledPatterns = append(m.compiledPatterns, compiled)
	}

	return nil
}

// MatchPatterns matches all patterns against log entries
func (m *PatternMatcher) MatchPatterns(ctx context.Context, patterns []*parser.Pattern, entries []*parser.LogEntry) ([]PatternMatch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.compiledPatterns) == 0 {
		return []PatternMatch{}, nil
	}

	// Initialize pattern matches
	matches := make(map[string]*PatternMatch)
	for _, cp := range m.compiledPatterns {
		matches[cp.pattern.ID] = &PatternMatch{
			Pattern: cp.pattern,
			Matches: []*parser.LogEntry{},
			Count:   0,
		}
	}

	// Process entries in batches for better performance
	batchSize := 1000
	for i := 0; i < len(entries); i += batchSize {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}

		batch := entries[i:end]
		m.processBatch(batch, matches)
	}

	// Convert map to slice and filter out empty matches
	var result []PatternMatch
	for _, match := range matches {
		if match.Count > 0 {
			result = append(result, *match)
		}
	}

	return result, nil
}

// processBatch processes a batch of entries against all patterns
// Optimized version: pre-compute search text once per entry
func (m *PatternMatcher) processBatch(entries []*parser.LogEntry, matches map[string]*PatternMatch) {
	// Pre-compute search text for all entries
	searchableEntries := m.precomputeSearchText(entries)

	// Match each entry against all patterns
	for _, searchableEntry := range searchableEntries {
		for _, cp := range m.compiledPatterns {
			if !m.matchSearchableEntry(searchableEntry, cp) {
				continue
			}

			match := matches[cp.pattern.ID]
			match.Matches = append(match.Matches, searchableEntry.entry)
			match.Count++

			// Update first/last seen timestamps
			entry := searchableEntry.entry
			if match.FirstSeen.IsZero() || entry.Timestamp.Before(match.FirstSeen) {
				match.FirstSeen = entry.Timestamp
			}
			if match.LastSeen.IsZero() || entry.Timestamp.After(match.LastSeen) {
				match.LastSeen = entry.Timestamp
			}
		}
	}
}

// precomputeSearchText pre-computes search text for a batch of entries
// This avoids repeated string concatenation in the hot path
func (m *PatternMatcher) precomputeSearchText(entries []*parser.LogEntry) []searchableEntry {
	searchableEntries := make([]searchableEntry, len(entries))

	for i, entry := range entries {
		var builder strings.Builder
		builder.Grow(len(entry.Message) + len(entry.Service) + len(entry.Raw) + 2) // Pre-allocate capacity

		builder.WriteString(strings.ToLower(entry.Message))
		builder.WriteByte(' ')
		builder.WriteString(strings.ToLower(entry.Service))
		builder.WriteByte(' ')
		builder.WriteString(strings.ToLower(entry.Raw))

		searchableEntries[i] = searchableEntry{
			entry:      entry,
			searchText: builder.String(),
		}
	}

	return searchableEntries
}

// matchSearchableEntry checks if a searchable entry matches a compiled pattern
func (m *PatternMatcher) matchSearchableEntry(se searchableEntry, cp *compiledPattern) bool {
	// Try regex matching first (more specific)
	if cp.regex != nil {
		if cp.regex.MatchString(se.entry.Raw) || cp.regex.MatchString(se.entry.Message) {
			return true
		}
	}

	// Try keyword matching (faster for simple patterns)
	if len(cp.keywordsLower) > 0 {
		for _, keyword := range cp.keywordsLower {
			if strings.Contains(se.searchText, keyword) {
				return true
			}
		}
	}

	return false
}

// compilePattern compiles a pattern for efficient matching
func (m *PatternMatcher) compilePattern(pattern *parser.Pattern) (*compiledPattern, error) {
	cp := &compiledPattern{
		pattern: pattern,
	}

	// Compile regex if present
	if pattern.Regex != "" {
		regex, err := regexp.Compile("(?i)" + pattern.Regex) // Case-insensitive
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
		cp.regex = regex
	}

	// Prepare keywords for efficient matching
	if len(pattern.Keywords) > 0 {
		cp.keywords = pattern.Keywords
		cp.keywordsLower = make([]string, len(pattern.Keywords))
		for i, keyword := range pattern.Keywords {
			cp.keywordsLower[i] = strings.ToLower(keyword)
		}
	}

	// Validate that pattern has either regex or keywords
	if cp.regex == nil && len(cp.keywords) == 0 {
		return nil, fmt.Errorf("pattern must have either regex or keywords")
	}

	return cp, nil
}

// GetCompiledPatterns returns the compiled patterns (for testing)
func (m *PatternMatcher) GetCompiledPatterns() []*compiledPattern {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.compiledPatterns
}

// MatchSingle matches a single entry against all patterns (useful for real-time analysis)
func (m *PatternMatcher) MatchSingle(entry *parser.LogEntry) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Pre-compute search text once
	searchableEntries := m.precomputeSearchText([]*parser.LogEntry{entry})
	se := searchableEntries[0]

	var matchedPatterns []string
	for _, cp := range m.compiledPatterns {
		if m.matchSearchableEntry(se, cp) {
			matchedPatterns = append(matchedPatterns, cp.pattern.ID)
		}
	}

	return matchedPatterns
}
