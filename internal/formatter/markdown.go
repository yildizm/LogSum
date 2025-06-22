package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/analyzer"
)

// markdownFormatter formats output as Markdown
type markdownFormatter struct{}

// NewMarkdown creates a new Markdown formatter
func NewMarkdown() Formatter {
	return &markdownFormatter{}
}

func (f *markdownFormatter) Format(analysis *analyzer.Analysis) ([]byte, error) {
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

	// AI Analysis (if available)
	f.writeAIAnalysis(&b, analysis)

	return []byte(b.String()), nil
}

// writeTableOfContents writes a professional table of contents
func (f *markdownFormatter) writeTableOfContents(b *strings.Builder, analysis *analyzer.Analysis) {
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

	b.WriteString("- [Recommendations](#recommendations)\n")

	// Add AI Analysis to TOC if available
	if analysis.Context != nil {
		if _, hasAI := analysis.Context["ai_summary"]; hasAI {
			b.WriteString("- [AI Analysis](#ai-analysis)\n")
		}
	}

	b.WriteString("\n")
}

// writeSummaryTable writes a professional summary table
func (f *markdownFormatter) writeSummaryTable(b *strings.Builder, analysis *analyzer.Analysis) {
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
func (f *markdownFormatter) writePatternSections(b *strings.Builder, patterns []analyzer.PatternMatch) {
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
func (f *markdownFormatter) writeInsightSections(b *strings.Builder, insights []analyzer.Insight) {
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
func (f *markdownFormatter) writeTimelineSection(b *strings.Builder, timeline *analyzer.Timeline) {
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
func (f *markdownFormatter) writeRecommendations(b *strings.Builder, analysis *analyzer.Analysis) {
	b.WriteString("## Recommendations\n\n")

	recommendations := generateRecommendations(analysis)

	for i, rec := range recommendations {
		fmt.Fprintf(b, "%d. %s\n", i+1, rec)
	}

	b.WriteString("\n---\n")
	b.WriteString("*Report generated by LogSum - High-Performance Log Analysis*\n")
}

// writeAIAnalysis writes AI-enhanced analysis results in markdown format
func (f *markdownFormatter) writeAIAnalysis(b *strings.Builder, analysis *analyzer.Analysis) {
	if analysis.Context == nil {
		return
	}

	aiData := f.extractAIData(analysis.Context)
	if !aiData.hasAnyData() {
		return
	}

	b.WriteString("\n## AI Analysis\n\n")

	f.writeAISummary(b, aiData.summary, aiData.hasSummary)
	f.writeAIErrorAnalysis(b, aiData.errorAnalysis, aiData.hasErrorAnalysis)
	f.writeAIRootCauses(b, aiData.rootCauses, aiData.hasRootCauses)
	f.writeAIRecommendations(b, aiData.recommendations, aiData.hasRecommendations)
}

// aiAnalysisData holds extracted AI analysis data
type aiAnalysisData struct {
	summary            string
	errorAnalysis      interface{}
	rootCauses         interface{}
	recommendations    interface{}
	hasSummary         bool
	hasErrorAnalysis   bool
	hasRootCauses      bool
	hasRecommendations bool
}

// hasAnyData checks if any AI analysis data is available
func (data aiAnalysisData) hasAnyData() bool {
	return data.hasSummary || data.hasErrorAnalysis || data.hasRootCauses || data.hasRecommendations
}

// extractAIData extracts AI analysis data from context
func (f *markdownFormatter) extractAIData(context map[string]interface{}) aiAnalysisData {
	summary, hasSummary := context["ai_summary"].(string)
	rootCauses, hasRootCauses := context["root_causes"]
	recommendations, hasRecommendations := context["recommendations"]
	errorAnalysis, hasErrorAnalysis := context["error_analysis"]

	return aiAnalysisData{
		summary:            summary,
		errorAnalysis:      errorAnalysis,
		rootCauses:         rootCauses,
		recommendations:    recommendations,
		hasSummary:         hasSummary,
		hasErrorAnalysis:   hasErrorAnalysis,
		hasRootCauses:      hasRootCauses,
		hasRecommendations: hasRecommendations,
	}
}

// writeAISummary writes the AI summary section
func (f *markdownFormatter) writeAISummary(b *strings.Builder, summary string, hasSummary bool) {
	if hasSummary && summary != "" {
		b.WriteString("### 🤖 AI Summary\n\n")
		b.WriteString(summary + "\n\n")
	}
}

// writeAIErrorAnalysis writes the error analysis section
func (f *markdownFormatter) writeAIErrorAnalysis(b *strings.Builder, errorAnalysis interface{}, hasErrorAnalysis bool) {
	if !hasErrorAnalysis {
		return
	}

	errorAnalysisMap, ok := errorAnalysis.(map[string]interface{})
	if !ok {
		return
	}

	b.WriteString("### 🔍 Error Analysis\n\n")

	f.writeErrorSummary(b, errorAnalysisMap)
	f.writeCriticalErrors(b, errorAnalysisMap)
}

// writeErrorSummary writes the error analysis summary
func (f *markdownFormatter) writeErrorSummary(b *strings.Builder, errorAnalysisMap map[string]interface{}) {
	if summary, ok := errorAnalysisMap["summary"].(string); ok && summary != "" {
		b.WriteString(summary + "\n\n")
	}
}

// writeCriticalErrors writes the critical errors section
func (f *markdownFormatter) writeCriticalErrors(b *strings.Builder, errorAnalysisMap map[string]interface{}) {
	criticalErrors, ok := errorAnalysisMap["critical_errors"].([]interface{})
	if !ok || len(criticalErrors) == 0 {
		return
	}

	b.WriteString("#### Critical Errors\n\n")
	for i, err := range criticalErrors {
		f.writeErrorItem(b, err, i+1)
	}
}

// writeErrorItem writes a single error item
func (f *markdownFormatter) writeErrorItem(b *strings.Builder, err interface{}, index int) {
	errMap, ok := err.(map[string]interface{})
	if !ok {
		return
	}

	title, ok := errMap["title"].(string)
	if !ok {
		return
	}

	fmt.Fprintf(b, "%d. **%s**\n", index, title)
	f.writeOptionalField(b, errMap, "description", "   - %s\n")
	f.writeOptionalConfidence(b, errMap, "confidence")
	b.WriteString("\n")
}

// writeOptionalField writes an optional field if it exists
func (f *markdownFormatter) writeOptionalField(b *strings.Builder, data map[string]interface{}, field, format string) {
	if value, ok := data[field].(string); ok && value != "" {
		fmt.Fprintf(b, format, value)
	}
}

// writeOptionalConfidence writes confidence percentage if it exists
func (f *markdownFormatter) writeOptionalConfidence(b *strings.Builder, data map[string]interface{}, field string) {
	if confidence, ok := data[field].(float64); ok {
		fmt.Fprintf(b, "   - Confidence: %.1f%%\n", confidence*100)
	}
}

// writeAIRootCauses writes the root cause analysis section
func (f *markdownFormatter) writeAIRootCauses(b *strings.Builder, rootCauses interface{}, hasRootCauses bool) {
	if !hasRootCauses {
		return
	}

	rootCausesList, ok := rootCauses.([]interface{})
	if !ok || len(rootCausesList) == 0 {
		return
	}

	b.WriteString("### 🎯 Root Cause Analysis\n\n")
	for i, cause := range rootCausesList {
		f.writeRootCauseItem(b, cause, i+1)
	}
}

// writeRootCauseItem writes a single root cause item
func (f *markdownFormatter) writeRootCauseItem(b *strings.Builder, cause interface{}, index int) {
	causeMap, ok := cause.(map[string]interface{})
	if !ok {
		return
	}

	title, ok := causeMap["title"].(string)
	if !ok {
		return
	}

	fmt.Fprintf(b, "%d. **%s**\n", index, title)
	f.writeOptionalField(b, causeMap, "description", "   - %s\n")
	f.writeOptionalConfidence(b, causeMap, "confidence")
	b.WriteString("\n")
}

// writeAIRecommendations writes the AI recommendations section
func (f *markdownFormatter) writeAIRecommendations(b *strings.Builder, recommendations interface{}, hasRecommendations bool) {
	if !hasRecommendations {
		return
	}

	recList, ok := recommendations.([]interface{})
	if !ok || len(recList) == 0 {
		return
	}

	b.WriteString("### 💡 AI Recommendations\n\n")
	for i, rec := range recList {
		f.writeRecommendationItem(b, rec, i+1)
	}
}

// writeRecommendationItem writes a single recommendation item
func (f *markdownFormatter) writeRecommendationItem(b *strings.Builder, rec interface{}, index int) {
	recMap, ok := rec.(map[string]interface{})
	if !ok {
		return
	}

	title, ok := recMap["title"].(string)
	if !ok {
		return
	}

	fmt.Fprintf(b, "%d. **%s**\n", index, title)
	f.writeOptionalField(b, recMap, "description", "   - %s\n")
	f.writeOptionalField(b, recMap, "priority", "   - Priority: %s\n")
	f.writeOptionalField(b, recMap, "effort", "   - Effort: %s\n")
	b.WriteString("\n")
}
