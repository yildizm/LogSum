package analyzer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/common"
)

// InsightGenerator generates insights from log analysis
type InsightGenerator struct {
	// Configuration for insight generation
	errorSpikeThreshold  float64 // Minimum error rate increase for spike detection
	anomalyConfidence    float64 // Minimum confidence for anomaly detection
	correlationThreshold float64 // Minimum correlation for root cause analysis
}

// NewInsightGenerator creates a new insight generator
func NewInsightGenerator() *InsightGenerator {
	return &InsightGenerator{
		errorSpikeThreshold:  2.0, // 2x increase in errors
		anomalyConfidence:    0.7, // 70% confidence minimum
		correlationThreshold: 0.6, // 60% correlation minimum
	}
}

// GenerateInsights generates insights from log entries and pattern matches
func (g *InsightGenerator) GenerateInsights(entries []*common.LogEntry, matches []PatternMatch) []Insight {
	var insights []Insight

	if len(entries) == 0 {
		return insights
	}

	// Generate different types of insights
	insights = append(insights, g.detectErrorSpikes(entries, matches)...)
	insights = append(insights, g.detectPerformanceIssues(entries, matches)...)
	insights = append(insights, g.detectAnomalies(entries, matches)...)
	insights = append(insights, g.detectRootCauses(entries, matches)...)

	// Sort insights by confidence (highest first)
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Confidence > insights[j].Confidence
	})

	return insights
}

// detectErrorSpikes detects sudden increases in error frequency
func (g *InsightGenerator) detectErrorSpikes(entries []*common.LogEntry, matches []PatternMatch) []Insight {
	var insights []Insight

	if len(entries) < 10 {
		return insights // Need minimum entries for spike detection
	}

	// Group entries by time windows (5-minute buckets)
	buckets := g.groupByTimeWindow(entries, 5*time.Minute)
	if len(buckets) < 3 {
		return insights // Need at least 3 buckets for comparison
	}

	// Calculate error rates for each bucket
	errorRates := make([]float64, len(buckets))
	for i, bucket := range buckets {
		errorCount := 0
		for _, entry := range bucket.entries {
			if entry.LogLevel >= common.LevelError {
				errorCount++
			}
		}
		if len(bucket.entries) > 0 {
			errorRates[i] = float64(errorCount) / float64(len(bucket.entries))
		}
	}

	// Detect spikes (significant increase from previous periods)
	for i := 2; i < len(errorRates); i++ {
		currentRate := errorRates[i]
		avgPreviousRate := (errorRates[i-1] + errorRates[i-2]) / 2

		if avgPreviousRate > 0 && currentRate/avgPreviousRate >= g.errorSpikeThreshold {
			// Found an error spike
			confidence := minFloat(0.95, currentRate/avgPreviousRate/g.errorSpikeThreshold)

			insight := Insight{
				Type:        InsightTypeErrorSpike,
				Severity:    common.LevelError,
				Title:       "Error Spike Detected",
				Description: g.formatErrorSpikeDescription(currentRate, avgPreviousRate, buckets[i].startTime),
				Evidence:    g.getErrorEntries(buckets[i].entries),
				Confidence:  confidence,
			}
			insights = append(insights, insight)
		}
	}

	return insights
}

// detectPerformanceIssues detects performance-related patterns
func (g *InsightGenerator) detectPerformanceIssues(entries []*common.LogEntry, matches []PatternMatch) []Insight {
	var insights []Insight

	// Look for performance-related patterns
	for _, match := range matches {
		if match.Pattern.Type == common.PatternTypePerformance && match.Count > 0 {
			confidence := g.calculatePatternConfidence(&match, len(entries))

			insight := Insight{
				Type:        InsightTypePerformance,
				Severity:    match.Pattern.Severity,
				Title:       "Performance Issue: " + match.Pattern.Name,
				Description: g.formatPerformanceDescription(&match),
				Evidence:    g.limitEvidence(match.Matches, 5),
				Confidence:  confidence,
			}
			insights = append(insights, insight)
		}
	}

	// Detect slow response patterns in messages
	slowResponses := g.detectSlowResponses(entries)
	if len(slowResponses) > 0 {
		insight := Insight{
			Type:        InsightTypePerformance,
			Severity:    common.LevelWarn,
			Title:       "Slow Response Times Detected",
			Description: g.formatSlowResponseDescription(len(slowResponses)),
			Evidence:    g.limitEvidence(slowResponses, 5),
			Confidence:  0.8,
		}
		insights = append(insights, insight)
	}

	return insights
}

// detectAnomalies detects unusual patterns in the logs
func (g *InsightGenerator) detectAnomalies(entries []*common.LogEntry, matches []PatternMatch) []Insight {
	insights := make([]Insight, 0, 10) // Pre-allocate with initial capacity

	// Look for anomaly-related patterns
	for _, match := range matches {
		if match.Pattern.Type == common.PatternTypeAnomaly && match.Count > 0 {
			confidence := g.calculatePatternConfidence(&match, len(entries))

			insight := Insight{
				Type:        InsightTypeAnomaly,
				Severity:    match.Pattern.Severity,
				Title:       "Anomaly: " + match.Pattern.Name,
				Description: g.formatAnomalyDescription(&match),
				Evidence:    g.limitEvidence(match.Matches, 3),
				Confidence:  confidence,
			}
			insights = append(insights, insight)
		}
	}

	// Detect unusual service patterns
	serviceAnomalies := g.detectServiceAnomalies(entries)
	insights = append(insights, serviceAnomalies...)

	return insights
}

// detectRootCauses attempts to find potential root causes for errors
func (g *InsightGenerator) detectRootCauses(entries []*common.LogEntry, matches []PatternMatch) []Insight {
	var insights []Insight

	// Group error patterns and look for correlations
	errorMatches := make([]PatternMatch, 0)
	for _, match := range matches {
		if match.Pattern.Type == common.PatternTypeError && match.Count > 0 {
			errorMatches = append(errorMatches, match)
		}
	}

	if len(errorMatches) < 2 {
		return insights // Need multiple error patterns for root cause analysis
	}

	// Find temporal correlations between error patterns
	correlations := g.findTemporalCorrelations(errorMatches)
	for _, correlation := range correlations {
		if correlation.strength >= g.correlationThreshold {
			insight := Insight{
				Type:        InsightTypeRootCause,
				Severity:    common.LevelError,
				Title:       "Potential Root Cause",
				Description: g.formatRootCauseDescription(correlation),
				Evidence:    g.limitEvidence(correlation.evidence, 5),
				Confidence:  correlation.strength,
			}
			insights = append(insights, insight)
		}
	}

	return insights
}

// Helper types and functions

type timeBucket struct {
	startTime time.Time
	endTime   time.Time
	entries   []*common.LogEntry
}

// TimeBucketer efficiently groups entries by time windows
type TimeBucketer struct {
	windowSize time.Duration
}

type correlation struct {
	pattern1 string
	pattern2 string
	strength float64
	evidence []*common.LogEntry
}

// NewTimeBucketer creates a new time bucketer
func NewTimeBucketer(windowSize time.Duration) *TimeBucketer {
	return &TimeBucketer{
		windowSize: windowSize,
	}
}

// GroupEntries groups entries by time windows using O(n) algorithm
func (tb *TimeBucketer) GroupEntries(entries []*common.LogEntry) []timeBucket {
	if len(entries) == 0 {
		return nil
	}

	// Use map for O(1) bucket lookup instead of O(n) linear search
	bucketMap := make(map[time.Time]*timeBucket)
	var bucketTimes []time.Time

	for _, entry := range entries {
		bucketStart := entry.Timestamp.Truncate(tb.windowSize)

		bucket, exists := bucketMap[bucketStart]
		if !exists {
			bucket = &timeBucket{
				startTime: bucketStart,
				endTime:   bucketStart.Add(tb.windowSize),
				entries:   make([]*common.LogEntry, 0, 10), // Pre-allocate capacity
			}
			bucketMap[bucketStart] = bucket
			bucketTimes = append(bucketTimes, bucketStart)
		}

		bucket.entries = append(bucket.entries, entry)
	}

	// Convert map to sorted slice
	return tb.sortedBuckets(bucketMap, bucketTimes)
}

// sortedBuckets converts bucket map to sorted slice
func (tb *TimeBucketer) sortedBuckets(bucketMap map[time.Time]*timeBucket, bucketTimes []time.Time) []timeBucket {
	// Sort bucket times chronologically
	sort.Slice(bucketTimes, func(i, j int) bool {
		return bucketTimes[i].Before(bucketTimes[j])
	})

	// Convert to slice in chronological order
	buckets := make([]timeBucket, len(bucketTimes))
	for i, bucketTime := range bucketTimes {
		buckets[i] = *bucketMap[bucketTime]
	}

	return buckets
}

// groupByTimeWindow is the legacy method, now uses optimized TimeBucketer
func (g *InsightGenerator) groupByTimeWindow(entries []*common.LogEntry, windowSize time.Duration) []timeBucket {
	bucketer := NewTimeBucketer(windowSize)
	return bucketer.GroupEntries(entries)
}

func (g *InsightGenerator) calculatePatternConfidence(match *PatternMatch, totalEntries int) float64 {
	// Base confidence on frequency and pattern quality
	frequency := float64(match.Count) / float64(totalEntries)

	// Higher frequency patterns get higher confidence (up to a point)
	freqScore := minFloat(frequency*10, 0.8)

	// Patterns with regex get slightly higher confidence
	patternScore := 0.2
	if match.Pattern.Regex != "" {
		patternScore = 0.3
	}

	return minFloat(freqScore+patternScore, 0.95)
}

func (g *InsightGenerator) getErrorEntries(entries []*common.LogEntry) []*common.LogEntry {
	var errorEntries []*common.LogEntry
	for _, entry := range entries {
		if entry.LogLevel >= common.LevelError {
			errorEntries = append(errorEntries, entry)
		}
	}
	return g.limitEvidence(errorEntries, 5)
}

// Utility functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (g *InsightGenerator) limitEvidence(entries []*common.LogEntry, limit int) []*common.LogEntry {
	if len(entries) <= limit {
		return entries
	}
	return entries[:limit]
}

func (g *InsightGenerator) detectSlowResponses(entries []*common.LogEntry) []*common.LogEntry {
	var slowEntries []*common.LogEntry

	slowKeywords := []string{"slow", "timeout", "taking too long", "high latency", "response time"}

	for _, entry := range entries {
		message := strings.ToLower(entry.Message)
		for _, keyword := range slowKeywords {
			if strings.Contains(message, keyword) {
				slowEntries = append(slowEntries, entry)
				break
			}
		}
	}

	return slowEntries
}

func (g *InsightGenerator) detectServiceAnomalies(entries []*common.LogEntry) []Insight {
	// Simple service-based anomaly detection
	serviceCounts := make(map[string]int)
	serviceErrors := make(map[string]int)

	for _, entry := range entries {
		if entry.Service != "" {
			serviceCounts[entry.Service]++
			if entry.LogLevel >= common.LevelError {
				serviceErrors[entry.Service]++
			}
		}
	}

	var insights []Insight
	for service, errorCount := range serviceErrors {
		totalCount := serviceCounts[service]
		if totalCount > 10 && float64(errorCount)/float64(totalCount) > 0.5 {
			// Service has >50% error rate with significant volume
			insight := Insight{
				Type:        InsightTypeAnomaly,
				Severity:    common.LevelError,
				Title:       "High Error Rate in Service",
				Description: g.formatServiceAnomalyDescription(service, errorCount, totalCount),
				Evidence:    []*common.LogEntry{}, // Would populate with service entries
				Confidence:  0.85,
			}
			insights = append(insights, insight)
		}
	}

	return insights
}

func (g *InsightGenerator) findTemporalCorrelations(matches []PatternMatch) []correlation {
	// Simplified correlation detection
	var correlations []correlation

	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			pattern1 := matches[i]
			pattern2 := matches[j]

			// Check if patterns occur close together in time
			strength := g.calculateTemporalCorrelation(&pattern1, &pattern2)
			if strength > g.correlationThreshold {
				correlations = append(correlations, correlation{
					pattern1: pattern1.Pattern.Name,
					pattern2: pattern2.Pattern.Name,
					strength: strength,
					evidence: append(g.limitEvidence(pattern1.Matches, 2),
						g.limitEvidence(pattern2.Matches, 2)...),
				})
			}
		}
	}

	return correlations
}

func (g *InsightGenerator) calculateTemporalCorrelation(match1, match2 *PatternMatch) float64 {
	// Simplified: check if patterns have overlapping time ranges
	if !match1.FirstSeen.IsZero() && !match2.FirstSeen.IsZero() {
		overlap := g.calculateTimeOverlap(match1.FirstSeen, match1.LastSeen,
			match2.FirstSeen, match2.LastSeen)
		return overlap
	}
	return 0.0
}

func (g *InsightGenerator) calculateTimeOverlap(start1, end1, start2, end2 time.Time) float64 {
	// Calculate overlap ratio
	latest_start := start1
	if start2.After(start1) {
		latest_start = start2
	}

	earliest_end := end1
	if end2.Before(end1) {
		earliest_end = end2
	}

	if earliest_end.After(latest_start) {
		overlap := earliest_end.Sub(latest_start)
		total1 := end1.Sub(start1)
		total2 := end2.Sub(start2)
		avgTotal := (total1 + total2) / 2

		if avgTotal > 0 {
			return float64(overlap) / float64(avgTotal)
		}
	}

	return 0.0
}

// Format functions for descriptions

func (g *InsightGenerator) formatErrorSpikeDescription(current, previous float64, timestamp time.Time) string {
	increase := (current / previous) * 100
	return fmt.Sprintf("Error rate increased by %.1f%% at %s (from %.1f%% to %.1f%%)",
		increase-100, timestamp.Format("15:04:05"), previous*100, current*100)
}

func (g *InsightGenerator) formatPerformanceDescription(match *PatternMatch) string {
	return fmt.Sprintf("Pattern '%s' detected %d times, indicating potential performance issues",
		match.Pattern.Name, match.Count)
}

func (g *InsightGenerator) formatSlowResponseDescription(count int) string {
	return fmt.Sprintf("Detected %d log entries indicating slow response times or timeouts", count)
}

func (g *InsightGenerator) formatAnomalyDescription(match *PatternMatch) string {
	return fmt.Sprintf("Anomalous pattern '%s' detected %d times", match.Pattern.Name, match.Count)
}

func (g *InsightGenerator) formatServiceAnomalyDescription(service string, errors, total int) string {
	errorRate := float64(errors) / float64(total) * 100
	return fmt.Sprintf("Service '%s' has high error rate: %d errors out of %d entries (%.1f%%)",
		service, errors, total, errorRate)
}

func (g *InsightGenerator) formatRootCauseDescription(corr correlation) string {
	return fmt.Sprintf("Strong correlation (%.1f%%) between '%s' and '%s' patterns suggests potential causal relationship",
		corr.strength*100, corr.pattern1, corr.pattern2)
}

// Helper functions
