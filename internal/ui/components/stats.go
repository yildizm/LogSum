package components

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
)

// StatsCard represents a statistics card component
type StatsCard struct {
	Title       string
	Value       string
	Description string
	Status      string // "success", "warning", "error", "info"
	Icon        string
	Width       int
	Height      int
}

// NewStatsCard creates a new stats card
func NewStatsCard(title, value, description string) *StatsCard {
	return &StatsCard{
		Title:       title,
		Value:       value,
		Description: description,
		Status:      "info",
		Width:       20,
		Height:      4,
	}
}

// SetStatus sets the status color of the card
func (s *StatsCard) SetStatus(status string) *StatsCard {
	s.Status = status
	return s
}

// SetIcon sets the icon for the card
func (s *StatsCard) SetIcon(icon string) *StatsCard {
	s.Icon = icon
	return s
}

// SetSize sets the size of the card
func (s *StatsCard) SetSize(width, height int) *StatsCard {
	s.Width = width
	s.Height = height
	return s
}

// Render renders the stats card
func (s *StatsCard) Render() string {
	successColor := lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"}
	warningColor := lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"}
	errorColor := lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"}
	infoColor := lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}
	bodyColor := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}

	// Choose style based on status
	var valueStyle lipgloss.Style
	switch s.Status {
	case "success":
		valueStyle = lipgloss.NewStyle().Foreground(successColor)
	case "warning":
		valueStyle = lipgloss.NewStyle().Foreground(warningColor)
	case "error":
		valueStyle = lipgloss.NewStyle().Foreground(errorColor)
	case "info":
		valueStyle = lipgloss.NewStyle().Foreground(infoColor)
	default:
		valueStyle = lipgloss.NewStyle().Foreground(bodyColor)
	}

	// Format content
	titleStyle := lipgloss.NewStyle().Foreground(infoColor).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(bodyColor)
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bodyColor).Padding(1)

	title := titleStyle.Render(s.Title)
	if s.Icon != "" {
		title = s.Icon + " " + title
	}

	value := valueStyle.Bold(true).Render(s.Value)
	description := mutedStyle.Render(s.Description)

	// Create content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		value,
		description,
	)

	// Apply box styling
	return boxStyle.
		Width(s.Width).
		Height(s.Height).
		Render(content)
}

// StatsDashboard represents a collection of stats cards
type StatsDashboard struct {
	cards      []*StatsCard
	columns    int
	cardWidth  int
	cardHeight int
}

// NewStatsDashboard creates a new stats dashboard
func NewStatsDashboard(columns int) *StatsDashboard {
	return &StatsDashboard{
		columns:    columns,
		cardWidth:  20,
		cardHeight: 4,
	}
}

// AddCard adds a stats card to the dashboard
func (d *StatsDashboard) AddCard(card *StatsCard) {
	card.SetSize(d.cardWidth, d.cardHeight)
	d.cards = append(d.cards, card)
}

// SetCardSize sets the default size for all cards
func (d *StatsDashboard) SetCardSize(width, height int) {
	d.cardWidth = width
	d.cardHeight = height
	for _, card := range d.cards {
		card.SetSize(width, height)
	}
}

// Render renders the stats dashboard
func (d *StatsDashboard) Render() string {
	if len(d.cards) == 0 {
		return ""
	}

	var rows []string

	// Group cards into rows
	for i := 0; i < len(d.cards); i += d.columns {
		end := i + d.columns
		if end > len(d.cards) {
			end = len(d.cards)
		}

		var rowCards []string
		for j := i; j < end; j++ {
			rowCards = append(rowCards, d.cards[j].Render())
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top, rowCards...)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// CreateAnalysisStats creates stats cards from analysis results
func CreateAnalysisStats(analysis *analyzer.Analysis) *StatsDashboard {
	dashboard := NewStatsDashboard(4)

	// Total entries card
	totalCard := NewStatsCard(
		"Total Entries",
		formatNumber(analysis.TotalEntries),
		"Log entries processed",
	).SetIcon("ℹ️").SetStatus("info")
	dashboard.AddCard(totalCard)

	// Errors card
	errorStatus := "success"
	if analysis.ErrorCount > 0 {
		errorRate := float64(analysis.ErrorCount) / float64(analysis.TotalEntries) * 100
		if errorRate > 10 {
			errorStatus = "error"
		} else if errorRate > 5 {
			errorStatus = "warning"
		}
	}

	errorCard := NewStatsCard(
		"Errors",
		formatNumber(analysis.ErrorCount),
		fmt.Sprintf("%.1f%% error rate", float64(analysis.ErrorCount)/float64(analysis.TotalEntries)*100),
	).SetIcon("❌").SetStatus(errorStatus)
	dashboard.AddCard(errorCard)

	// Warnings card
	warningCard := NewStatsCard(
		"Warnings",
		formatNumber(analysis.WarnCount),
		"Warning messages",
	).SetIcon("⚠️").SetStatus("warning")
	dashboard.AddCard(warningCard)

	// Patterns card
	patternCard := NewStatsCard(
		"Patterns",
		formatNumber(len(analysis.Patterns)),
		"Pattern matches found",
	).SetIcon("🔍").SetStatus("info")
	dashboard.AddCard(patternCard)

	// Time range card
	timeCard := NewStatsCard(
		"Time Range",
		formatTimeRange(analysis.StartTime, analysis.EndTime),
		"Log time span",
	).SetIcon("⏰").SetStatus("info")
	dashboard.AddCard(timeCard)

	// Insights card
	insightStatus := "info"
	if len(analysis.Insights) > 0 {
		// Check for high-severity insights
		for _, insight := range analysis.Insights {
			if insight.Severity >= common.LevelError {
				insightStatus = "error"
				break
			} else if insight.Severity >= common.LevelWarn {
				insightStatus = "warning"
			}
		}
	}

	insightCard := NewStatsCard(
		"Insights",
		formatNumber(len(analysis.Insights)),
		"Smart analysis insights",
	).SetIcon("💡").SetStatus(insightStatus)
	dashboard.AddCard(insightCard)

	return dashboard
}

// formatNumber formats large numbers with commas
func formatNumber(n int) string {
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// formatTimeRange formats a time range for display
func formatTimeRange(start, end interface{}) string {
	// Simplified time range display
	return "Multiple logs"
}

// SummaryBox creates a summary information box
type SummaryBox struct {
	Title   string
	Content []string
	Width   int
}

// NewSummaryBox creates a new summary box
func NewSummaryBox(title string, width int) *SummaryBox {
	return &SummaryBox{
		Title: title,
		Width: width,
	}
}

// AddLine adds a line to the summary
func (s *SummaryBox) AddLine(line string) {
	s.Content = append(s.Content, line)
}

// AddKeyValue adds a key-value pair to the summary
func (s *SummaryBox) AddKeyValue(key, value string) {
	s.Content = append(s.Content, fmt.Sprintf("%-15s: %s", key, value))
}

// Render renders the summary box
func (s *SummaryBox) Render() string {
	headerColor := lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}
	bodyColor := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}

	headerStyle := lipgloss.NewStyle().Foreground(headerColor).Bold(true)
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bodyColor).Padding(1)

	title := headerStyle.Render(s.Title)

	content := make([]string, 0, len(s.Content)+2)
	content = append(content, title, "")

	bodyStyle := lipgloss.NewStyle().Foreground(bodyColor)

	for _, line := range s.Content {
		content = append(content, bodyStyle.Render(line))
	}

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)

	return boxStyle.Width(s.Width).Render(joined)
}
