package formatter

import (
	"fmt"
	"sort"
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

	// AI Analysis section (if available)
	f.writeAIAnalysis(&b, analysis)

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

	sort.Slice(sortedPatterns, func(i, j int) bool {
		return sortedPatterns[i].Count > sortedPatterns[j].Count
	})

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

// writeAIAnalysis writes AI-enhanced analysis results
func (f *terminalFormatter) writeAIAnalysis(b *strings.Builder, analysis *analyzer.Analysis) {
	if analysis.Context == nil {
		return
	}

	aiData := f.extractTerminalAIData(analysis.Context)
	if !aiData.hasAnyData() {
		return
	}

	f.writeTerminalAIHeader(b)
	f.writeTerminalAISummary(b, aiData.summary, aiData.hasSummary)
	f.writeTerminalRootCauses(b, aiData.rootCauses, aiData.hasRootCauses)
	f.writeTerminalRecommendations(b, aiData.recommendations, aiData.hasRecommendations)
}

// terminalAIData holds extracted AI analysis data for terminal formatting
type terminalAIData struct {
	summary            string
	rootCauses         interface{}
	recommendations    interface{}
	hasSummary         bool
	hasRootCauses      bool
	hasRecommendations bool
	hasErrorAnalysis   bool
}

// hasAnyData checks if any AI analysis data is available
func (data terminalAIData) hasAnyData() bool {
	return data.hasSummary || data.hasRootCauses || data.hasRecommendations || data.hasErrorAnalysis
}

// extractTerminalAIData extracts AI analysis data from context
func (f *terminalFormatter) extractTerminalAIData(context map[string]interface{}) terminalAIData {
	summary, hasSummary := context["ai_summary"].(string)
	rootCauses, hasRootCauses := context["root_causes"]
	recommendations, hasRecommendations := context["recommendations"]
	_, hasErrorAnalysis := context["error_analysis"]

	return terminalAIData{
		summary:            summary,
		rootCauses:         rootCauses,
		recommendations:    recommendations,
		hasSummary:         hasSummary,
		hasRootCauses:      hasRootCauses,
		hasRecommendations: hasRecommendations,
		hasErrorAnalysis:   hasErrorAnalysis,
	}
}

// writeTerminalAIHeader writes the AI analysis header
func (f *terminalFormatter) writeTerminalAIHeader(b *strings.Builder) {
	aiSymbol := termfmt.GetEmoji("ai", f.opts)
	if aiSymbol == "" {
		aiSymbol = "🤖" // Fallback
	}
	fmt.Fprintf(b, "\n%s AI Analysis\n", aiSymbol)
	b.WriteString(strings.Repeat("─", 50) + "\n\n")
}

// writeTerminalAISummary writes the AI summary section
func (f *terminalFormatter) writeTerminalAISummary(b *strings.Builder, summary string, hasSummary bool) {
	if hasSummary && summary != "" {
		summarySymbol := termfmt.GetEmoji("summary", f.opts)
		if summarySymbol == "" {
			summarySymbol = "📝" // Fallback
		}
		fmt.Fprintf(b, "%s Summary\n", summarySymbol)
		b.WriteString(summary + "\n\n")
	}
}

// writeTerminalRootCauses writes the root causes section
func (f *terminalFormatter) writeTerminalRootCauses(b *strings.Builder, rootCauses interface{}, hasRootCauses bool) {
	f.writeTerminalListSection(b, rootCauses, hasRootCauses, "target", "🎯", "Root Causes", f.buildRootCauseTreeItems)
}

// buildRootCauseTreeItems builds tree items for root causes
func (f *terminalFormatter) buildRootCauseTreeItems(rootCausesList []interface{}) []termfmt.TreeItem {
	items := make([]termfmt.TreeItem, 0, len(rootCausesList))
	for i, cause := range rootCausesList {
		causeMap, ok := cause.(map[string]interface{})
		if !ok {
			continue
		}

		title, ok := causeMap["title"].(string)
		if !ok {
			continue
		}

		var children []termfmt.TreeItem
		if desc, ok := causeMap["description"].(string); ok {
			children = append(children, termfmt.TreeItem{
				Label: "Description",
				Value: desc,
			})
		}
		if confidence, ok := causeMap["confidence"].(float64); ok {
			confidenceText := fmt.Sprintf("%.1f%%", confidence*100)
			children = append(children, termfmt.TreeItem{
				Label: "Confidence",
				Value: confidenceText,
			})
		}

		item := termfmt.TreeItem{
			Label:    fmt.Sprintf("Cause %d", i+1),
			Value:    title,
			Children: children,
		}
		items = append(items, item)
	}
	return items
}

// writeTerminalRecommendations writes the recommendations section
func (f *terminalFormatter) writeTerminalRecommendations(b *strings.Builder, recommendations interface{}, hasRecommendations bool) {
	f.writeTerminalListSection(b, recommendations, hasRecommendations, "recommendations", "💡", "AI Recommendations", f.buildRecommendationTreeItems)
}

// writeTerminalListSection is a generic helper to write list sections and eliminate code duplication
func (f *terminalFormatter) writeTerminalListSection(b *strings.Builder, data interface{}, hasData bool, emojiKey, fallbackEmoji, title string, itemBuilder func([]interface{}) []termfmt.TreeItem) {
	if !hasData {
		return
	}

	dataList, ok := data.([]interface{})
	if !ok || len(dataList) == 0 {
		return
	}

	symbol := termfmt.GetEmoji(emojiKey, f.opts)
	if symbol == "" {
		symbol = fallbackEmoji // Fallback
	}
	fmt.Fprintf(b, "%s %s\n", symbol, title)

	items := itemBuilder(dataList)
	if len(items) > 0 {
		tree := termfmt.TreeViewWithOptions(items, f.opts)
		b.WriteString(tree + "\n\n")
	}
}

// buildRecommendationTreeItems builds tree items for recommendations
func (f *terminalFormatter) buildRecommendationTreeItems(recList []interface{}) []termfmt.TreeItem {
	items := make([]termfmt.TreeItem, 0, len(recList))
	for i, rec := range recList {
		recMap, ok := rec.(map[string]interface{})
		if !ok {
			continue
		}

		title, ok := recMap["title"].(string)
		if !ok {
			continue
		}

		var children []termfmt.TreeItem
		if desc, ok := recMap["description"].(string); ok {
			children = append(children, termfmt.TreeItem{
				Label: "Description",
				Value: desc,
			})
		}
		if priority, ok := recMap["priority"].(string); ok {
			children = append(children, termfmt.TreeItem{
				Label: "Priority",
				Value: priority,
			})
		}

		item := termfmt.TreeItem{
			Label:    fmt.Sprintf("Recommendation %d", i+1),
			Value:    title,
			Children: children,
		}
		items = append(items, item)
	}
	return items
}
