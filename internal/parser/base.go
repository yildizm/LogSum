package parser

import (
	"bufio"
	"fmt"
	"io"
)

// BaseParser provides common functionality for all parsers
type BaseParser struct {
	name       string
	lineNumber int
}

// ParseReader implements the common reader parsing logic
func (b *BaseParser) ParseReader(reader io.Reader) ([]*LogEntry, error) {
	var entries []*LogEntry
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	b.lineNumber = 0
	for scanner.Scan() {
		b.lineNumber++
		line := scanner.Text()

		if line == "" {
			continue // Skip empty lines
		}

		entry, err := b.parseLine(line)
		if err != nil {
			// Log error but continue parsing
			continue
		}

		entry.LineNumber = b.lineNumber
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("scanner error: %w", err)
	}

	return entries, nil
}

// parseLine must be implemented by specific parsers
func (b *BaseParser) parseLine(line string) (*LogEntry, error) {
	panic("parseLine must be implemented by specific parser")
}
