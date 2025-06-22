package analyzer

import (
	"context"
	"sort"
	"time"

	"github.com/yildizm/LogSum/internal/common"
)

// AnalyzerEngine implements the Analyzer and Engine interfaces
type AnalyzerEngine struct {
	patterns           []*common.Pattern
	matcher            *PatternMatcher
	insightGen         *InsightGenerator
	timelineGen        *TimelineGenerator
	timelineBucketSize time.Duration
	enableInsights     bool
}

func NewEngine() *AnalyzerEngine {
	return &AnalyzerEngine{
		patterns:           []*common.Pattern{},
		matcher:            NewPatternMatcher(),
		insightGen:         NewInsightGenerator(),
		timelineGen:        NewTimelineGenerator(),
		timelineBucketSize: 5 * time.Minute, // Default 5-minute buckets
		enableInsights:     true,
	}
}

// Analyze performs comprehensive analysis on log entries
func (e *AnalyzerEngine) Analyze(ctx context.Context, entries []*common.LogEntry) (*Analysis, error) {
	if len(entries) == 0 {
		return &Analysis{
			Patterns:   []PatternMatch{},
			Insights:   []Insight{},
			RawEntries: []*common.LogEntry{},
		}, nil
	}

	// Initialize analysis
	analysis := &Analysis{
		TotalEntries: len(entries),
		Patterns:     []PatternMatch{},
		Insights:     []Insight{},
		RawEntries:   entries, // Store raw entries for correlation
	}

	// Sort entries by timestamp for timeline analysis
	sortedEntries := make([]*common.LogEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Timestamp.Before(sortedEntries[j].Timestamp)
	})

	// Calculate time range and basic stats
	e.calculateBasicStats(analysis, sortedEntries)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return analysis, ctx.Err()
	default:
	}

	// Pattern matching
	if len(e.patterns) > 0 {
		matches, err := e.matcher.MatchPatterns(ctx, e.patterns, sortedEntries)
		if err != nil {
			return analysis, err
		}
		analysis.Patterns = matches

		// Update error/warning counts based on pattern matches
		e.updateCountsFromPatterns(analysis, matches)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return analysis, ctx.Err()
	default:
	}

	// Generate insights
	if e.enableInsights {
		insights := e.insightGen.GenerateInsights(sortedEntries, analysis.Patterns)
		analysis.Insights = insights
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return analysis, ctx.Err()
	default:
	}

	// Timeline analysis
	if e.timelineBucketSize > 0 {
		timeline := e.timelineGen.GenerateTimeline(sortedEntries, e.timelineBucketSize)
		analysis.Timeline = timeline
	}

	return analysis, nil
}

// AddPattern adds a single pattern to the analyzer
func (e *AnalyzerEngine) AddPattern(pattern *common.Pattern) error {
	e.patterns = append(e.patterns, pattern)
	return e.matcher.AddPattern(pattern)
}

// SetPatterns sets all patterns for the analyzer
func (e *AnalyzerEngine) SetPatterns(patterns []*common.Pattern) error {
	e.patterns = patterns
	return e.matcher.SetPatterns(patterns)
}

// WithTimeline enables timeline analysis with specified bucket size
func (e *AnalyzerEngine) WithTimeline(bucketSize time.Duration) Engine {
	e.timelineBucketSize = bucketSize
	return e
}

// WithPatterns loads patterns from a source (file/directory)
func (e *AnalyzerEngine) WithPatterns(source string) (Engine, error) {
	// This would be implemented to load from files
	// For now, just return self
	return e, nil
}

// WithInsights enables or disables insight generation
func (e *AnalyzerEngine) WithInsights() Engine {
	e.enableInsights = true
	return e
}

// DisableInsights disables insight generation
func (e *AnalyzerEngine) DisableInsights() Engine {
	e.enableInsights = false
	return e
}

// calculateBasicStats calculates basic statistics from log entries
func (e *AnalyzerEngine) calculateBasicStats(analysis *Analysis, entries []*common.LogEntry) {
	if len(entries) == 0 {
		return
	}

	// Find time range
	analysis.StartTime = entries[0].Timestamp
	analysis.EndTime = entries[len(entries)-1].Timestamp

	// Count by severity level
	for _, entry := range entries {
		switch entry.LogLevel {
		case common.LevelError, common.LevelFatal:
			analysis.ErrorCount++
		case common.LevelWarn:
			analysis.WarnCount++
		}
	}
}

// GetPatterns returns the current patterns
func (e *AnalyzerEngine) GetPatterns() []*common.Pattern {
	return e.patterns
}

// SetTimelineBucketSize sets the timeline bucket size
func (e *AnalyzerEngine) SetTimelineBucketSize(size time.Duration) {
	e.timelineBucketSize = size
}

// updateCountsFromPatterns updates error/warning counts based on pattern matches
func (e *AnalyzerEngine) updateCountsFromPatterns(analysis *Analysis, matches []PatternMatch) {
	// Track unique entries to avoid double counting
	errorEntries := make(map[*common.LogEntry]bool)
	warningEntries := make(map[*common.LogEntry]bool)

	for _, match := range matches {
		switch match.Pattern.Type {
		case common.PatternTypeError:
			for _, entry := range match.Matches {
				// Only count if not already counted as error by log level
				if entry.LogLevel != common.LevelError && entry.LogLevel != common.LevelFatal {
					errorEntries[entry] = true
				}
			}
		case common.PatternTypeAnomaly, common.PatternTypePerformance:
			for _, entry := range match.Matches {
				// Count anomalies and performance issues as warnings if not already counted
				if entry.LogLevel != common.LevelError && entry.LogLevel != common.LevelFatal && entry.LogLevel != common.LevelWarn {
					warningEntries[entry] = true
				}
			}
		}
	}

	// Add pattern-based counts to existing counts
	analysis.ErrorCount += len(errorEntries)
	analysis.WarnCount += len(warningEntries)
}
