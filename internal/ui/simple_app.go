package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/emoji"
	"github.com/yildizm/LogSum/internal/parser"
)

// Message types are now in messages.go for sharing

// SimpleModel represents a simplified TUI model for initial implementation
type SimpleModel struct {
	width     int
	height    int
	entries   []*parser.LogEntry
	patterns  []*parser.Pattern
	analysis  *analyzer.Analysis
	analyzing bool
	ready     bool
	quitting  bool
}

// NewSimpleModel creates a new simple model
func NewSimpleModel(entries []*parser.LogEntry, patterns []*parser.Pattern) *SimpleModel {
	return &SimpleModel{
		entries:  entries,
		patterns: patterns,
	}
}

// Init initializes the simple model
func (m *SimpleModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.startAnalysis(),
	)
}

// Update handles messages
func (m *SimpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case analysisCompleteMsg:
		m.analysis = msg.analysis
		m.analyzing = false

	case analysisErrorMsg:
		m.analyzing = false
	}

	return m, nil
}

// View renders the simple model
func (m *SimpleModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.quitting {
		return "Thanks for using LogSum! 👋\n"
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B82F6"))

	var content string

	switch {
	case m.analyzing:
		content = emoji.GetEmoji("pattern") + " Analyzing logs...\n\nPress 'q' to quit"
	case m.analysis != nil:
		content = fmt.Sprintf(`
LogSum Analysis Results

%s Total Entries: %d
%s Errors: %d
%s Warnings: %d
%s Patterns Found: %d
%s Insights: %d

Press 'q' to quit`,
			emoji.GetEmoji("statistics"), m.analysis.TotalEntries,
			emoji.GetEmoji("error"), m.analysis.ErrorCount,
			emoji.GetEmoji("warning"), 0, // TODO: Calculate warnings
			emoji.GetEmoji("pattern"), len(m.analysis.Patterns),
			emoji.GetEmoji("insight"), len(m.analysis.Insights))
	default:
		content = "No analysis available\n\nPress 'q' to quit"
	}

	return style.Render(content)
}

// startAnalysis starts the analysis process
func (m *SimpleModel) startAnalysis() tea.Cmd {
	m.analyzing = true
	return CreateAnalysisCommand(m.entries, m.patterns)
}

// SimpleRun runs the simplified TUI
func SimpleRun(entries []*parser.LogEntry, patterns []*parser.Pattern) error {
	model := NewSimpleModel(entries, patterns)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
