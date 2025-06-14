package formatter

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/analyzer"
)

// csvFormatter formats pattern matches as CSV
type csvFormatter struct{}

// NewCSV creates a new CSV formatter
func NewCSV() Formatter {
	return &csvFormatter{}
}

func (f *csvFormatter) Format(analysis *analyzer.Analysis) ([]byte, error) {
	var b bytes.Buffer
	writer := csv.NewWriter(&b)

	// CSV headers
	headers := []string{
		"Pattern ID",
		"Pattern Name",
		"Pattern Type",
		"Match Count",
		"First Seen",
		"Last Seen",
		"Severity",
		"Sample Message",
	}

	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write pattern match data
	for _, match := range analysis.Patterns {
		sampleMessage := ""
		if len(match.Matches) > 0 {
			sampleMessage = escapeCSVString(match.Matches[0].Message)
		}

		record := []string{
			match.Pattern.ID,
			match.Pattern.Name,
			string(match.Pattern.Type),
			fmt.Sprintf("%d", match.Count),
			formatCSVTime(match.FirstSeen),
			formatCSVTime(match.LastSeen),
			fmt.Sprintf("%d", match.Pattern.Severity),
			sampleMessage,
		}

		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return b.Bytes(), nil
}

// formatCSVTime formats time for CSV output
func formatCSVTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// escapeCSVString properly escapes strings for CSV
func escapeCSVString(s string) string {
	// Remove newlines and truncate long messages
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	if len(s) > 100 {
		s = s[:97] + "..."
	}

	return s
}
