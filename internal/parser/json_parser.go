package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JSONParser parses JSON formatted logs
type JSONParser struct {
	BaseParser
}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{
		BaseParser: BaseParser{name: "json"},
	}
}

// Parse parses a single JSON log line
func (p *JSONParser) Parse(line string) (*LogEntry, error) {
	return p.parseLine(line)
}

// parseLine implements JSON parsing logic
func (p *JSONParser) parseLine(line string) (*LogEntry, error) {
	// Parse and validate JSON
	raw, err := p.parseJSONLine(line)
	if err != nil {
		return nil, err
	}

	entry := &LogEntry{
		Raw:      line,
		Metadata: make(map[string]string),
	}

	// Extract fields using helper functions
	p.extractTimestamp(raw, entry)
	p.extractLevel(raw, entry)
	p.extractMessage(raw, entry)
	p.extractService(raw, entry)
	p.extractTraceID(raw, entry)
	p.extractMetadata(raw, entry)

	return entry, nil
}

// CanParse checks if line appears to be JSON
func (p *JSONParser) CanParse(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}")
}

// Name returns parser name
func (p *JSONParser) Name() string {
	return p.name
}

// parseJSONLine validates and parses JSON line
func (p *JSONParser) parseJSONLine(line string) (map[string]interface{}, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return raw, nil
}

// extractTimestamp extracts timestamp from various field names
func (p *JSONParser) extractTimestamp(raw map[string]interface{}, entry *LogEntry) {
	for _, key := range []string{"timestamp", "time", "@timestamp", "ts"} {
		if val, ok := raw[key]; ok {
			if t, err := parseTimestamp(val); err == nil {
				entry.Timestamp = t
				delete(raw, key)
				break
			}
		}
	}
}

// extractLevel extracts log level from various field names
func (p *JSONParser) extractLevel(raw map[string]interface{}, entry *LogEntry) {
	for _, key := range []string{"level", "severity", "log.level"} {
		if val, ok := raw[key]; ok {
			if s, ok := val.(string); ok {
				entry.Level = ParseLogLevel(s)
				delete(raw, key)
				break
			}
		}
	}
}

// extractMessage extracts message from various field names
func (p *JSONParser) extractMessage(raw map[string]interface{}, entry *LogEntry) {
	for _, key := range []string{"message", "msg", "log"} {
		if val, ok := raw[key]; ok {
			if s, ok := val.(string); ok {
				entry.Message = s
				delete(raw, key)
				break
			}
		}
	}
}

// extractService extracts service name from various field names
func (p *JSONParser) extractService(raw map[string]interface{}, entry *LogEntry) {
	for _, key := range []string{"service", "app", "application"} {
		if val, ok := raw[key]; ok {
			if s, ok := val.(string); ok {
				entry.Service = s
				delete(raw, key)
				break
			}
		}
	}
}

// extractTraceID extracts trace ID from various field names
func (p *JSONParser) extractTraceID(raw map[string]interface{}, entry *LogEntry) {
	for _, key := range []string{"trace_id", "traceId", "correlation_id"} {
		if val, ok := raw[key]; ok {
			if s, ok := val.(string); ok {
				entry.TraceID = s
				delete(raw, key)
				break
			}
		}
	}
}

// extractMetadata extracts remaining fields as metadata
func (p *JSONParser) extractMetadata(raw map[string]interface{}, entry *LogEntry) {
	for k, v := range raw {
		if s, ok := v.(string); ok {
			entry.Metadata[k] = s
		} else {
			// Convert non-string values
			entry.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}
}

// parseTimestamp attempts to parse various timestamp formats
func parseTimestamp(val interface{}) (time.Time, error) {
	switch v := val.(type) {
	case string:
		// Try common formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02 15:04:05",
			"Jan 02 15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unknown time format: %s", v)
	case float64:
		// Unix timestamp
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type: %T", val)
	}
}
