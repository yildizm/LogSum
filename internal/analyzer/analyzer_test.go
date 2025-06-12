package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/parser"
)

func TestAnalyzerEngine(t *testing.T) {
	engine := NewEngine()

	// Test pattern
	pattern := &parser.Pattern{
		ID:       "test_error",
		Name:     "Test Error",
		Type:     parser.PatternTypeError,
		Keywords: []string{"error", "failed"},
		Severity: parser.LevelError,
	}

	err := engine.AddPattern(pattern)
	if err != nil {
		t.Fatalf("Failed to add pattern: %v", err)
	}

	// Test entries
	entries := []*parser.LogEntry{
		{
			Timestamp: time.Now(),
			Level:     parser.LevelInfo,
			Message:   "Application started",
		},
		{
			Timestamp: time.Now().Add(time.Minute),
			Level:     parser.LevelError,
			Message:   "Database connection failed",
		},
		{
			Timestamp: time.Now().Add(2 * time.Minute),
			Level:     parser.LevelError,
			Message:   "Authentication error occurred",
		},
	}

	ctx := context.Background()
	analysis, err := engine.Analyze(ctx, entries)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Verify results
	if analysis.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", analysis.TotalEntries)
	}

	if analysis.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", analysis.ErrorCount)
	}

	if len(analysis.Patterns) == 0 {
		t.Error("Expected pattern matches, got none")
	}
}

func TestPatternMatcher(t *testing.T) {
	matcher := NewPatternMatcher()

	// Test regex pattern
	regexPattern := &parser.Pattern{
		ID:    "regex_test",
		Name:  "Regex Test",
		Type:  parser.PatternTypeError,
		Regex: "database.*failed",
	}

	// Test keyword pattern
	keywordPattern := &parser.Pattern{
		ID:       "keyword_test",
		Name:     "Keyword Test",
		Type:     parser.PatternTypeError,
		Keywords: []string{"timeout", "connection"},
	}

	err := matcher.SetPatterns([]*parser.Pattern{regexPattern, keywordPattern})
	if err != nil {
		t.Fatalf("Failed to set patterns: %v", err)
	}

	entries := []*parser.LogEntry{
		{Message: "Database connection failed"},
		{Message: "Connection timeout occurred"},
		{Message: "Normal operation"},
	}

	ctx := context.Background()
	matches, err := matcher.MatchPatterns(ctx, []*parser.Pattern{regexPattern, keywordPattern}, entries)
	if err != nil {
		t.Fatalf("Pattern matching failed: %v", err)
	}

	if len(matches) == 0 {
		t.Error("Expected pattern matches, got none")
	}

	// Check specific matches
	foundRegex := false
	foundKeyword := false
	for _, match := range matches {
		if match.Pattern.ID == "regex_test" && match.Count > 0 {
			foundRegex = true
		}
		if match.Pattern.ID == "keyword_test" && match.Count > 0 {
			foundKeyword = true
		}
	}

	if !foundRegex {
		t.Error("Regex pattern should have matched")
	}
	if !foundKeyword {
		t.Error("Keyword pattern should have matched")
	}
}

func TestInsightGenerator(t *testing.T) {
	gen := NewInsightGenerator()

	// Create entries with error spike pattern
	baseTime := time.Now()
	var entries []*parser.LogEntry

	// Normal period
	for i := 0; i < 10; i++ {
		entries = append(entries, &parser.LogEntry{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Level:     parser.LevelInfo,
			Message:   "Normal operation",
		})
	}

	// Error spike period
	for i := 10; i < 20; i++ {
		entries = append(entries, &parser.LogEntry{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Level:     parser.LevelError,
			Message:   "Error occurred",
		})
	}

	matches := []PatternMatch{} // Empty for this test
	insights := gen.GenerateInsights(entries, matches)

	if len(insights) == 0 {
		t.Error("Expected insights to be generated")
	}

	// Check for error spike insight
	foundErrorSpike := false
	for _, insight := range insights {
		if insight.Type == InsightTypeErrorSpike {
			foundErrorSpike = true
			break
		}
	}

	if !foundErrorSpike {
		t.Error("Expected error spike insight to be generated")
	}
}

func TestTimelineGenerator(t *testing.T) {
	gen := NewTimelineGenerator()

	baseTime := time.Now()
	entries := []*parser.LogEntry{
		{
			Timestamp: baseTime,
			Level:     parser.LevelInfo,
			Message:   "First entry",
		},
		{
			Timestamp: baseTime.Add(3 * time.Minute),
			Level:     parser.LevelError,
			Message:   "Error entry",
		},
		{
			Timestamp: baseTime.Add(7 * time.Minute),
			Level:     parser.LevelWarn,
			Message:   "Warning entry",
		},
	}

	timeline := gen.GenerateTimeline(entries, 5*time.Minute)

	if timeline == nil {
		t.Fatal("Timeline should not be nil")
	}

	if len(timeline.Buckets) == 0 {
		t.Error("Timeline should have buckets")
	}

	if timeline.BucketSize != 5*time.Minute {
		t.Errorf("Expected bucket size 5m, got %v", timeline.BucketSize)
	}

	// Verify entries are distributed correctly
	totalEntries := 0
	for _, bucket := range timeline.Buckets {
		totalEntries += bucket.EntryCount
	}

	if totalEntries != len(entries) {
		t.Errorf("Expected %d total entries in buckets, got %d", len(entries), totalEntries)
	}
}

// Benchmark tests
func BenchmarkPatternMatching10K(b *testing.B) {
	benchmarkPatternMatching(b, 10000)
}

func BenchmarkPatternMatching100K(b *testing.B) {
	benchmarkPatternMatching(b, 100000)
}

func benchmarkPatternMatching(b *testing.B, entryCount int) {
	matcher := NewPatternMatcher()

	// Setup patterns
	patterns := []*parser.Pattern{
		{
			ID:       "error_pattern",
			Name:     "Error Pattern",
			Type:     parser.PatternTypeError,
			Keywords: []string{"error", "failed", "exception"},
		},
		{
			ID:    "timeout_pattern",
			Name:  "Timeout Pattern",
			Type:  parser.PatternTypeError,
			Regex: "timeout|timed out",
		},
	}

	err := matcher.SetPatterns(patterns)
	if err != nil {
		b.Fatalf("Failed to set patterns: %v", err)
	}

	// Generate test entries
	entries := make([]*parser.LogEntry, entryCount)
	messages := []string{
		"Normal operation",
		"Database error occurred",
		"Connection failed",
		"Request timeout",
		"System running normally",
		"Authentication failed",
		"Network timeout detected",
	}

	for i := 0; i < entryCount; i++ {
		entries[i] = &parser.LogEntry{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Level:     parser.LevelInfo,
			Message:   messages[i%len(messages)],
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := matcher.MatchPatterns(ctx, patterns, entries)
		if err != nil {
			b.Fatalf("Pattern matching failed: %v", err)
		}
	}
}

func BenchmarkFullAnalysis(b *testing.B) {
	engine := NewEngine()

	// Setup patterns
	patterns := []*parser.Pattern{
		{
			ID:       "error_pattern",
			Name:     "Error Pattern",
			Type:     parser.PatternTypeError,
			Keywords: []string{"error", "failed"},
		},
		{
			ID:       "perf_pattern",
			Name:     "Performance Pattern",
			Type:     parser.PatternTypePerformance,
			Keywords: []string{"slow", "timeout"},
		},
	}

	err := engine.SetPatterns(patterns)
	if err != nil {
		b.Fatalf("Failed to set patterns: %v", err)
	}

	// Generate test entries
	entryCount := 10000
	entries := make([]*parser.LogEntry, entryCount)

	for i := 0; i < entryCount; i++ {
		level := parser.LevelInfo
		message := "Normal operation"

		// Add some errors and performance issues
		if i%100 == 0 {
			level = parser.LevelError
			message = "Database error occurred"
		} else if i%200 == 0 {
			level = parser.LevelWarn
			message = "Slow query detected"
		}

		entries[i] = &parser.LogEntry{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Level:     level,
			Message:   message,
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := engine.Analyze(ctx, entries)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

// Test helper functions
func createTestEntries(count int, errorRatio float64) []*parser.LogEntry {
	entries := make([]*parser.LogEntry, count)
	baseTime := time.Now()

	for i := 0; i < count; i++ {
		level := parser.LevelInfo
		message := "Normal operation"

		if float64(i)/float64(count) < errorRatio {
			level = parser.LevelError
			message = "Error occurred"
		}

		entries[i] = &parser.LogEntry{
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
			Level:     level,
			Message:   message,
			Service:   "test-service",
		}
	}

	return entries
}

func TestPerformanceRequirement(t *testing.T) {
	// Test that 10K entries can be processed in < 500ms (allowing for CI environment)
	engine := NewEngine()

	pattern := &parser.Pattern{
		ID:       "test_pattern",
		Name:     "Test Pattern",
		Type:     parser.PatternTypeError,
		Keywords: []string{"error"},
	}

	err := engine.AddPattern(pattern)
	if err != nil {
		t.Fatalf("Failed to add pattern: %v", err)
	}
	entries := createTestEntries(10000, 0.1) // 10% error rate

	start := time.Now()
	ctx := context.Background()
	_, err = engine.Analyze(ctx, entries)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if duration > 500*time.Millisecond {
		t.Errorf("Performance requirement not met: took %v, should be < 500ms", duration)
	}

	t.Logf("Analysis of 10K entries took: %v", duration)
}
