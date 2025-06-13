package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/go-logparser"
)

func TestAnalyzerEngine(t *testing.T) {
	engine := NewEngine()

	// Test pattern
	pattern := &common.Pattern{
		ID:       "test_error",
		Name:     "Test Error",
		Type:     common.PatternTypeError,
		Keywords: []string{"error", "failed"},
		Severity: common.LevelError,
	}

	err := engine.AddPattern(pattern)
	if err != nil {
		t.Fatalf("Failed to add pattern: %v", err)
	}

	// Test entries
	entries := []*common.LogEntry{
		createTestEntry(time.Now(), common.LevelInfo, "INFO", "Application started"),
		createTestEntry(time.Now().Add(time.Minute), common.LevelError, "ERROR", "Database connection failed"),
		createTestEntry(time.Now().Add(2*time.Minute), common.LevelError, "ERROR", "Authentication error occurred"),
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
	regexPattern := &common.Pattern{
		ID:    "regex_test",
		Name:  "Regex Test",
		Type:  common.PatternTypeError,
		Regex: "database.*failed",
	}

	// Test keyword pattern
	keywordPattern := &common.Pattern{
		ID:       "keyword_test",
		Name:     "Keyword Test",
		Type:     common.PatternTypeError,
		Keywords: []string{"timeout", "connection"},
	}

	err := matcher.SetPatterns([]*common.Pattern{regexPattern, keywordPattern})
	if err != nil {
		t.Fatalf("Failed to set patterns: %v", err)
	}

	entries := []*common.LogEntry{
		createTestEntry(time.Now(), common.LevelInfo, "INFO", "Database connection failed"),
		createTestEntry(time.Now(), common.LevelInfo, "INFO", "Connection timeout occurred"),
		createTestEntry(time.Now(), common.LevelInfo, "INFO", "Normal operation"),
	}

	ctx := context.Background()
	matches, err := matcher.MatchPatterns(ctx, []*common.Pattern{regexPattern, keywordPattern}, entries)
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
	var entries []*common.LogEntry

	// Normal period
	for i := 0; i < 10; i++ {
		entries = append(entries, createTestEntry(
			baseTime.Add(time.Duration(i)*time.Minute),
			common.LevelInfo, "INFO", "Normal operation"))
	}

	// Error spike period
	for i := 10; i < 20; i++ {
		entries = append(entries, createTestEntry(
			baseTime.Add(time.Duration(i)*time.Minute),
			common.LevelError, "ERROR", "Error occurred"))
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
	entries := []*common.LogEntry{
		createTestEntry(baseTime, common.LevelInfo, "INFO", "First entry"),
		createTestEntry(baseTime.Add(3*time.Minute), common.LevelError, "ERROR", "Error entry"),
		createTestEntry(baseTime.Add(7*time.Minute), common.LevelWarn, "WARN", "Warning entry"),
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
	patterns := []*common.Pattern{
		{
			ID:       "error_pattern",
			Name:     "Error Pattern",
			Type:     common.PatternTypeError,
			Keywords: []string{"error", "failed", "exception"},
		},
		{
			ID:    "timeout_pattern",
			Name:  "Timeout Pattern",
			Type:  common.PatternTypeError,
			Regex: "timeout|timed out",
		},
	}

	err := matcher.SetPatterns(patterns)
	if err != nil {
		b.Fatalf("Failed to set patterns: %v", err)
	}

	// Generate test entries
	entries := make([]*common.LogEntry, entryCount)
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
		entries[i] = createTestEntry(
			time.Now().Add(time.Duration(i)*time.Second),
			common.LevelInfo, "INFO", messages[i%len(messages)])
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
	patterns := []*common.Pattern{
		{
			ID:       "error_pattern",
			Name:     "Error Pattern",
			Type:     common.PatternTypeError,
			Keywords: []string{"error", "failed"},
		},
		{
			ID:       "perf_pattern",
			Name:     "Performance Pattern",
			Type:     common.PatternTypePerformance,
			Keywords: []string{"slow", "timeout"},
		},
	}

	err := engine.SetPatterns(patterns)
	if err != nil {
		b.Fatalf("Failed to set patterns: %v", err)
	}

	// Generate test entries
	entryCount := 10000
	entries := make([]*common.LogEntry, entryCount)

	for i := 0; i < entryCount; i++ {
		level := common.LevelInfo
		levelStr := "INFO"
		message := "Normal operation"

		// Add some errors and performance issues
		if i%100 == 0 {
			level = common.LevelError
			levelStr = "ERROR"
			message = "Database error occurred"
		} else if i%200 == 0 {
			level = common.LevelWarn
			levelStr = "WARN"
			message = "Slow query detected"
		}

		entries[i] = createTestEntry(
			time.Now().Add(time.Duration(i)*time.Second),
			level, levelStr, message)
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
func createTestEntry(timestamp time.Time, level common.LogLevel, levelStr, message string) *common.LogEntry {
	return &common.LogEntry{
		LogEntry: logparser.LogEntry{
			Timestamp: timestamp,
			Level:     levelStr,
			Message:   message,
		},
		LogLevel: level,
	}
}

func createTestEntries(count int, errorRatio float64) []*common.LogEntry {
	entries := make([]*common.LogEntry, count)
	baseTime := time.Now()

	for i := 0; i < count; i++ {
		level := common.LevelInfo
		levelStr := "INFO"
		message := "Normal operation"

		if float64(i)/float64(count) < errorRatio {
			level = common.LevelError
			levelStr = "ERROR"
			message = "Error occurred"
		}

		entry := createTestEntry(
			baseTime.Add(time.Duration(i)*time.Second),
			level, levelStr, message)
		entry.Service = "test-service"
		entries[i] = entry
	}

	return entries
}

func TestPerformanceRequirement(t *testing.T) {
	// Test that 10K entries can be processed in < 500ms (allowing for CI environment)
	engine := NewEngine()

	pattern := &common.Pattern{
		ID:       "test_pattern",
		Name:     "Test Pattern",
		Type:     common.PatternTypeError,
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
