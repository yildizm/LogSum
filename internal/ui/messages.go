package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/parser"
)

// Common message types shared across UI models
type analysisCompleteMsg struct {
	analysis *analyzer.Analysis
	entries  []*parser.LogEntry
}

type analysisErrorMsg struct {
	err error
}

// CreateAnalysisCommand creates a tea command that performs analysis
func CreateAnalysisCommand(entries []*parser.LogEntry, patterns []*parser.Pattern) tea.Cmd {
	return func() tea.Msg {
		result := common.PerformAnalysis(entries, patterns)
		if result.Error != nil {
			return analysisErrorMsg{err: result.Error}
		}
		return analysisCompleteMsg{
			analysis: result.Analysis,
			entries:  result.Entries,
		}
	}
}
