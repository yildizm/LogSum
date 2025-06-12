package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TextParser parses plain text logs with common patterns
type TextParser struct {
	BaseParser
	patterns []*textPattern
}

type textPattern struct {
	regex    *regexp.Regexp
	tsFormat string
	tsIndex  int
	lvlIndex int
	msgIndex int
}

// NewTextParser creates a new text parser
func NewTextParser() *TextParser {
	p := &TextParser{
		BaseParser: BaseParser{name: "text"},
	}
	p.initPatterns()
	return p
}

func (p *TextParser) initPatterns() {
	// Common log patterns
	patterns := []struct {
		pattern  string
		tsFormat string
		tsIndex  int
		lvlIndex int
		msgIndex int
	}{
		// Syslog format: Jan 02 15:04:05 hostname process[pid]: message
		{
			pattern:  `^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+\S+\s+\S+:\s+\[?(\w+)\]?\s+(.*)$`,
			tsFormat: "Jan 02 15:04:05",
			tsIndex:  1,
			lvlIndex: 2,
			msgIndex: 3,
		},
		// Common format: 2006-01-02 15:04:05 [LEVEL] message
		{
			pattern:  `^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+\[(\w+)\]\s+(.*)$`,
			tsFormat: "2006-01-02 15:04:05",
			tsIndex:  1,
			lvlIndex: 2,
			msgIndex: 3,
		},
		// ISO format: 2006-01-02T15:04:05.000Z [LEVEL] message
		{
			pattern:  `^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z?)\s+\[?(\w+)\]?\s+(.*)$`,
			tsFormat: time.RFC3339,
			tsIndex:  1,
			lvlIndex: 2,
			msgIndex: 3,
		},
		// Simple format: [LEVEL] message
		{
			pattern:  `^\[(\w+)\]\s+(.*)$`,
			tsFormat: "",
			tsIndex:  0,
			lvlIndex: 1,
			msgIndex: 2,
		},
	}

	for _, pt := range patterns {
		re, err := regexp.Compile(pt.pattern)
		if err != nil {
			continue
		}
		p.patterns = append(p.patterns, &textPattern{
			regex:    re,
			tsFormat: pt.tsFormat,
			tsIndex:  pt.tsIndex,
			lvlIndex: pt.lvlIndex,
			msgIndex: pt.msgIndex,
		})
	}
}

// Parse parses a single text log line
func (p *TextParser) Parse(line string) (*LogEntry, error) {
	return p.parseLine(line)
}

// parseLine implements text parsing logic
func (p *TextParser) parseLine(line string) (*LogEntry, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	entry := &LogEntry{
		Raw:      line,
		Message:  line,      // Default to full line
		Level:    LevelInfo, // Default level
		Metadata: make(map[string]string),
	}

	// Try each pattern
	for _, pattern := range p.patterns {
		matches := pattern.regex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		// Extract timestamp
		if pattern.tsIndex > 0 && pattern.tsIndex < len(matches) {
			if t, err := time.Parse(pattern.tsFormat, matches[pattern.tsIndex]); err == nil {
				entry.Timestamp = t
			}
		}

		// Extract level
		if pattern.lvlIndex > 0 && pattern.lvlIndex < len(matches) {
			entry.Level = ParseLogLevel(matches[pattern.lvlIndex])
		}

		// Extract message
		if pattern.msgIndex > 0 && pattern.msgIndex < len(matches) {
			entry.Message = matches[pattern.msgIndex]
		}

		// Extract service name from message if possible
		if serviceMatch := regexp.MustCompile(`^\[([^\]]+)\]`).FindStringSubmatch(entry.Message); serviceMatch != nil {
			entry.Service = serviceMatch[1]
			entry.Message = strings.TrimSpace(entry.Message[len(serviceMatch[0]):])
		}

		break // Use first matching pattern
	}

	// If no timestamp found, use current time
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return entry, nil
}

// CanParse checks if line can be parsed as text
func (p *TextParser) CanParse(line string) bool {
	// Text parser is the fallback, can parse anything
	return true
}

// Name returns parser name
func (p *TextParser) Name() string {
	return p.name
}
