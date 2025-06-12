package parser

import (
	"strings"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// LogEntry represents a parsed log line
type LogEntry struct {
	Timestamp  time.Time         `json:"timestamp"`
	Level      LogLevel          `json:"level"`
	Message    string            `json:"message"`
	Service    string            `json:"service,omitempty"`
	TraceID    string            `json:"trace_id,omitempty"`
	Source     string            `json:"source,omitempty"`
	LineNumber int               `json:"line_number"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Raw        string            `json:"-"`
}

// Pattern represents a log pattern for detection
type Pattern struct {
	ID          string      `yaml:"id" json:"id"`
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description" json:"description"`
	Type        PatternType `yaml:"type" json:"type"`
	Regex       string      `yaml:"regex,omitempty" json:"regex,omitempty"`
	Keywords    []string    `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Severity    LogLevel    `yaml:"severity" json:"severity"`
	Tags        []string    `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// PatternType defines types of patterns
type PatternType string

const (
	PatternTypeError       PatternType = "error"
	PatternTypeAnomaly     PatternType = "anomaly"
	PatternTypePerformance PatternType = "performance"
	PatternTypeSecurity    PatternType = "security"
)

// String methods for LogLevel
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses string to LogLevel
func ParseLogLevel(s string) LogLevel {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}
