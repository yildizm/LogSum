package parser

import (
	"fmt"
	"strings"
	"sync"
)

// DefaultFactory is the default parser factory
var DefaultFactory = NewFactory()

// parserFactory implements the Factory interface
type parserFactory struct {
	parsers map[string]Parser
	mu      sync.RWMutex
}

// NewFactory creates a new parser factory
func NewFactory() Factory {
	f := &parserFactory{
		parsers: make(map[string]Parser),
	}

	// Register default parsers
	f.RegisterParser("json", NewJSONParser())
	f.RegisterParser("logfmt", NewLogfmtParser())
	f.RegisterParser("text", NewTextParser())

	return f
}

// CreateParser creates a parser for the specified format
func (f *parserFactory) CreateParser(format string) (Parser, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	format = strings.ToLower(format)
	if parser, ok := f.parsers[format]; ok {
		return parser, nil
	}

	return nil, fmt.Errorf("unknown log format: %s", format)
}

// DetectFormat attempts to detect log format from samples
func (f *parserFactory) DetectFormat(samples []string) (string, error) {
	if len(samples) == 0 {
		return "", fmt.Errorf("no samples provided")
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Count successful parses for each format
	scores := make(map[string]int)

	for _, sample := range samples {
		for format, parser := range f.parsers {
			if parser.CanParse(sample) {
				scores[format]++
			}
		}
	}

	// Find format with highest score
	var bestFormat string
	var bestScore int

	// Check in order of preference
	preferredOrder := []string{"json", "logfmt", "text"}
	for _, format := range preferredOrder {
		if score := scores[format]; score > bestScore {
			bestFormat = format
			bestScore = score
		}
	}

	if bestFormat == "" {
		return "text", nil // Default to text
	}

	return bestFormat, nil
}

// RegisterParser registers a new parser
func (f *parserFactory) RegisterParser(format string, parser Parser) {
	f.mu.Lock()
	defer f.mu.Unlock()

	format = strings.ToLower(format)
	f.parsers[format] = parser
}

// ParseAuto attempts to parse with format detection
func ParseAuto(lines []string) ([]*LogEntry, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	// Detect format from first few lines
	sampleSize := 10
	if len(lines) < sampleSize {
		sampleSize = len(lines)
	}

	format, err := DefaultFactory.DetectFormat(lines[:sampleSize])
	if err != nil {
		return nil, fmt.Errorf("format detection failed: %w", err)
	}

	parser, err := DefaultFactory.CreateParser(format)
	if err != nil {
		return nil, err
	}

	entries := make([]*LogEntry, 0, len(lines)) // Pre-allocate with capacity
	for i, line := range lines {
		entry, err := parser.Parse(line)
		if err != nil {
			continue // Skip unparseable lines
		}
		entry.LineNumber = i + 1
		entries = append(entries, entry)
	}

	return entries, nil
}
