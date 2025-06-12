package common

import (
	"context"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/parser"
)

// AnalysisResult represents the result of log analysis
type AnalysisResult struct {
	Analysis *analyzer.Analysis
	Entries  []*parser.LogEntry
	Error    error
}

// PerformAnalysis performs log analysis with the given entries and patterns
func PerformAnalysis(entries []*parser.LogEntry, patterns []*parser.Pattern) AnalysisResult {
	engine := analyzer.NewEngine()
	if len(patterns) > 0 {
		if err := engine.SetPatterns(patterns); err != nil {
			return AnalysisResult{Error: err}
		}
	}

	ctx := context.Background()
	analysis, err := engine.Analyze(ctx, entries)
	if err != nil {
		return AnalysisResult{Error: err}
	}

	return AnalysisResult{
		Analysis: analysis,
		Entries:  entries,
	}
}
