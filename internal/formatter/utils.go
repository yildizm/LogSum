package formatter

import (
	"fmt"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/go-termfmt"
)

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

// getPatternEmoji returns emoji for pattern types using go-termfmt
func getPatternEmoji(patternType common.PatternType) string {
	opts := termfmt.DefaultOptions()
	switch patternType {
	case common.PatternTypeError:
		return termfmt.GetEmoji("error_pattern", opts)
	case common.PatternTypeAnomaly:
		return termfmt.GetEmoji("anomaly_pattern", opts)
	case common.PatternTypePerformance:
		return termfmt.GetEmoji("perf_pattern", opts)
	case common.PatternTypeSecurity:
		return termfmt.GetEmoji("security_pattern", opts)
	default:
		return termfmt.GetEmoji("pattern", opts)
	}
}

// getSeverityEmoji returns emoji for severity levels using go-termfmt
func getSeverityEmoji(severity common.LogLevel) string {
	opts := termfmt.DefaultOptions()
	switch severity {
	case common.LevelFatal, common.LevelError:
		return termfmt.GetEmoji("error", opts)
	case common.LevelWarn:
		return termfmt.GetEmoji("warning", opts)
	case common.LevelInfo:
		return termfmt.GetEmoji("info", opts)
	default:
		return termfmt.GetEmoji("insight", opts)
	}
}

// createConfidenceBar creates ASCII confidence bar using go-termfmt
func createConfidenceBar(confidence float64) string {
	opts := termfmt.DefaultOptions()
	return termfmt.CreateConfidenceBar(confidence, opts)
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
		case common.PatternTypeError:
			if match.Count > 5 {
				recommendations = append(recommendations,
					fmt.Sprintf("Address recurring %s pattern (%d occurrences)",
						match.Pattern.Name, match.Count))
			}
		case common.PatternTypePerformance:
			recommendations = append(recommendations,
				fmt.Sprintf("Optimize performance issues related to %s",
					match.Pattern.Name))
		case common.PatternTypeSecurity:
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
