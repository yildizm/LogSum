package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/emoji"
)

// Message types for the interactive app
type analysisProgressMsg struct {
	step string
}

// InteractiveViewState represents different views in the interactive app
type InteractiveViewState int

const (
	InteractiveViewAnalyzing InteractiveViewState = iota
	InteractiveViewMainMenu
	InteractiveViewPatterns
	InteractiveViewInsights
	InteractiveViewLogs
	InteractiveViewHelp
)

// InteractiveModel represents a fully interactive TUI model
type InteractiveModel struct {
	width     int
	height    int
	entries   []*common.LogEntry
	patterns  []*common.Pattern
	analysis  *analyzer.Analysis
	analyzing bool
	ready     bool
	quitting  bool

	// Navigation state
	currentView   InteractiveViewState
	selectedIndex int
	maxIndex      int

	// Animation state
	spinnerFrame int
	tick         int
	analysisStep string

	// Colors and styles
	primaryColor   lipgloss.AdaptiveColor
	secondaryColor lipgloss.AdaptiveColor
	successColor   lipgloss.AdaptiveColor
	warningColor   lipgloss.AdaptiveColor
	errorColor     lipgloss.AdaptiveColor
	selectedColor  lipgloss.AdaptiveColor
}

// NewInteractiveModel creates a new interactive model
func NewInteractiveModel(entries []*common.LogEntry, patterns []*common.Pattern) *InteractiveModel {
	return &InteractiveModel{
		entries:        entries,
		patterns:       patterns,
		currentView:    InteractiveViewAnalyzing,
		analysisStep:   emoji.GetEmoji("rocket") + " Initializing LogSum...",
		primaryColor:   lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"},
		secondaryColor: lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"},
		successColor:   lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"},
		warningColor:   lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"},
		errorColor:     lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"},
		selectedColor:  lipgloss.AdaptiveColor{Light: "#DBEAFE", Dark: "#1E3A8A"},
	}
}

// Init initializes the interactive model
func (m *InteractiveModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.startAnalysis(),
		tick(),
	)
}

// Update handles messages and navigation
func (m *InteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tickMsg:
		return m.handleTick()
	case analysisCompleteMsg:
		return m.handleAnalysisComplete(msg)
	case analysisErrorMsg:
		return m.handleAnalysisError(msg)
	case analysisProgressMsg:
		return m.handleAnalysisProgress(msg)
	}

	return m, nil
}

// handleSelection handles enter key presses
func (m *InteractiveModel) handleSelection() (tea.Model, tea.Cmd) {
	if m.currentView == InteractiveViewMainMenu && m.analysis != nil {
		switch m.selectedIndex {
		case 0:
			m.currentView = InteractiveViewPatterns
		case 1:
			m.currentView = InteractiveViewInsights
		case 2:
			m.currentView = InteractiveViewLogs
		case 3:
			m.currentView = InteractiveViewHelp
		}
		m.selectedIndex = 0
		m.updateMaxIndex()
	}
	return m, nil
}

// updateMaxIndex updates the maximum selectable index for current view
func (m *InteractiveModel) updateMaxIndex() {
	switch m.currentView {
	case InteractiveViewMainMenu:
		m.maxIndex = 3 // 4 menu items (0-3)
	case InteractiveViewPatterns:
		m.maxIndex = max(0, len(m.analysis.Patterns)-1)
	case InteractiveViewInsights:
		m.maxIndex = max(0, len(m.analysis.Insights)-1)
	case InteractiveViewLogs:
		m.maxIndex = max(0, min(20, len(m.entries))-1) // Show up to 20 recent logs
	default:
		m.maxIndex = 0
	}
}

// View renders the interactive model
func (m *InteractiveModel) View() string {
	if !m.ready {
		return m.renderLoadingScreen()
	}

	if m.quitting {
		return m.renderGoodbyeScreen()
	}

	switch m.currentView {
	case InteractiveViewAnalyzing:
		return m.renderAnalyzingScreen()
	case InteractiveViewMainMenu:
		return m.renderMainMenu()
	case InteractiveViewPatterns:
		return m.renderPatternsView()
	case InteractiveViewInsights:
		return m.renderInsightsView()
	case InteractiveViewLogs:
		return m.renderLogsView()
	case InteractiveViewHelp:
		return m.renderHelpView()
	default:
		return m.renderMainMenu()
	}
}

func (m *InteractiveModel) renderLoadingScreen() string {
	loading := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render("Initializing LogSum...")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loading)
}

func (m *InteractiveModel) renderGoodbyeScreen() string {
	goodbye := lipgloss.NewStyle().
		Foreground(m.successColor).
		Bold(true).
		Render("Thanks for using LogSum! 👋✨")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, goodbye)
}

func (m *InteractiveModel) renderAnalyzingScreen() string {
	spinner := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(spinnerChars[m.spinnerFrame])

	logo := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(`
╦  ╔═╗╔═╗╔═╗╦ ╦╔╦╗
║  ║ ║║ ╦╚═╗║ ║║║║
╩═╝╚═╝╚═╝╚═╝╚═╝╩ ╩`)

	statusText := "High-Performance Log Analysis"
	dots := strings.Repeat(".", (m.tick/5)%4)
	status := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render(statusText + dots)

	// Current analysis step
	currentStep := lipgloss.NewStyle().
		Foreground(m.warningColor).
		Bold(true).
		Render(m.analysisStep)

	// Progress info
	progress := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Render(fmt.Sprintf(emoji.GetEmoji("statistics")+" Processing %d log entries", len(m.entries)))

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		fmt.Sprintf("%s %s", spinner, status),
		"",
		currentStep,
		"",
		progress,
	)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getRainbowColor()).
		Padding(2, 4).
		Width(60)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, border.Render(content))
}

func (m *InteractiveModel) renderMainMenu() string {
	title := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render("LogSum")

	if m.analysis == nil {
		errorMsg := lipgloss.NewStyle().
			Foreground(m.errorColor).
			Render(emoji.GetEmoji("error") + " No analysis data available")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center, title, "", errorMsg))
	}

	// Stats summary
	stats := fmt.Sprintf(
		emoji.GetEmoji("statistics")+" %d entries • "+emoji.GetEmoji("error")+" %d errors • "+emoji.GetEmoji("pattern")+" %d patterns • "+emoji.GetEmoji("insight")+" %d insights",
		m.analysis.TotalEntries,
		m.analysis.ErrorCount,
		len(m.analysis.Patterns),
		len(m.analysis.Insights),
	)

	statsStyled := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render(stats)

	// Menu options
	menuItems := []string{
		emoji.GetEmoji("pattern") + " View Patterns",
		emoji.GetEmoji("insight") + " View Insights",
		emoji.GetEmoji("recommendations") + " View Recent Logs",
		emoji.GetEmoji("help") + " Help",
	}

	menuList := make([]string, 0, len(menuItems))
	for i, item := range menuItems {
		style := lipgloss.NewStyle().Foreground(m.secondaryColor)
		prefix := "  "

		if i == m.selectedIndex {
			style = style.Background(m.selectedColor).Foreground(m.primaryColor).Bold(true)
			prefix = "▶ "
		}

		menuList = append(menuList, style.Render(prefix+item))
	}

	// Instructions
	instructions := []string{
		emoji.GetEmoji("target") + " Navigation: ↑↓ or j/k to move, Enter to select",
		emoji.GetEmoji("number") + " Quick keys: 1-Patterns, 2-Insights, 3-Logs, h-Help",
		emoji.GetEmoji("door") + " Exit: q to quit, Esc to go back",
	}

	instructionList := make([]string, 0, len(instructions))
	for _, instruction := range instructions {
		instructionList = append(instructionList, lipgloss.NewStyle().
			Foreground(m.secondaryColor).
			Render(instruction))
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		statsStyled,
		"",
		lipgloss.JoinVertical(lipgloss.Left, menuList...),
		"",
		lipgloss.JoinVertical(lipgloss.Left, instructionList...),
	)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getRainbowColor()).
		Padding(1, 3).
		Width(min(m.width-4, 80))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, border.Render(content))
}

func (m *InteractiveModel) renderPatternsView() string {
	title := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(emoji.GetEmoji("pattern") + " Pattern Matches")

	if len(m.analysis.Patterns) == 0 {
		noPatterns := lipgloss.NewStyle().
			Foreground(m.secondaryColor).
			Render("No patterns detected")

		content := lipgloss.JoinVertical(lipgloss.Center, title, "", noPatterns, "",
			lipgloss.NewStyle().Foreground(m.secondaryColor).Render("Press Esc to go back"))

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	patternList := make([]string, 0, len(m.analysis.Patterns)*2)
	for i, pattern := range m.analysis.Patterns {
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.selectedIndex {
			prefix = "▶ "
			style = style.Background(m.selectedColor).Foreground(m.primaryColor).Bold(true)
		} else {
			style = style.Foreground(m.getPatternColor(pattern.Pattern.Type))
		}

		text := fmt.Sprintf("%s%s (%d matches) - %s",
			prefix, pattern.Pattern.Name, pattern.Count, pattern.Pattern.Type)

		patternList = append(patternList, style.Render(text))

		// Show details for selected pattern
		if i == m.selectedIndex {
			details := fmt.Sprintf("    📝 %s", pattern.Pattern.Description)
			if pattern.Pattern.Regex != "" {
				details += fmt.Sprintf("\n    🔧 Regex: %s", pattern.Pattern.Regex)
			}
			if len(pattern.Pattern.Keywords) > 0 {
				details += fmt.Sprintf("\n    🔑 Keywords: %s", strings.Join(pattern.Pattern.Keywords, ", "))
			}

			patternList = append(patternList, lipgloss.NewStyle().
				Foreground(m.secondaryColor).
				Render(details))
		}
	}

	instructions := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render("↑↓ Navigate • Esc Back • q Quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, patternList...),
		"",
		instructions,
	)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.primaryColor).
		Padding(1, 2).
		Width(min(m.width-4, 100)).
		Height(min(m.height-4, 30))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, border.Render(content))
}

func (m *InteractiveModel) renderInsightsView() string {
	title := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(emoji.GetEmoji("insight") + " Insights")

	if len(m.analysis.Insights) == 0 {
		noInsights := lipgloss.NewStyle().
			Foreground(m.secondaryColor).
			Render("No insights generated")

		content := lipgloss.JoinVertical(lipgloss.Center, title, "", noInsights, "",
			lipgloss.NewStyle().Foreground(m.secondaryColor).Render("Press Esc to go back"))

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	insightList := make([]string, 0, len(m.analysis.Insights)*2)
	for i, insight := range m.analysis.Insights {
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.selectedIndex {
			prefix = "▶ "
			style = style.Background(m.selectedColor).Foreground(m.primaryColor).Bold(true)
		} else {
			style = style.Foreground(m.getSeverityColor(insight.Severity))
		}

		confidence := fmt.Sprintf("(%.0f%%)", insight.Confidence*100)
		text := fmt.Sprintf("%s%s %s", prefix, insight.Title, confidence)

		insightList = append(insightList, style.Render(text))

		// Show details for selected insight
		if i == m.selectedIndex {
			details := fmt.Sprintf("    "+emoji.GetEmoji("recommendations")+" %s", insight.Description)
			details += fmt.Sprintf("\n    "+emoji.GetEmoji("tag")+"  Type: %s", insight.Type)
			details += fmt.Sprintf("\n    "+emoji.GetEmoji("scale")+"  Severity: %s", insight.Severity)
			if len(insight.Evidence) > 0 {
				details += fmt.Sprintf("\n    "+emoji.GetEmoji("statistics")+" Evidence: %d log entries", len(insight.Evidence))
			}

			insightList = append(insightList, lipgloss.NewStyle().
				Foreground(m.secondaryColor).
				Render(details))
		}
	}

	instructions := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render("↑↓ Navigate • Esc Back • q Quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, insightList...),
		"",
		instructions,
	)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.primaryColor).
		Padding(1, 2).
		Width(min(m.width-4, 100)).
		Height(min(m.height-4, 30))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, border.Render(content))
}

func (m *InteractiveModel) renderLogsView() string {
	title := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(emoji.GetEmoji("recommendations") + " Recent Log Entries")

	// Show recent error/warning entries first
	var relevantEntries []*common.LogEntry
	for i := len(m.entries) - 1; i >= 0 && len(relevantEntries) < 20; i-- {
		entry := m.entries[i]
		if entry.LogLevel >= common.LevelWarn {
			relevantEntries = append(relevantEntries, entry)
		}
	}

	if len(relevantEntries) == 0 {
		noLogs := lipgloss.NewStyle().
			Foreground(m.secondaryColor).
			Render("No recent error/warning logs")

		content := lipgloss.JoinVertical(lipgloss.Center, title, "", noLogs, "",
			lipgloss.NewStyle().Foreground(m.secondaryColor).Render("Press Esc to go back"))

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	logList := make([]string, 0, len(relevantEntries))
	for i, entry := range relevantEntries {
		if i > m.maxIndex {
			break
		}

		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.selectedIndex {
			prefix = "▶ "
			style = style.Background(m.selectedColor).Foreground(m.primaryColor).Bold(true)
		} else {
			style = style.Foreground(m.getLogLevelColor(entry.LogLevel))
		}

		timestamp := ""
		if !entry.Timestamp.IsZero() {
			timestamp = entry.Timestamp.Format("15:04:05") + " "
		}

		message := entry.Message
		if len(message) > 80 {
			message = message[:77] + "..."
		}

		text := fmt.Sprintf("%s%s[%s] %s", prefix, timestamp, entry.LogLevel, message)
		logList = append(logList, style.Render(text))
	}

	instructions := lipgloss.NewStyle().
		Foreground(m.secondaryColor).
		Render("↑↓ Navigate • Esc Back • q Quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, logList...),
		"",
		instructions,
	)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.primaryColor).
		Padding(1, 2).
		Width(min(m.width-4, 120)).
		Height(min(m.height-4, 30))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, border.Render(content))
}

func (m *InteractiveModel) renderHelpView() string {
	title := lipgloss.NewStyle().
		Foreground(m.primaryColor).
		Bold(true).
		Render(emoji.GetEmoji("help") + " LogSum Help")

	helpSections := []string{
		"🎯 Navigation:",
		"  ↑↓ or j/k    Move up/down in lists",
		"  Enter or Space    Select item",
		"  Esc    Go back to main menu",
		"  m    Return to main menu",
		"",
		"🔢 Quick Keys:",
		"  1    View Patterns",
		"  2    View Insights",
		"  3    View Recent Logs",
		"  h or ?    Show this help",
		"",
		"🚪 Exit:",
		"  q    Quit application",
		"  Ctrl+C    Force quit",
		"",
		"💡 About LogSum:",
		"  High-performance log analysis tool",
		"  Detects patterns and generates insights",
		"  Built with Bubble Tea for beautiful TUI",
	}

	var helpList []string
	for _, line := range helpSections {
		switch {
		case strings.HasPrefix(line, "🎯"), strings.HasPrefix(line, "🔢"),
			strings.HasPrefix(line, "🚪"), strings.HasPrefix(line, "💡"):
			helpList = append(helpList, lipgloss.NewStyle().
				Foreground(m.primaryColor).
				Bold(true).
				Render(line))
		case line == "":
			helpList = append(helpList, "")
		default:
			helpList = append(helpList, lipgloss.NewStyle().
				Foreground(m.secondaryColor).
				Render(line))
		}
	}

	instructions := lipgloss.NewStyle().
		Foreground(m.warningColor).
		Bold(true).
		Render("Press Esc to go back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, helpList...),
		"",
		instructions,
	)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.primaryColor).
		Padding(1, 2).
		Width(min(m.width-4, 80))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, border.Render(content))
}

// Helper functions (same as before)
func (m *InteractiveModel) getPatternColor(patternType common.PatternType) lipgloss.AdaptiveColor {
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

func (m *InteractiveModel) getSeverityColor(severity common.LogLevel) lipgloss.AdaptiveColor {
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

func (m *InteractiveModel) getLogLevelColor(level common.LogLevel) lipgloss.AdaptiveColor {
	switch level {
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

func (m *InteractiveModel) getRainbowColor() lipgloss.AdaptiveColor {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7", "#DDA0DD", "#98D8C8",
	}
	return lipgloss.AdaptiveColor{
		Light: colors[m.tick/10%len(colors)],
		Dark:  colors[m.tick/10%len(colors)],
	}
}

func (m *InteractiveModel) startAnalysis() tea.Cmd {
	m.analyzing = true

	return tea.Sequence(
		// Step 1: Initialize
		func() tea.Msg {
			return analysisProgressMsg{step: "📋 Loading pattern definitions..."}
		},

		// Step 2: Load patterns
		func() tea.Msg {
			return analysisProgressMsg{step: "🔍 Parsing log entries..."}
		},

		// Step 3: Parse logs
		func() tea.Msg {
			return analysisProgressMsg{step: "🎯 Detecting patterns..."}
		},

		// Step 4: Pattern matching
		func() tea.Msg {
			return analysisProgressMsg{step: "🧠 Generating insights..."}
		},

		// Step 5: Insight generation
		func() tea.Msg {
			return analysisProgressMsg{step: "📊 Building timeline..."}
		},

		// Step 6: Final analysis
		func() tea.Msg {

			engine := analyzer.NewEngine()
			if len(m.patterns) > 0 {
				if err := engine.SetPatterns(m.patterns); err != nil {
					return analysisErrorMsg{err: err}
				}
			}

			ctx := context.Background()
			analysis, err := engine.Analyze(ctx, m.entries)
			if err != nil {
				return analysisErrorMsg{err: err}
			}

			// Final completion step
			time.Sleep(300 * time.Millisecond)

			return analysisCompleteMsg{
				analysis: analysis,
				entries:  m.entries,
			}
		},
	)
}

// Handler functions for Update method

// handleWindowResize handles window resize events
func (m *InteractiveModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true
	return m, nil
}

// handleKeyPress handles keyboard input
func (m *InteractiveModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m.handleQuit()
	case "esc":
		return m.handleEscape()
	case "h", "?":
		return m.handleHelp()
	case "up", "k":
		return m.handleMoveUp()
	case "down", "j":
		return m.handleMoveDown()
	case "enter", " ":
		return m.handleSelection()
	case "1", "2", "3", "m":
		return m.handleNumberKey(msg.String())
	}
	return m, nil
}

// handleQuit handles quit commands
func (m *InteractiveModel) handleQuit() (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}

// handleEscape handles escape key
func (m *InteractiveModel) handleEscape() (tea.Model, tea.Cmd) {
	if m.currentView != InteractiveViewMainMenu && m.currentView != InteractiveViewAnalyzing {
		m.currentView = InteractiveViewMainMenu
		m.selectedIndex = 0
		m.updateMaxIndex()
	}
	return m, nil
}

// handleHelp handles help key
func (m *InteractiveModel) handleHelp() (tea.Model, tea.Cmd) {
	if m.analysis != nil {
		m.currentView = InteractiveViewHelp
	}
	return m, nil
}

// handleMoveUp handles up movement
func (m *InteractiveModel) handleMoveUp() (tea.Model, tea.Cmd) {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
	return m, nil
}

// handleMoveDown handles down movement
func (m *InteractiveModel) handleMoveDown() (tea.Model, tea.Cmd) {
	if m.selectedIndex < m.maxIndex {
		m.selectedIndex++
	}
	return m, nil
}

// handleNumberKey handles numbered shortcuts
func (m *InteractiveModel) handleNumberKey(key string) (tea.Model, tea.Cmd) {
	if m.analysis == nil {
		return m, nil
	}

	switch key {
	case "1":
		m.currentView = InteractiveViewPatterns
	case "2":
		m.currentView = InteractiveViewInsights
	case "3":
		m.currentView = InteractiveViewLogs
	case "m":
		m.currentView = InteractiveViewMainMenu
	}

	m.selectedIndex = 0
	m.updateMaxIndex()
	return m, nil
}

// handleTick handles timer ticks
func (m *InteractiveModel) handleTick() (tea.Model, tea.Cmd) {
	m.tick++
	m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerChars)
	return m, tick()
}

// handleAnalysisComplete handles analysis completion
func (m *InteractiveModel) handleAnalysisComplete(msg analysisCompleteMsg) (tea.Model, tea.Cmd) {
	m.analysis = msg.analysis
	m.analyzing = false
	m.currentView = InteractiveViewMainMenu
	m.updateMaxIndex()
	return m, nil
}

// handleAnalysisError handles analysis errors
func (m *InteractiveModel) handleAnalysisError(_ analysisErrorMsg) (tea.Model, tea.Cmd) {
	m.analyzing = false
	m.currentView = InteractiveViewMainMenu
	return m, nil
}

// handleAnalysisProgress handles analysis progress updates
func (m *InteractiveModel) handleAnalysisProgress(msg analysisProgressMsg) (tea.Model, tea.Cmd) {
	m.analysisStep = msg.step
	return m, nil
}

// InteractiveRun runs the fully interactive TUI
func InteractiveRun(entries []*common.LogEntry, patterns []*common.Pattern) error {
	model := NewInteractiveModel(entries, patterns)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
