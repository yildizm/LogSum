package components

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/analyzer"
)

// TimelineChart represents a timeline visualization component
type TimelineChart struct {
	Title    string
	Timeline *analyzer.Timeline
	Width    int
	Height   int
	ShowAxis bool
}

// NewTimelineChart creates a new timeline chart
func NewTimelineChart(title string, timeline *analyzer.Timeline, width, height int) *TimelineChart {
	return &TimelineChart{
		Title:    title,
		Timeline: timeline,
		Width:    width,
		Height:   height,
		ShowAxis: true,
	}
}

// Render renders the timeline chart
func (t *TimelineChart) Render() string {
	if t.Timeline == nil || len(t.Timeline.Buckets) == 0 {
		return t.renderEmpty()
	}

	var content []string

	// Add title
	content = append(content, lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}).Bold(true).Render(t.Title), "")

	// Render chart
	chart := t.renderChart()
	content = append(content, chart)

	// Add time axis if enabled
	if t.ShowAxis {
		axis := t.renderTimeAxis()
		content = append(content, axis)
	}

	// Add summary
	summary := t.renderSummary()
	content = append(content, "", summary)

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Padding(1).Width(t.Width).Render(joined)
}

// renderEmpty renders an empty timeline
func (t *TimelineChart) renderEmpty() string {

	content := []string{
		lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}).Bold(true).Render(t.Title),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render("No timeline data available"),
	}

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Padding(1).Width(t.Width).Render(joined)
}

// renderChart renders the main chart area
func (t *TimelineChart) renderChart() string {

	chartWidth := t.Width - 10  // Leave space for Y-axis labels
	chartHeight := t.Height - 8 // Leave space for title, axis, and summary

	if chartHeight < 3 {
		chartHeight = 3
	}

	// Find max values for scaling
	maxTotal := 0
	maxErrors := 0
	for _, bucket := range t.Timeline.Buckets {
		if bucket.EntryCount > maxTotal {
			maxTotal = bucket.EntryCount
		}
		if bucket.ErrorCount > maxErrors {
			maxErrors = bucket.ErrorCount
		}
	}

	if maxTotal == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render("No data to display")
	}

	var lines []string

	// Render chart from top to bottom
	for row := chartHeight - 1; row >= 0; row-- {
		var line strings.Builder

		// Y-axis label
		value := int(float64(maxTotal) * float64(row) / float64(chartHeight-1))
		label := fmt.Sprintf("%4d │", value)
		line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render(label))

		// Chart bars
		for i, bucket := range t.Timeline.Buckets {
			if i >= chartWidth {
				break
			}

			// Calculate bar heights
			totalHeight := int(float64(bucket.EntryCount) * float64(chartHeight-1) / float64(maxTotal))
			errorHeight := int(float64(bucket.ErrorCount) * float64(chartHeight-1) / float64(maxTotal))

			char := " "
			style := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})

			if row <= totalHeight {
				if row <= errorHeight {
					// Error portion
					char = "█"
					style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"})
				} else {
					// Normal portion
					char = "█"
					style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"})
				}
			}

			line.WriteString(style.Render(char))
		}

		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

// renderTimeAxis renders the time axis labels
func (t *TimelineChart) renderTimeAxis() string {

	chartWidth := t.Width - 10

	var line strings.Builder
	line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render("     └"))

	// Add horizontal line
	for i := 0; i < chartWidth; i++ {
		line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render("─"))
	}

	// Time labels
	var timeLabels []string
	timeLabels = append(timeLabels, lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render("     "))

	// Show time labels for key points
	numLabels := min(5, len(t.Timeline.Buckets))
	step := len(t.Timeline.Buckets) / numLabels
	if step == 0 {
		step = 1
	}

	for i := 0; i < len(t.Timeline.Buckets); i += step {
		if i >= chartWidth {
			break
		}

		bucket := t.Timeline.Buckets[i]
		timeStr := bucket.Start.Format("15:04")

		// Position the label
		pos := i * chartWidth / len(t.Timeline.Buckets)
		for len(timeLabels)-1 < pos {
			timeLabels = append(timeLabels, " ")
		}

		if pos < len(timeLabels) {
			timeLabels[pos] = timeStr
		}
	}

	timeAxis := strings.Join(timeLabels, "")

	return line.String() + "\n" + lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render(timeAxis)
}

// renderSummary renders timeline summary information
func (t *TimelineChart) renderSummary() string {

	totalEntries := 0
	totalErrors := 0
	for _, bucket := range t.Timeline.Buckets {
		totalEntries += bucket.EntryCount
		totalErrors += bucket.ErrorCount
	}

	duration := t.Timeline.Buckets[len(t.Timeline.Buckets)-1].End.Sub(t.Timeline.Buckets[0].Start)
	bucketDuration := t.Timeline.BucketSize

	summary := []string{
		fmt.Sprintf("Duration: %s", formatDuration(duration)),
		fmt.Sprintf("Buckets: %d (%s each)", len(t.Timeline.Buckets), formatDuration(bucketDuration)),
		fmt.Sprintf("Total: %d entries, %d errors", totalEntries, totalErrors),
	}

	if totalEntries > 0 {
		errorRate := float64(totalErrors) / float64(totalEntries) * 100
		summary = append(summary, fmt.Sprintf("Error rate: %.1f%%", errorRate))
	}

	return lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).Render(strings.Join(summary, " | "))
}

// SparklineChart represents a compact sparkline chart
type SparklineChart struct {
	Values []float64
	Width  int
	Min    float64
	Max    float64
}

// NewSparklineChart creates a new sparkline chart
func NewSparklineChart(values []float64, width int) *SparklineChart {
	minVal := math.Inf(1)
	maxVal := math.Inf(-1)

	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	return &SparklineChart{
		Values: values,
		Width:  width,
		Min:    minVal,
		Max:    maxVal,
	}
}

// Render renders the sparkline chart
func (s *SparklineChart) Render() string {
	if len(s.Values) == 0 {
		return ""
	}

	// Sparkline characters (from lowest to highest)
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	var result strings.Builder

	step := len(s.Values) / s.Width
	if step == 0 {
		step = 1
	}

	for i := 0; i < s.Width && i*step < len(s.Values); i++ {
		value := s.Values[i*step]

		// Normalize value to 0-1 range
		normalized := 0.0
		if s.Max > s.Min {
			normalized = (value - s.Min) / (s.Max - s.Min)
		}

		// Map to character index
		charIndex := int(normalized * float64(len(chars)-1))
		if charIndex >= len(chars) {
			charIndex = len(chars) - 1
		}

		result.WriteString(chars[charIndex])
	}

	return result.String()
}

// ErrorRateChart creates a chart showing error rates over time
func NewErrorRateChart(timeline *analyzer.Timeline, width, height int) *TimelineChart {
	return NewTimelineChart("Error Rate Over Time", timeline, width, height)
}

// VolumeChart creates a chart showing log volume over time
func NewVolumeChart(timeline *analyzer.Timeline, width, height int) *TimelineChart {
	return NewTimelineChart("Log Volume Over Time", timeline, width, height)
}

// Helper functions

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.1fm", d.Minutes())
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1fh", d.Hours())
	default:
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}
