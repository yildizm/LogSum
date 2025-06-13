package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
)

// ColorfulModel represents an enhanced, colorful TUI model
type ColorfulModel struct {
	width     int
	height    int
	entries   []*common.LogEntry
	patterns  []*common.Pattern
	analysis  *analyzer.Analysis
	analyzing bool
	ready     bool
	quitting  bool

	// Animation state
	spinnerFrame int
	tick         int

	// Colors and styles
	primaryColor   lipgloss.AdaptiveColor
	secondaryColor lipgloss.AdaptiveColor
	successColor   lipgloss.AdaptiveColor
	warningColor   lipgloss.AdaptiveColor
	errorColor     lipgloss.AdaptiveColor
}

// NewColorfulModel creates a new colorful model
func NewColorfulModel(entries []*common.LogEntry, patterns []*common.Pattern) *ColorfulModel {
	return &ColorfulModel{
		entries:        entries,
		patterns:       patterns,
		primaryColor:   lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"},
		secondaryColor: lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"},
		successColor:   lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"},
		warningColor:   lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"},
		errorColor:     lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"},
	}
}

// Animation message
type tickMsg time.Time

// Animation command
func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init initializes the colorful model
func (m *ColorfulModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.startAnalysis(),
		tick(),
	)
}

// Update handles messages
func (m *ColorfulModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		m.tick++
		m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerChars)
		return m, tick()

	case analysisCompleteMsg:
		m.analysis = msg.analysis
		m.analyzing = false

	case analysisErrorMsg:
		m.analyzing = false
	}

	return m, nil
}

// Spinner characters
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// View renders the colorful model
func (m *ColorfulModel) View() string {
	if !m.ready {
		return m.renderLoadingScreen()
	}

	if m.quitting {
		return m.renderGoodbyeScreen()
	}

	if m.analyzing {
		return m.renderAnalyzingScreen()
	}

	if m.analysis != nil {
		return m.renderResultsScreen()
	}

	return m.renderErrorScreen()
}

func (m *ColorfulModel) renderLoadingScreen() string {
	loading := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render("Initializing LogSum...")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loading)
}

func (m *ColorfulModel) renderGoodbyeScreen() string {
	goodbye := lipgloss.NewStyle().
		Foreground(m.successColor).
		Bold(true).
		Render("Thanks for using LogSum! 👋✨")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, goodbye)
}

func (m *ColorfulModel) renderAnalyzingScreen() string {
	// Animated spinner
	spinner := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(spinnerChars[m.spinnerFrame])

	// Logo
	logo := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(`
╦  ╔═╗╔═╗╔═╗╦ ╦╔╦╗
║  ║ ║║ ╦╚═╗║ ║║║║
╩═╝╚═╝╚═╝╚═╝╚═╝╩ ╩`)

	// Status text with animation
	statusText := "High-Performance Log Analysis"
	dots := strings.Repeat(".", (m.tick/5)%4)
	status := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render(statusText + dots)

	// Progress info
	progress := lipgloss.NewStyle().
		Foreground(m.warningColor).
		Bold(true).
		Render(fmt.Sprintf("🔍 Analyzing %d log entries...", len(m.entries)))

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render("Press 'q' to quit")

	// Combine elements
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		fmt.Sprintf("%s %s", spinner, status),
		"",
		progress,
		"",
		instructions,
	)

	// Create colorful border
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.primaryColor).
		Padding(2, 4).
		Width(60)

	boxed := border.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxed)
}

func (m *ColorfulModel) renderResultsScreen() string {
	// Title with gradient effect
	title := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render("🚀 LogSum Analysis Results")

	// Stats with colors
	stats := []string{
		m.coloredStat("📊", "Total Entries", fmt.Sprintf("%d", m.analysis.TotalEntries), m.primaryColor),
		m.coloredStat("❌", "Errors", fmt.Sprintf("%d", m.analysis.ErrorCount), m.errorColor),
		m.coloredStat("⚠️", "Warnings", "0", m.warningColor), // TODO: Calculate warnings
		m.coloredStat("🔍", "Patterns Found", fmt.Sprintf("%d", len(m.analysis.Patterns)), m.successColor),
		m.coloredStat("💡", "Insights", fmt.Sprintf("%d", len(m.analysis.Insights)), m.primaryColor),
	}

	// Pattern list with colors
	var patternList []string
	if len(m.analysis.Patterns) > 0 {
		patternList = append(patternList, lipgloss.NewStyle().
			Foreground(m.primaryColor).
			Bold(true).
			Render("🎯 Detected Patterns:"))

		for i, pattern := range m.analysis.Patterns {
			if i >= 3 { // Limit to first 3 patterns
				patternList = append(patternList, lipgloss.NewStyle().
					Foreground(m.secondaryColor).
					Render(fmt.Sprintf("   ... and %d more", len(m.analysis.Patterns)-3)))
				break
			}

			patternColor := m.getPatternColor(pattern.Pattern.Type)
			patternText := lipgloss.NewStyle().
				Foreground(patternColor).
				Render(fmt.Sprintf("   • %s (%d matches)", pattern.Pattern.Name, pattern.Count))
			patternList = append(patternList, patternText)
		}
	}

	// Insights with colors
	var insightList []string
	if len(m.analysis.Insights) > 0 {
		insightList = append(insightList, "", lipgloss.NewStyle().
			Foreground(m.primaryColor).
			Bold(true).
			Render("🧠 Insights:"))

		for i, insight := range m.analysis.Insights {
			if i >= 2 { // Limit to first 2 insights
				insightList = append(insightList, lipgloss.NewStyle().
					Foreground(m.secondaryColor).
					Render(fmt.Sprintf("   ... and %d more", len(m.analysis.Insights)-2)))
				break
			}

			insightColor := m.getSeverityColor(insight.Severity)
			confidence := fmt.Sprintf("(%.0f%% confidence)", insight.Confidence*100)
			insightText := lipgloss.NewStyle().
				Foreground(insightColor).
				Render(fmt.Sprintf("   • %s %s", insight.Title, confidence))
			insightList = append(insightList, insightText)
		}
	}

	// Instructions with animation
	quitText := "Press 'q' to quit"
	if m.tick%20 < 10 {
		quitText = lipgloss.NewStyle().Foreground(m.warningColor).Render(quitText)
	} else {
		quitText = lipgloss.NewStyle().Foreground(m.secondaryColor).Render(quitText)
	}

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, stats...),
		"",
		lipgloss.JoinVertical(lipgloss.Left, patternList...),
		lipgloss.JoinVertical(lipgloss.Left, insightList...),
		"",
		quitText,
	)

	// Create rainbow border effect
	borderColor := m.getRainbowColor()
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 3).
		Width(minIntColorful(m.width-4, 80))

	boxed := border.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxed)
}

func (m *ColorfulModel) renderErrorScreen() string {
	errorMsg := lipgloss.NewStyle().
		Foreground(m.errorColor).
		Bold(true).
		Render("❌ No analysis data available")

	instructions := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render("Press 'q' to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		errorMsg,
		"",
		instructions,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// Helper functions

func (m *ColorfulModel) coloredStat(icon, label, value string, color lipgloss.AdaptiveColor) string {
	iconStyled := lipgloss.NewStyle().Render(icon)
	labelStyled := lipgloss.NewStyle().Foreground(m.secondaryColor).Render(label + ":")
	valueStyled := lipgloss.NewStyle().Foreground(color).Bold(true).Render(value)

	return fmt.Sprintf("%s %s %s", iconStyled, labelStyled, valueStyled)
}

func (m *ColorfulModel) getPatternColor(patternType common.PatternType) lipgloss.AdaptiveColor {
	switch patternType {
	case common.PatternTypeError:
		return m.errorColor
	case common.PatternTypeAnomaly:
		return m.warningColor
	case common.PatternTypePerformance:
		return lipgloss.AdaptiveColor{Light: "#F97316", Dark: "#FB923C"}
	case common.PatternTypeSecurity:
		return lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#EF4444"}
	default:
		return m.primaryColor
	}
}

func (m *ColorfulModel) getSeverityColor(severity common.LogLevel) lipgloss.AdaptiveColor {
	switch severity {
	case common.LevelError:
		return m.errorColor
	case common.LevelWarn:
		return m.warningColor
	case common.LevelInfo:
		return m.primaryColor
	default:
		return m.secondaryColor
	}
}

func (m *ColorfulModel) getRainbowColor() lipgloss.AdaptiveColor {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7", "#DDA0DD", "#98D8C8",
	}
	return lipgloss.AdaptiveColor{
		Light: colors[m.tick/10%len(colors)],
		Dark:  colors[m.tick/10%len(colors)],
	}
}

func (m *ColorfulModel) startAnalysis() tea.Cmd {
	m.analyzing = true
	return CreateAnalysisCommand(m.entries, m.patterns)
}

func minIntColorful(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ColorfulRun runs the enhanced colorful TUI
func ColorfulRun(entries []*common.LogEntry, patterns []*common.Pattern) error {
	model := NewColorfulModel(entries, patterns)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
