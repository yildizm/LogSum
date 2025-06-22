package formatter

import (
	"strings"
	"testing"

	"github.com/yildizm/LogSum/internal/common"
)

func TestWriteTopPatterns_Sorting(t *testing.T) {
	formatter := &terminalFormatter{}

	// Test data with various counts
	patterns := []common.PatternMatch{
		{Pattern: &common.Pattern{Name: "error"}, Count: 5},
		{Pattern: &common.Pattern{Name: "warning"}, Count: 10},
		{Pattern: &common.Pattern{Name: "info"}, Count: 2},
		{Pattern: &common.Pattern{Name: "debug"}, Count: 8},
		{Pattern: &common.Pattern{Name: "fatal"}, Count: 1},
	}

	var b strings.Builder
	formatter.writeTopPatterns(&b, patterns)

	output := b.String()

	// Verify the sorting order by checking positions
	warningPos := strings.Index(output, "warning")
	debugPos := strings.Index(output, "debug")
	errorPos := strings.Index(output, "error")
	infoPos := strings.Index(output, "info")
	fatalPos := strings.Index(output, "fatal")

	// Should be sorted by count: warning(10) > debug(8) > error(5) > info(2) > fatal(1)
	if warningPos > debugPos {
		t.Errorf("warning should appear before debug in sorted output")
	}
	if debugPos > errorPos {
		t.Errorf("debug should appear before error in sorted output")
	}
	if errorPos > infoPos {
		t.Errorf("error should appear before info in sorted output")
	}
	if infoPos > fatalPos {
		t.Errorf("info should appear before fatal in sorted output")
	}
}

func TestWriteTopPatterns_MaxFive(t *testing.T) {
	formatter := &terminalFormatter{}

	// Test data with more than 5 patterns
	patterns := []common.PatternMatch{
		{Pattern: &common.Pattern{Name: "pattern1"}, Count: 10},
		{Pattern: &common.Pattern{Name: "pattern2"}, Count: 9},
		{Pattern: &common.Pattern{Name: "pattern3"}, Count: 8},
		{Pattern: &common.Pattern{Name: "pattern4"}, Count: 7},
		{Pattern: &common.Pattern{Name: "pattern5"}, Count: 6},
		{Pattern: &common.Pattern{Name: "pattern6"}, Count: 5},
		{Pattern: &common.Pattern{Name: "pattern7"}, Count: 4},
	}

	var b strings.Builder
	formatter.writeTopPatterns(&b, patterns)

	output := b.String()
	lines := strings.Split(output, "\n")

	// Count non-empty lines excluding the header
	nonEmptyLines := 0
	for i, line := range lines {
		if i > 0 && strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	// Should have at most 5 pattern lines
	if nonEmptyLines > 5 {
		t.Errorf("Expected at most 5 patterns, got %d lines", nonEmptyLines)
	}

	// Verify pattern6 and pattern7 are not in output (lowest counts)
	if strings.Contains(output, "pattern6") || strings.Contains(output, "pattern7") {
		t.Errorf("Should not include lowest count patterns when more than 5 exist")
	}
}

func TestWriteTopPatterns_EmptyInput(t *testing.T) {
	formatter := &terminalFormatter{}

	var patterns []common.PatternMatch
	var b strings.Builder
	formatter.writeTopPatterns(&b, patterns)

	output := b.String()

	// Should have header but no patterns
	if !strings.Contains(output, "Top Patterns") {
		t.Errorf("Should contain header even with empty input")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 1 {
		t.Errorf("Should only have header line with empty input, got %d lines", len(lines))
	}
}
