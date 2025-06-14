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

	b.WriteString("- [Recommendations](#recommendations)\n\n")
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
