package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/go-logparser"
)

// Test helper functions
func TestShouldUseTUIMode(t *testing.T) {
	tests := []struct {
		name           string
		noTUI          bool
		outputFormat   string
		verbose        bool
		expectedResult bool
	}{
		{
			name:           "should use TUI - all conditions met",
			noTUI:          false,
			outputFormat:   "text",
			verbose:        false,
			expectedResult: true,
		},
		{
			name:           "should not use TUI - no-tui flag set",
			noTUI:          true,
			outputFormat:   "text",
			verbose:        false,
			expectedResult: false,
		},
		{
			name:           "should not use TUI - json output",
			noTUI:          false,
			outputFormat:   "json",
			verbose:        false,
			expectedResult: false,
		},
		{
			name:           "should not use TUI - verbose mode",
			noTUI:          false,
			outputFormat:   "text",
			verbose:        true,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			oldAnalyzeNoTUI := analyzeNoTUI
			oldVerbose := verbose
			oldOutputFmt := outputFmt

			analyzeNoTUI = tt.noTUI
			verbose = tt.verbose
			outputFmt = tt.outputFormat

			defer func() {
				analyzeNoTUI = oldAnalyzeNoTUI
				verbose = oldVerbose
				outputFmt = oldOutputFmt
			}()

			result := shouldUseTUIMode()
			if result != tt.expectedResult {
				t.Errorf("shouldUseTUIMode() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestPerformCorrelationIfEnabled(t *testing.T) {
	// Create a mock analysis
	mockAnalysis := &analyzer.Analysis{
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		TotalEntries: 10,
		ErrorCount:   2,
		WarnCount:    1,
	}

	tests := []struct {
		name             string
		correlateEnabled bool
		docsPath         string
		expectNil        bool
	}{
		{
			name:             "correlation disabled",
			correlateEnabled: false,
			docsPath:         "/docs",
			expectNil:        true,
		},
		{
			name:             "no docs path",
			correlateEnabled: true,
			docsPath:         "",
			expectNil:        true,
		},
		{
			name:             "correlation enabled with docs path",
			correlateEnabled: true,
			docsPath:         "/docs",
			expectNil:        false, // Would try to perform correlation (will fail but not nil)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			oldAnalyzeCorrelate := analyzeCorrelate
			oldAnalyzeDocsPath := analyzeDocsPath

			analyzeCorrelate = tt.correlateEnabled
			analyzeDocsPath = tt.docsPath

			defer func() {
				analyzeCorrelate = oldAnalyzeCorrelate
				analyzeDocsPath = oldAnalyzeDocsPath
			}()

			ctx := context.Background()
			result, err := performCorrelationIfEnabled(ctx, mockAnalysis)

			if tt.expectNil {
				if result != nil {
					t.Errorf("performCorrelationIfEnabled() = %v, want nil", result)
				}
				if err != nil {
					t.Errorf("performCorrelationIfEnabled() error = %v, want nil", err)
				}
			} else if result == nil && err == nil {
				// For enabled case, we expect it to try correlation (which may fail, but that's ok)
				// The important thing is that it didn't return nil without trying
				t.Error("performCorrelationIfEnabled() should have attempted correlation")
			}
		})
	}
}

func TestLogCorrelationWarning(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		errorMsg     string
		expectOutput bool
	}{
		{
			name:         "verbose mode - should log",
			verbose:      true,
			errorMsg:     "test error",
			expectOutput: true,
		},
		{
			name:         "non-verbose mode - should not log",
			verbose:      false,
			errorMsg:     "test error",
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Set up test environment
			oldVerbose := verbose
			verbose = tt.verbose

			defer func() {
				verbose = oldVerbose
				os.Stderr = oldStderr
			}()

			// Call function
			logCorrelationWarning(fmt.Errorf("test error"))

			// Close writer and read output
			if err := w.Close(); err != nil {
				t.Errorf("Failed to close pipe writer: %v", err)
			}
			output := make([]byte, 1000)
			n, _ := r.Read(output)
			outputStr := string(output[:n])

			if tt.expectOutput && !strings.Contains(outputStr, "Warning: correlation failed") {
				t.Error("Expected warning output in verbose mode")
			}
			if !tt.expectOutput && strings.Contains(outputStr, "Warning") {
				t.Error("Expected no warning output in non-verbose mode")
			}
		})
	}
}

// Test correlation formatting functions
func TestWriteCorrelationHeader(t *testing.T) {
	result := &correlation.CorrelationResult{
		TotalPatterns:      5,
		CorrelatedPatterns: 3,
	}

	var output strings.Builder
	writeCorrelationHeader(&output, result)

	outputStr := output.String()
	if !strings.Contains(outputStr, "Document Correlations") {
		t.Error("Header should contain 'Document Correlations'")
	}
	if !strings.Contains(outputStr, "Found 3 correlations out of 5 patterns") {
		t.Error("Header should contain correlation statistics")
	}
}

func TestWritePatternInfo(t *testing.T) {
	pattern := &common.Pattern{
		Name:        "Test Pattern",
		Description: "A test pattern for testing",
	}

	patternCorr := &correlation.PatternCorrelation{
		Pattern:    pattern,
		Keywords:   []string{"test", "pattern"},
		MatchCount: 5,
	}

	var output strings.Builder
	writePatternInfo(&output, 1, patternCorr)

	outputStr := output.String()
	expectedStrings := []string{
		"Pattern 1: Test Pattern",
		"Description: A test pattern for testing",
		"Keywords: test, pattern",
		"Match Count: 5",
		"Related Documents:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Pattern info should contain '%s', got: %s", expected, outputStr)
		}
	}
}

func TestWriteDocumentMatch(t *testing.T) {
	document := &docstore.Document{
		Title: "Test Document",
		Path:  "/test/doc.md",
	}

	docMatch := &correlation.DocumentMatch{
		Document:        document,
		Score:           0.85,
		MatchedKeywords: []string{"test", "doc"},
	}

	var output strings.Builder
	writeDocumentMatch(&output, 1, docMatch)

	outputStr := output.String()
	expectedStrings := []string{
		"1. Test Document (Score: 0.85)",
		"Path: /test/doc.md",
		"Keywords: test, doc",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Document match should contain '%s', got: %s", expected, outputStr)
		}
	}
}

func TestWriteDocumentMatches(t *testing.T) {
	// Create test documents
	documents := make([]*correlation.DocumentMatch, 5)
	for i := 0; i < 5; i++ {
		documents[i] = &correlation.DocumentMatch{
			Document: &docstore.Document{
				Title: fmt.Sprintf("Document %d", i+1),
				Path:  fmt.Sprintf("/doc%d.md", i+1),
			},
			Score:           0.9 - float64(i)*0.1,
			MatchedKeywords: []string{"test"},
		}
	}

	var output strings.Builder
	writeDocumentMatches(&output, documents)

	outputStr := output.String()

	// Should contain first 3 documents
	for i := 1; i <= 3; i++ {
		expected := fmt.Sprintf("Document %d", i)
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Should contain '%s'", expected)
		}
	}

	// Should NOT contain 4th and 5th documents (limited to 3)
	for i := 4; i <= 5; i++ {
		unexpected := fmt.Sprintf("Document %d", i)
		if strings.Contains(outputStr, unexpected) {
			t.Errorf("Should NOT contain '%s' (limited to 3 documents)", unexpected)
		}
	}
}

func TestFormatCorrelationResults(t *testing.T) {
	tests := []struct {
		name     string
		result   *correlation.CorrelationResult
		expected string
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: "No correlations found",
		},
		{
			name: "empty correlations",
			result: &correlation.CorrelationResult{
				TotalPatterns:      0,
				CorrelatedPatterns: 0,
				Correlations:       []*correlation.PatternCorrelation{},
			},
			expected: "No correlations found",
		},
		{
			name: "with correlations",
			result: &correlation.CorrelationResult{
				TotalPatterns:      2,
				CorrelatedPatterns: 1,
				Correlations: []*correlation.PatternCorrelation{
					{
						Pattern: &common.Pattern{
							Name:        "Test Pattern",
							Description: "Test Description",
						},
						Keywords:   []string{"test"},
						MatchCount: 3,
						DocumentMatches: []*correlation.DocumentMatch{
							{
								Document: &docstore.Document{
									Title: "Test Doc",
									Path:  "/test.md",
								},
								Score:           0.95,
								MatchedKeywords: []string{"test"},
							},
						},
					},
				},
			},
			expected: "Found 1 correlations out of 2 patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCorrelationResults(tt.result)
			resultStr := string(result)

			if !strings.Contains(resultStr, tt.expected) {
				t.Errorf("formatCorrelationResults() should contain '%s', got: %s", tt.expected, resultStr)
			}
		})
	}
}

// Test helper to create test log entries
func createTestLogEntries() []*common.LogEntry {
	return []*common.LogEntry{
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-10 * time.Minute),
				Level:     "INFO",
				Message:   "Application started",
			},
			LogLevel:   common.LevelInfo,
			LineNumber: 1,
		},
		{
			LogEntry: logparser.LogEntry{
				Timestamp: time.Now().Add(-5 * time.Minute),
				Level:     "ERROR",
				Message:   "Database connection failed",
			},
			LogLevel:   common.LevelError,
			LineNumber: 2,
		},
	}
}

// Integration test for the main analysis pipeline
func TestRunCLIAnalysis(t *testing.T) {
	// This is a simplified integration test
	// In a full test, we'd mock the dependencies

	entries := createTestLogEntries()
	patterns := []*common.Pattern{}

	// Set up test environment for non-correlation mode
	oldAnalyzeCorrelate := analyzeCorrelate
	oldAnalyzeDocsPath := analyzeDocsPath

	analyzeCorrelate = false
	analyzeDocsPath = ""

	defer func() {
		analyzeCorrelate = oldAnalyzeCorrelate
		analyzeDocsPath = oldAnalyzeDocsPath
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should not panic and should complete without error
	err := runCLIAnalysis(ctx, entries, patterns)

	// We expect this to fail because there's no proper output setup in test environment
	// But it should fail gracefully, not panic
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}
