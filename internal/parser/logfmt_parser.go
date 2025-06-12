package parser

import (
	"fmt"
	"strings"
)

// LogfmtParser parses logfmt formatted logs
type LogfmtParser struct {
	BaseParser
}

// NewLogfmtParser creates a new logfmt parser
func NewLogfmtParser() *LogfmtParser {
	return &LogfmtParser{
		BaseParser: BaseParser{name: "logfmt"},
	}
}

// Parse parses a single logfmt line
func (p *LogfmtParser) Parse(line string) (*LogEntry, error) {
	return p.parseLine(line)
}

// parseLine implements logfmt parsing logic
func (p *LogfmtParser) parseLine(line string) (*LogEntry, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	entry := &LogEntry{
		Raw:      line,
		Metadata: make(map[string]string),
	}

	// Parse key=value pairs
	pairs := parseLogfmtPairs(line)

	// Extract standard fields
	if val, ok := pairs["timestamp"]; ok {
		if t, err := parseTimestamp(val); err == nil {
			entry.Timestamp = t
		}
		delete(pairs, "timestamp")
	} else if val, ok := pairs["time"]; ok {
		if t, err := parseTimestamp(val); err == nil {
			entry.Timestamp = t
		}
		delete(pairs, "time")
	}

	if val, ok := pairs["level"]; ok {
		entry.Level = ParseLogLevel(val)
		delete(pairs, "level")
	}

	if val, ok := pairs["msg"]; ok {
		entry.Message = val
		delete(pairs, "msg")
	} else if val, ok := pairs["message"]; ok {
		entry.Message = val
		delete(pairs, "message")
	}

	if val, ok := pairs["service"]; ok {
		entry.Service = val
		delete(pairs, "service")
	}

	if val, ok := pairs["trace_id"]; ok {
		entry.TraceID = val
		delete(pairs, "trace_id")
	}

	// Remaining pairs go to metadata
	for k, v := range pairs {
		entry.Metadata[k] = v
	}

	return entry, nil
}

// parseLogfmtPairs parses key=value pairs from a line
func parseLogfmtPairs(line string) map[string]string {
	pairs := make(map[string]string)

	var key string
	var value strings.Builder
	inQuotes := false
	inKey := true

	for i := 0; i < len(line); i++ {
		ch := line[i]

		switch {
		case ch == '=' && inKey && !inQuotes:
			inKey = false

		case ch == '"' && !inKey:
			if i > 0 && line[i-1] != '\\' {
				inQuotes = !inQuotes
			} else {
				value.WriteByte(ch)
			}

		case ch == ' ' && !inQuotes && !inKey:
			// End of value
			if key != "" {
				pairs[key] = value.String()
			}
			key = ""
			value.Reset()
			inKey = true

		case inKey:
			key += string(ch)

		default:
			value.WriteByte(ch)
		}
	}

	// Handle last pair
	if key != "" {
		pairs[key] = value.String()
	}

	return pairs
}

// CanParse checks if line appears to be logfmt
func (p *LogfmtParser) CanParse(line string) bool {
	// Simple heuristic: contains key=value pattern
	return strings.Contains(line, "=") &&
		(strings.Contains(line, "level=") ||
			strings.Contains(line, "msg=") ||
			strings.Contains(line, "time="))
}

// Name returns parser name
func (p *LogfmtParser) Name() string {
	return p.name
}
