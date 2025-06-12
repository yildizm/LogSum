package parser

import (
	"io"
)

// Parser defines the interface for log parsers
type Parser interface {
	// Parse parses a single log line
	Parse(line string) (*LogEntry, error)

	// ParseReader parses all logs from a reader
	ParseReader(reader io.Reader) ([]*LogEntry, error)

	// CanParse checks if this parser can handle the given line
	CanParse(line string) bool

	// Name returns the parser name
	Name() string
}

// ParserFunc is an adapter to allow functions to implement Parser
type ParserFunc func(line string) (*LogEntry, error)

// Factory creates parsers based on format detection
type Factory interface {
	// CreateParser creates appropriate parser for the format
	CreateParser(format string) (Parser, error)

	// DetectFormat attempts to detect log format from sample
	DetectFormat(sample []string) (string, error)

	// RegisterParser registers a new parser type
	RegisterParser(format string, parser Parser)
}
