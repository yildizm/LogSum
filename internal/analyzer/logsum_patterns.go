package analyzer

import (
	"fmt"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/go-promptfmt"
)

// LogAnalysisPattern creates prompts specifically for LogSum log analysis
type LogAnalysisPattern struct {
	promptfmt.BasePattern
	Analysis        *common.Analysis
	ErrorEntries    []*common.LogEntry
	SampleSize      int
	IncludeInsights bool
	IncludePatterns bool
}

// NewLogAnalysisPattern creates a new LogSum-specific log analysis pattern
func NewLogAnalysisPattern() *LogAnalysisPattern {
	return &LogAnalysisPattern{
		BasePattern: promptfmt.BasePattern{
			Description: "Analyzes log data with LogSum-specific context and structure",
			Tags:        []string{"log-analysis", "logsum", "system-monitoring"},
		},
		SampleSize:      10,
		IncludeInsights: true,
		IncludePatterns: true,
	}
}

func (lap *LogAnalysisPattern) WithAnalysis(analysis *common.Analysis) *LogAnalysisPattern {
	lap.Analysis = analysis
	return lap
}

func (lap *LogAnalysisPattern) WithErrorEntries(entries []*common.LogEntry) *LogAnalysisPattern {
	lap.ErrorEntries = entries
	return lap
}

func (lap *LogAnalysisPattern) WithSampleSize(size int) *LogAnalysisPattern {
	lap.SampleSize = size
	return lap
}

func (lap *LogAnalysisPattern) WithoutInsights() *LogAnalysisPattern {
	lap.IncludeInsights = false
	return lap
}

func (lap *LogAnalysisPattern) WithoutPatterns() *LogAnalysisPattern {
	lap.IncludePatterns = false
	return lap
}

func (lap *LogAnalysisPattern) Build() *promptfmt.Prompt {
	if lap.Analysis == nil {
		// Return basic log analysis prompt if no analysis provided
		return promptfmt.New().
			System("You are a log analysis expert specializing in system monitoring and troubleshooting.").
			User("Please analyze the provided log data and identify key issues, patterns, and recommendations.").
			Build()
	}

	pb := promptfmt.New().
		System("You are a LogSum AI assistant specializing in log analysis. Provide structured insights about system health, errors, and operational patterns.").
		User("Analyze this LogSum analysis result:\n\nTime Range: %s to %s\nTotal Entries: %d\nErrors: %d, Warnings: %d",
			lap.Analysis.StartTime.Format(time.RFC3339),
			lap.Analysis.EndTime.Format(time.RFC3339),
			lap.Analysis.TotalEntries,
			lap.Analysis.ErrorCount,
			lap.Analysis.WarnCount)

	// Add error samples if provided
	if len(lap.ErrorEntries) > 0 {
		lap.addErrorSamples(pb)
	}

	// Add patterns context
	if lap.IncludePatterns && len(lap.Analysis.Patterns) > 0 {
		lap.addPatternsContext(pb)
	}

	// Add insights context
	if lap.IncludeInsights && len(lap.Analysis.Insights) > 0 {
		lap.addInsightsContext(pb)
	}

	// Define expected response structure
	type LogAnalysisResponse struct {
		Summary     string `json:"summary"`
		HealthScore int    `json:"health_score"` // 0-100 scale
		KeyFindings []struct {
			Type        string   `json:"type"` // "error", "warning", "pattern", "anomaly"
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Severity    string   `json:"severity"`   // "critical", "high", "medium", "low"
			Confidence  float64  `json:"confidence"` // 0-1 scale
			Impact      string   `json:"impact"`
			Evidence    []string `json:"evidence"`
		} `json:"key_findings"`
		Recommendations []struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Priority    string   `json:"priority"` // "urgent", "high", "medium", "low"
			Category    string   `json:"category"` // "monitoring", "performance", "security", etc.
			ActionItems []string `json:"action_items"`
			Effort      string   `json:"effort"` // "minimal", "low", "medium", "high", "significant"
		} `json:"recommendations"`
		Trends struct {
			ErrorTrend   string   `json:"error_trend"`   // "increasing", "decreasing", "stable", "fluctuating"
			VolumeChange string   `json:"volume_change"` // percentage change description
			TimePatterns []string `json:"time_patterns"` // time-based patterns observed
		} `json:"trends"`
	}

	return pb.ExpectJSON(&LogAnalysisResponse{}).Build()
}

func (lap *LogAnalysisPattern) addErrorSamples(pb *promptfmt.PromptBuilder) {
	sampleSize := minInt(lap.SampleSize, len(lap.ErrorEntries))
	if sampleSize == 0 {
		return
	}

	errorSamples := "Recent Error Samples:\n"
	for i := 0; i < sampleSize; i++ {
		entry := lap.ErrorEntries[i]
		errorSamples += fmt.Sprintf("[%s] %s: %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level,
			entry.Message)
	}

	pb.AddContext("error_samples", errorSamples)
}

func (lap *LogAnalysisPattern) addPatternsContext(pb *promptfmt.PromptBuilder) {
	patternsText := "Detected Patterns:\n"
	for i, pattern := range lap.Analysis.Patterns {
		if i >= 5 { // Limit to top 5 patterns
			break
		}
		patternsText += fmt.Sprintf("- %s: %d occurrences (%s)\n",
			pattern.Pattern.Name,
			pattern.Count,
			pattern.Pattern.Type)
	}

	pb.AddContext("patterns", patternsText)
}

func (lap *LogAnalysisPattern) addInsightsContext(pb *promptfmt.PromptBuilder) {
	insightsText := "LogSum Insights:\n"
	for _, insight := range lap.Analysis.Insights {
		insightsText += fmt.Sprintf("- %s (%s): %s\n",
			insight.Title,
			insight.Type,
			insight.Description)
	}

	pb.AddContext("insights", insightsText)
}

// TimelineAnalysisPattern creates prompts for analyzing temporal patterns in logs
type TimelineAnalysisPattern struct {
	promptfmt.BasePattern
	Entries        []*common.LogEntry
	TimeWindows    []string
	FocusOnErrors  bool
	IncludeMetrics bool
}

func NewTimelineAnalysisPattern() *TimelineAnalysisPattern {
	return &TimelineAnalysisPattern{
		BasePattern: promptfmt.BasePattern{
			Description: "Analyzes temporal patterns and trends in log data",
			Tags:        []string{"timeline", "temporal", "trends", "patterns"},
		},
		TimeWindows:    []string{"hourly", "daily"},
		FocusOnErrors:  true,
		IncludeMetrics: true,
	}
}

func (tap *TimelineAnalysisPattern) WithEntries(entries []*common.LogEntry) *TimelineAnalysisPattern {
	tap.Entries = entries
	return tap
}

func (tap *TimelineAnalysisPattern) WithTimeWindows(windows ...string) *TimelineAnalysisPattern {
	tap.TimeWindows = windows
	return tap
}

func (tap *TimelineAnalysisPattern) WithoutErrorFocus() *TimelineAnalysisPattern {
	tap.FocusOnErrors = false
	return tap
}

func (tap *TimelineAnalysisPattern) WithoutMetrics() *TimelineAnalysisPattern {
	tap.IncludeMetrics = false
	return tap
}

func (tap *TimelineAnalysisPattern) Build() *promptfmt.Prompt {
	pb := promptfmt.New().
		System("You are a temporal log analysis expert. Analyze time-based patterns, trends, and anomalies in log data.").
		User("Analyze temporal patterns in this log data with %d entries", len(tap.Entries))

	// Add time windows context
	if len(tap.TimeWindows) > 0 {
		pb.AddContext("time_windows", "Analyze patterns in these time windows: "+strings.Join(tap.TimeWindows, ", "))
	}

	// Add entry samples organized by time
	if len(tap.Entries) > 0 {
		tap.addTimelineContext(pb)
	}

	// Define expected response structure
	type TimelineResponse struct {
		TimelineAnalysis []struct {
			TimeWindow string `json:"time_window"`
			Period     string `json:"period"`
			Events     int    `json:"events"`
			Errors     int    `json:"errors"`
			Trend      string `json:"trend"`
			Notable    string `json:"notable"`
		} `json:"timeline_analysis"`
		Patterns []struct {
			Pattern     string   `json:"pattern"`
			Description string   `json:"description"`
			Frequency   string   `json:"frequency"`
			TimeSpan    string   `json:"time_span"`
			Examples    []string `json:"examples"`
		} `json:"patterns"`
		Anomalies []struct {
			Timestamp   string  `json:"timestamp"`
			Type        string  `json:"type"`
			Description string  `json:"description"`
			Severity    string  `json:"severity"`
			Confidence  float64 `json:"confidence"`
		} `json:"anomalies"`
	}

	return pb.ExpectJSON(&TimelineResponse{}).Build()
}

func (tap *TimelineAnalysisPattern) addTimelineContext(pb *promptfmt.PromptBuilder) {
	// Group entries by hour for timeline analysis
	hourlyGroups := make(map[string][]*common.LogEntry)

	for _, entry := range tap.Entries {
		hour := entry.Timestamp.Format("2006-01-02 15:00")
		hourlyGroups[hour] = append(hourlyGroups[hour], entry)
	}

	timelineText := "Hourly Breakdown:\n"
	for hour, entries := range hourlyGroups {
		errorCount := 0
		for _, entry := range entries {
			if entry.LogLevel == common.LevelError || entry.LogLevel == common.LevelFatal {
				errorCount++
			}
		}
		timelineText += fmt.Sprintf("%s: %d total entries, %d errors\n", hour, len(entries), errorCount)
	}

	pb.AddContext("timeline", timelineText)
}

// Convenience functions for LogSum-specific patterns
func LogAnalysis() *LogAnalysisPattern {
	return NewLogAnalysisPattern()
}

func TimelineAnalysis() *TimelineAnalysisPattern {
	return NewTimelineAnalysisPattern()
}
