package formatter

import (
	"fmt"
	"strings"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/go-termfmt"
)

// terminalFormatter formats output as plain text for terminal display using go-termfmt
type terminalFormatter struct {
	opts *termfmt.TerminalOptions
}

// NewTerminal creates a new terminal formatter with optional color support
func NewTerminal(color bool) Formatter {
	opts := termfmt.DefaultOptions()
	opts.Color = color
	opts.Emoji = true
	return &terminalFormatter{opts: opts}
}

func (f *terminalFormatter) Format(analysis *analyzer.Analysis) ([]byte, error) {
	var b strings.Builder

	// Header with custom box drawing to match original
	f.writeHeader(&b)

	// Statistics section with tree view
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

	return []byte(b.String()), nil
}

// writeStatistics writes statistics with tree-style formatting using go-termfmt
func (f *terminalFormatter) writeStatistics(b *strings.Builder, analysis *analyzer.Analysis) {
	symbol := termfmt.GetEmoji("statistics", f.opts)
	b.WriteString(symbol + " Statistics\n")

	// Calculate percentages
	errorRate := 0.0
	warningRate := 0.0
	if analysis.TotalEntries > 0 {
		errorRate = float64(analysis.ErrorCount) / float64(analysis.TotalEntries) * 100
		warningRate = float64(analysis.WarnCount) / float64(analysis.TotalEntries) * 100
	}

	// Create tree items for statistics
	items := []termfmt.TreeItem{
		{Label: "Total Entries", Value: formatNumber(analysis.TotalEntries)},
		{Label: "Errors", Value: fmt.Sprintf("%d (%.1f%%)", analysis.ErrorCount, errorRate)},
		{Label: "Warnings", Value: fmt.Sprintf("%d (%.1f%%)", analysis.WarnCount, warningRate)},
	}

	// Add duration if available
	if !analysis.StartTime.IsZero() && !analysis.EndTime.IsZero() {
		duration := analysis.EndTime.Sub(analysis.StartTime)
		items = append(items, termfmt.TreeItem{Label: "Duration", Value: duration.String(), Last: true})
	} else {
		items = append(items, termfmt.TreeItem{Label: "Time Range", Value: "N/A", Last: true})
	}

	tree := termfmt.TreeViewWithOptions(items, f.opts)
	b.WriteString(tree + "\n\n")
}

// writeTopPatterns writes top patterns with visual indicators to match original format
func (f *terminalFormatter) writeTopPatterns(b *strings.Builder, patterns []analyzer.PatternMatch) {
	// Use fallback symbol to match original exactly
	opts := termfmt.DefaultOptions()
	opts.Emoji = false // Force fallback
	symbol := termfmt.GetEmoji("help", opts)
	b.WriteString(symbol + " Top Patterns\n")

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

// writeKeyInsights writes key insights with confidence indicators using go-termfmt
func (f *terminalFormatter) writeKeyInsights(b *strings.Builder, insights []analyzer.Insight) {
	symbol := termfmt.GetEmoji("insights", f.opts)
	b.WriteString(symbol + " Key Insights\n")

	// Create tree items for insights
	items := make([]termfmt.TreeItem, 0, len(insights))
	for i, insight := range insights {
		emoji := getSeverityEmoji(insight.Severity)
		confidenceBar := termfmt.CreateConfidenceBar(insight.Confidence, f.opts)

		item := termfmt.TreeItem{
			Label: fmt.Sprintf("%s %s", emoji, insight.Title),
			Value: fmt.Sprintf("(%.0f%% confidence)", insight.Confidence*100),
			Children: []termfmt.TreeItem{
				{Label: confidenceBar + " " + insight.Description, Value: ""},
			},
			Last: i == len(insights)-1,
		}
		items = append(items, item)
	}

	tree := termfmt.TreeViewWithOptions(items, f.opts)
	b.WriteString(tree + "\n\n")
}

// writeTextRecommendations writes recommendations for text format using go-termfmt
func (f *terminalFormatter) writeTextRecommendations(b *strings.Builder, analysis *analyzer.Analysis) {
	recommendations := generateRecommendations(analysis)

	symbol := termfmt.GetEmoji("recommendations", f.opts)
	b.WriteString(symbol + " Recommendations\n")

	for i, rec := range recommendations {
		if i < 3 { // Limit to top 3 recommendations for text format
			b.WriteString("• " + rec + "\n")
		}
	}
}

// writeHeader writes a beautiful header with box drawing to match original
func (f *terminalFormatter) writeHeader(b *strings.Builder) {
	header := "Log Analysis Summary"
	headerLen := len(header)

	b.WriteString("╔" + strings.Repeat("═", headerLen+2) + "╗\n")
	b.WriteString("║ " + header + " ║\n")
	b.WriteString("╚" + strings.Repeat("═", headerLen+2) + "╝\n\n")
}
