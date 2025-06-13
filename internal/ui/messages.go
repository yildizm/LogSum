package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
)

// Common message types shared across UI models
type analysisCompleteMsg struct {
	analysis *analyzer.Analysis
	entries  []*common.LogEntry
}

type analysisErrorMsg struct {
	err error
}

// CreateAnalysisCommand creates a tea command that performs analysis
func CreateAnalysisCommand(entries []*common.LogEntry, patterns []*common.Pattern) tea.Cmd {
	return func() tea.Msg {
		engine := analyzer.NewEngine()
		if len(patterns) > 0 {
			if err := engine.SetPatterns(patterns); err != nil {
				return analysisErrorMsg{err: err}
			}
		}

		ctx := context.Background()
		analysis, err := engine.Analyze(ctx, entries)
		if err != nil {
			return analysisErrorMsg{err: err}
		}

		return analysisCompleteMsg{
			analysis: analysis,
			entries:  entries,
		}
	}
}
