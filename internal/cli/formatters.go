package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/parser"
)

// OutputFormatter defines the interface for output formatting
type OutputFormatter interface {
	Format(analysis *analyzer.Analysis) (string, error)
}

// Enhanced JSON output structures as specified in TASK-007

// EnhancedJSONOutput represents the enhanced JSON structure
type EnhancedJSONOutput struct {
	Summary  *SummaryOutput   `json:"summary"`
	Patterns []*PatternOutput `json:"patterns"`
	Insights []*InsightOutput `json:"insights"`
	Timeline *TimelineOutput  `json:"timeline,omitempty"`
}

// SummaryOutput represents the summary section
type SummaryOutput struct {
	TotalEntries int        `json:"total_entries"`
	ErrorCount   int        `json:"error_count"`
	WarningCount int        `json:"warning_count"`
	TimeRange    *TimeRange `json:"time_range,omitempty"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration string    `json:"duration"`
}

// PatternOutput represents enhanced pattern match output
type PatternOutput struct {
	Pattern       *parser.Pattern    `json:"pattern"`
	Matches       int                `json:"matches"`
	FirstSeen     time.Time          `json:"first_seen,omitempty"`
	LastSeen      time.Time          `json:"last_seen,omitempty"`
	SampleEntries []*parser.LogEntry `json:"sample_entries,omitempty"`
}

// InsightOutput represents enhanced insight output
type InsightOutput struct {
	Type          string  `json:"type"`
	Severity      int     `json:"severity"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Confidence    float64 `json:"confidence"`
	EvidenceCount int     `json:"evidence_count"`
}

// TimelineOutput represents timeline data
type TimelineOutput struct {
	BucketSize string                `json:"bucket_size"`
	Buckets    []analyzer.TimeBucket `json:"buckets"`
}

// TextFormatter formats output as plain text
type TextFormatter struct{}

func (f *TextFormatter) Format(analysis *analyzer.Analysis) (string, error) {
	var b strings.Builder

	f.writeHeader(&b)

	// Statistics section with tree-style layout
	f.writeStatistics(&b, analysis)

	// Top patterns section
	if len(analysis.Patterns) > 0 {
		f.writeTopPatterns(&b, analysis.Patterns)
	}

	// Key insights section
	if len(analysis.Insights) > 0 {
		f.writeKeyInsights(&b, analysis.Insights)
	}

	// Recommendations section
	f.writeTextRecommendations(&b, analysis)

	return b.String(), nil
}

// JSONFormatter formats output as JSON
type JSONFormatter struct{}

func (f *JSONFormatter) Format(analysis *analyzer.Analysis) (string, error) {
	// Create enhanced JSON structure as specified in TASK-007
	output := &EnhancedJSONOutput{
		Summary:  createSummary(analysis),
		Patterns: createPatternOutputs(analysis.Patterns),
		Insights: createInsightOutputs(analysis.Insights),
		Timeline: createTimelineOutput(analysis.Timeline),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// MarkdownFormatter formats output as Markdown
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(analysis *analyzer.Analysis) (string, error) {
	var b strings.Builder

	// Header with generation timestamp
	b.WriteString("# Log Analysis Report\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Table of Contents
	f.writeTableOfContents(&b, analysis)

	// Summary with professional table
	f.writeSummaryTable(&b, analysis)

	// Detected Patterns with enhanced formatting
	if len(analysis.Patterns) > 0 {
		f.writePatternSections(&b, analysis.Patterns)
	}

	// Insights with confidence indicators
	if len(analysis.Insights) > 0 {
		f.writeInsightSections(&b, analysis.Insights)
	}

	// Timeline Analysis
	if analysis.Timeline != nil {
		f.writeTimelineSection(&b, analysis.Timeline)
	}

	// Recommendations
	f.writeRecommendations(&b, analysis)

	return b.String(), nil
}

// CSVFormatter formats pattern matches as CSV
type CSVFormatter struct{}

func (f *CSVFormatter) Format(analysis *analyzer.Analysis) (string, error) {
	var b strings.Builder
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
		return "", fmt.Errorf("failed to write CSV headers: %w", err)
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
			return "", fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	return b.String(), nil
}

// GetFormatter returns the appropriate formatter for the given format
func GetFormatter(format string) OutputFormatter {
	switch strings.ToLower(format) {
	case "json":
		return &JSONFormatter{}
	case "markdown", "md":
		return &MarkdownFormatter{}
	case "csv":
		return &CSVFormatter{}
	default:
		return &TextFormatter{}
	}
}

// Helper functions for enhanced JSON output

// createSummary creates enhanced summary output
func createSummary(analysis *analyzer.Analysis) *SummaryOutput {
	summary := &SummaryOutput{
		TotalEntries: analysis.TotalEntries,
		ErrorCount:   analysis.ErrorCount,
		WarningCount: analysis.WarnCount,
	}

	// Add time range if available
	if !analysis.StartTime.IsZero() && !analysis.EndTime.IsZero() {
		summary.TimeRange = &TimeRange{
			Start:    analysis.StartTime,
			End:      analysis.EndTime,
			Duration: analysis.EndTime.Sub(analysis.StartTime).String(),
		}
	}

	return summary
}

// createPatternOutputs creates enhanced pattern outputs with sample entries
func createPatternOutputs(patterns []analyzer.PatternMatch) []*PatternOutput {
	outputs := make([]*PatternOutput, 0, len(patterns))

	for _, match := range patterns {
		output := &PatternOutput{
			Pattern:   match.Pattern,
			Matches:   match.Count,
			FirstSeen: match.FirstSeen,
			LastSeen:  match.LastSeen,
		}

		// Add first 3 sample entries as specified in TASK-007
		if len(match.Matches) > 0 {
			sampleCount := 3
			if len(match.Matches) < sampleCount {
				sampleCount = len(match.Matches)
			}
			output.SampleEntries = match.Matches[:sampleCount]
		}

		outputs = append(outputs, output)
	}

	return outputs
}

// createInsightOutputs creates enhanced insight outputs
func createInsightOutputs(insights []analyzer.Insight) []*InsightOutput {
	outputs := make([]*InsightOutput, 0, len(insights))

	for _, insight := range insights {
		output := &InsightOutput{
			Type:          string(insight.Type),
			Severity:      int(insight.Severity),
			Title:         insight.Title,
			Description:   insight.Description,
			Confidence:    insight.Confidence,
			EvidenceCount: len(insight.Evidence),
		}

		outputs = append(outputs, output)
	}

	return outputs
}

// createTimelineOutput creates enhanced timeline output
func createTimelineOutput(timeline *analyzer.Timeline) *TimelineOutput {
	if timeline == nil {
		return nil
	}

	return &TimelineOutput{
		BucketSize: timeline.BucketSize.String(),
		Buckets:    timeline.Buckets,
	}
}

// Enhanced Markdown formatter helper methods

// writeTableOfContents writes a professional table of contents
func (f *MarkdownFormatter) writeTableOfContents(b *strings.Builder, analysis *analyzer.Analysis) {
	b.WriteString("## Table of Contents\n")
	b.WriteString("- [Summary](#summary)\n")

	if len(analysis.Patterns) > 0 {
		b.WriteString("- [Detected Patterns](#detected-patterns)\n")
	}

	if len(analysis.Insights) > 0 {
		b.WriteString("- [Insights](#insights)\n")
	}

	if analysis.Timeline != nil {
		b.WriteString("- [Timeline Analysis](#timeline-analysis)\n")
	}

	b.WriteString("- [Recommendations](#recommendations)\n\n")
}

// writeSummaryTable writes a professional summary table
func (f *MarkdownFormatter) writeSummaryTable(b *strings.Builder, analysis *analyzer.Analysis) {
	b.WriteString("## Summary\n\n")

	// Calculate percentages
	errorRate := 0.0
	warningRate := 0.0
	if analysis.TotalEntries > 0 {
		errorRate = float64(analysis.ErrorCount) / float64(analysis.TotalEntries) * 100
		warningRate = float64(analysis.WarnCount) / float64(analysis.TotalEntries) * 100
	}

	timeRange := "N/A"
	if !analysis.StartTime.IsZero() && !analysis.EndTime.IsZero() {
		duration := analysis.EndTime.Sub(analysis.StartTime)
		timeRange = duration.String()
	}

	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(b, "| Total Entries | %s |\n", formatNumber(analysis.TotalEntries))
	fmt.Fprintf(b, "| Errors | %d (%.1f%%) |\n", analysis.ErrorCount, errorRate)
	fmt.Fprintf(b, "| Warnings | %d (%.1f%%) |\n", analysis.WarnCount, warningRate)
	fmt.Fprintf(b, "| Time Range | %s |\n", timeRange)
	fmt.Fprintf(b, "| Patterns Detected | %d |\n\n", len(analysis.Patterns))
}

// writePatternSections writes enhanced pattern sections with samples
func (f *MarkdownFormatter) writePatternSections(b *strings.Builder, patterns []analyzer.PatternMatch) {
	b.WriteString("## Detected Patterns\n\n")

	for _, match := range patterns {
		// Pattern header with emoji
		emoji := getPatternEmoji(match.Pattern.Type)
		fmt.Fprintf(b, "### %s %s (%d occurrences)\n",
			emoji, match.Pattern.Name, match.Count)

		// Time information
		if !match.FirstSeen.IsZero() {
			fmt.Fprintf(b, "First seen: %s | Last seen: %s\n\n",
				match.FirstSeen.Format("15:04:05"),
				match.LastSeen.Format("15:04:05"))
		}

		// Pattern description
		if match.Pattern.Description != "" {
			fmt.Fprintf(b, "**Description**: %s\n\n", match.Pattern.Description)
		}

		// Sample entries
		if len(match.Matches) > 0 {
			b.WriteString("Sample entries:\n")
			b.WriteString("```\n")
			sampleCount := 3
			if len(match.Matches) < sampleCount {
				sampleCount = len(match.Matches)
			}
			for i := 0; i < sampleCount; i++ {
				entry := match.Matches[i]
				fmt.Fprintf(b, "%s\n", entry.Message)
			}
			b.WriteString("```\n\n")
		}
	}
}

// writeInsightSections writes enhanced insight sections
func (f *MarkdownFormatter) writeInsightSections(b *strings.Builder, insights []analyzer.Insight) {
	b.WriteString("## Insights\n\n")

	for _, insight := range insights {
		// Insight header with severity emoji
		emoji := getSeverityEmoji(insight.Severity)
		fmt.Fprintf(b, "### %s %s (Confidence: %.0f%%)\n",
			emoji, insight.Title, insight.Confidence*100)

		// Confidence bar (ASCII representation)
		confidenceBar := createConfidenceBar(insight.Confidence)
		fmt.Fprintf(b, "**Confidence**: %s %.0f%%\n\n",
			confidenceBar, insight.Confidence*100)

		// Description and evidence count
		fmt.Fprintf(b, "**Description**: %s\n", insight.Description)
		fmt.Fprintf(b, "**Evidence**: %d log entries\n\n", len(insight.Evidence))
	}
}

// writeTimelineSection writes timeline analysis with ASCII chart
func (f *MarkdownFormatter) writeTimelineSection(b *strings.Builder, timeline *analyzer.Timeline) {
	b.WriteString("## Timeline Analysis\n\n")

	fmt.Fprintf(b, "**Bucket Size**: %s\n\n", timeline.BucketSize.String())

	// ASCII timeline chart
	b.WriteString("```\n")
	b.WriteString("Activity Timeline:\n")

	maxEntries := 0
	for _, bucket := range timeline.Buckets {
		if bucket.EntryCount > maxEntries {
			maxEntries = bucket.EntryCount
		}
	}

	for _, bucket := range timeline.Buckets {
		barLength := 20
		if maxEntries > 0 {
			barLength = int(float64(bucket.EntryCount) / float64(maxEntries) * 20)
		}

		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 20-barLength)
		fmt.Fprintf(b, "%s │%s│ %d entries\n",
			bucket.Start.Format("15:04"), bar, bucket.EntryCount)
	}
	b.WriteString("```\n\n")
}

// writeRecommendations writes actionable recommendations
func (f *MarkdownFormatter) writeRecommendations(b *strings.Builder, analysis *analyzer.Analysis) {
	b.WriteString("## Recommendations\n\n")

	recommendations := generateRecommendations(analysis)

	for i, rec := range recommendations {
		fmt.Fprintf(b, "%d. %s\n", i+1, rec)
	}

	b.WriteString("\n---\n")
	b.WriteString("*Report generated by LogSum - High-Performance Log Analysis*\n")
}

// Helper functions for enhanced Markdown formatting

// formatNumber formats numbers with commas for readability
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return addCommas(fmt.Sprintf("%d", n))
}

// addCommas adds commas to number strings
func addCommas(s string) string {
	if len(s) <= 3 {
		return s
	}
	return addCommas(s[:len(s)-3]) + "," + s[len(s)-3:]
}

// getPatternEmoji returns emoji for pattern types (deprecated, use GetPatternEmoji)
func getPatternEmoji(patternType parser.PatternType) string {
	return GetPatternEmoji(patternType)
}

// getSeverityEmoji returns emoji for severity levels (deprecated, use GetSeverityEmoji)
func getSeverityEmoji(severity parser.LogLevel) string {
	return GetSeverityEmoji(severity)
}

// createConfidenceBar creates ASCII confidence bar (deprecated, use CreateConfidenceBar)
func createConfidenceBar(confidence float64) string {
	return CreateConfidenceBar(confidence)
}

// generateRecommendations generates actionable recommendations
func generateRecommendations(analysis *analyzer.Analysis) []string {
	var recommendations []string

	// Error-based recommendations
	if analysis.ErrorCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Investigate %d error(s) found in the logs", analysis.ErrorCount))
	}

	// Pattern-based recommendations
	for _, match := range analysis.Patterns {
		switch match.Pattern.Type {
		case parser.PatternTypeError:
			if match.Count > 5 {
				recommendations = append(recommendations,
					fmt.Sprintf("Address recurring %s pattern (%d occurrences)",
						match.Pattern.Name, match.Count))
			}
		case parser.PatternTypePerformance:
			recommendations = append(recommendations,
				fmt.Sprintf("Optimize performance issues related to %s",
					match.Pattern.Name))
		case parser.PatternTypeSecurity:
			recommendations = append(recommendations,
				fmt.Sprintf("Review security concerns: %s (%d occurrences)",
					match.Pattern.Name, match.Count))
		}
	}

	// Generic recommendations if none specific
	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Monitor system regularly for new patterns",
			"Consider setting up automated alerting for critical patterns",
			"Review log retention and analysis policies")
	}

	return recommendations
}

// Enhanced Text formatter helper methods

// writeHeader writes a beautiful header with box drawing
func (f *TextFormatter) writeHeader(b *strings.Builder) {
	header := "Log Analysis Summary"
	headerLen := len(header)

	b.WriteString("╔" + strings.Repeat("═", headerLen+2) + "╗\n")
	b.WriteString("║ " + header + " ║\n")
	b.WriteString("╚" + strings.Repeat("═", headerLen+2) + "╝\n\n")
}

// writeStatistics writes statistics with tree-style formatting
func (f *TextFormatter) writeStatistics(b *strings.Builder, analysis *analyzer.Analysis) {
	b.WriteString(GetSymbol("statistics") + " Statistics\n")

	// Calculate percentages
	errorRate := 0.0
	warningRate := 0.0
	if analysis.TotalEntries > 0 {
		errorRate = float64(analysis.ErrorCount) / float64(analysis.TotalEntries) * 100
		warningRate = float64(analysis.WarnCount) / float64(analysis.TotalEntries) * 100
	}

	fmt.Fprintf(b, "├─ Total Entries: %s\n", formatNumber(analysis.TotalEntries))
	fmt.Fprintf(b, "├─ Errors: %d (%.1f%%)\n", analysis.ErrorCount, errorRate)
	fmt.Fprintf(b, "├─ Warnings: %d (%.1f%%)\n", analysis.WarnCount, warningRate)

	if !analysis.StartTime.IsZero() && !analysis.EndTime.IsZero() {
		duration := analysis.EndTime.Sub(analysis.StartTime)
		fmt.Fprintf(b, "└─ Duration: %s\n\n", duration.String())
	} else {
		b.WriteString("└─ Time Range: N/A\n\n")
	}
}

// writeTopPatterns writes top patterns with visual indicators
func (f *TextFormatter) writeTopPatterns(b *strings.Builder, patterns []analyzer.PatternMatch) {
	b.WriteString(GetSymbol("patterns") + " Top Patterns\n")

	// Sort patterns by count (descending) and take top 5
	sortedPatterns := make([]analyzer.PatternMatch, len(patterns))
	copy(sortedPatterns, patterns)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sortedPatterns)-1; i++ {
		for j := 0; j < len(sortedPatterns)-i-1; j++ {
			if sortedPatterns[j].Count < sortedPatterns[j+1].Count {
				sortedPatterns[j], sortedPatterns[j+1] = sortedPatterns[j+1], sortedPatterns[j]
			}
		}
	}

	maxPatterns := 5
	if len(sortedPatterns) < maxPatterns {
		maxPatterns = len(sortedPatterns)
	}

	for i := 0; i < maxPatterns; i++ {
		pattern := sortedPatterns[i]
		emoji := getPatternEmoji(pattern.Pattern.Type)

		if i == maxPatterns-1 {
			fmt.Fprintf(b, "└─ %s %s (%d)\n", emoji, pattern.Pattern.Name, pattern.Count)
		} else {
			fmt.Fprintf(b, "├─ %s %s (%d)\n", emoji, pattern.Pattern.Name, pattern.Count)
		}
	}
	b.WriteString("\n")
}

// writeKeyInsights writes key insights with confidence indicators
func (f *TextFormatter) writeKeyInsights(b *strings.Builder, insights []analyzer.Insight) {
	b.WriteString(GetSymbol("insights") + " Key Insights\n")

	for i, insight := range insights {
		emoji := getSeverityEmoji(insight.Severity)
		confidenceBar := createConfidenceBar(insight.Confidence)

		if i == len(insights)-1 {
			fmt.Fprintf(b, "└─ %s %s (%.0f%% confidence)\n",
				emoji, insight.Title, insight.Confidence*100)
			fmt.Fprintf(b, "   %s %s\n", confidenceBar, insight.Description)
		} else {
			fmt.Fprintf(b, "├─ %s %s (%.0f%% confidence)\n",
				emoji, insight.Title, insight.Confidence*100)
			fmt.Fprintf(b, "│  %s %s\n", confidenceBar, insight.Description)
		}
	}
	b.WriteString("\n")
}

// writeTextRecommendations writes recommendations for text format
func (f *TextFormatter) writeTextRecommendations(b *strings.Builder, analysis *analyzer.Analysis) {
	recommendations := generateRecommendations(analysis)

	b.WriteString(GetSymbol("recommendations") + " Recommendations\n")
	for i, rec := range recommendations {
		if i < 3 { // Limit to top 3 recommendations for text format
			fmt.Fprintf(b, "• %s\n", rec)
		}
	}
}

// CSV formatter helper functions

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
