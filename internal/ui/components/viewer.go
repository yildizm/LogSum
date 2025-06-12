package components

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/parser"
	"github.com/yildizm/LogSum/internal/ui"
)

// LogViewer represents a log entry viewer component
type LogViewer struct {
	Title         string
	Entries       []*parser.LogEntry
	CurrentIndex  int
	Width         int
	Height        int
	ShowLineNo    bool
	ShowTimestamp bool
	ShowLevel     bool
	ShowService   bool
	Highlight     []string // Terms to highlight
	contextLines  int      // Number of context lines to show
}

// NewLogViewer creates a new log viewer
func NewLogViewer(title string, width, height int) *LogViewer {
	return &LogViewer{
		Title:         title,
		Width:         width,
		Height:        height,
		ShowLineNo:    true,
		ShowTimestamp: true,
		ShowLevel:     true,
		ShowService:   true,
		contextLines:  2,
	}
}

// SetEntries sets the log entries to display
func (v *LogViewer) SetEntries(entries []*parser.LogEntry) {
	v.Entries = entries
	v.CurrentIndex = 0
}

// SetHighlight sets terms to highlight in the logs
func (v *LogViewer) SetHighlight(terms []string) {
	v.Highlight = terms
}

// SetContextLines sets the number of context lines to show
func (v *LogViewer) SetContextLines(lines int) {
	v.contextLines = lines
}

// Next moves to the next entry
func (v *LogViewer) Next() bool {
	if v.CurrentIndex < len(v.Entries)-1 {
		v.CurrentIndex++
		return true
	}
	return false
}

// Previous moves to the previous entry
func (v *LogViewer) Previous() bool {
	if v.CurrentIndex > 0 {
		v.CurrentIndex--
		return true
	}
	return false
}

// GetCurrentEntry returns the current entry
func (v *LogViewer) GetCurrentEntry() *parser.LogEntry {
	if v.CurrentIndex >= 0 && v.CurrentIndex < len(v.Entries) {
		return v.Entries[v.CurrentIndex]
	}
	return nil
}

// Render renders the log viewer
func (v *LogViewer) Render() string {
	if len(v.Entries) == 0 {
		return v.renderEmpty()
	}

	styles := ui.GetStyles()

	var content []string

	// Add title and navigation info
	title := fmt.Sprintf("%s (%d/%d)", v.Title, v.CurrentIndex+1, len(v.Entries))
	content = append(content, styles.Header.Render(title), "")

	// Render current entry with context
	entryContent := v.renderEntryWithContext()
	content = append(content, entryContent...)

	// Add navigation help
	if len(v.Entries) > 1 {
		help := "Use ←/→ or h/l to navigate entries"
		content = append(content, "", styles.Muted.Render(help))
	}

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)
	return styles.Panel.Width(v.Width).Render(joined)
}

// renderEmpty renders an empty log viewer
func (v *LogViewer) renderEmpty() string {
	styles := ui.GetStyles()

	content := []string{
		styles.Header.Render(v.Title),
		"",
		styles.Muted.Render("No log entries to display"),
	}

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)
	return styles.Panel.Width(v.Width).Render(joined)
}

// renderEntryWithContext renders the current entry with surrounding context
func (v *LogViewer) renderEntryWithContext() []string {
	var lines []string

	// Determine range to show
	start := max(0, v.CurrentIndex-v.contextLines)
	end := min(len(v.Entries), v.CurrentIndex+v.contextLines+1)

	for i := start; i < end; i++ {
		entry := v.Entries[i]
		isCurrent := i == v.CurrentIndex

		line := v.renderLogEntry(entry, isCurrent)
		lines = append(lines, line)
	}

	return lines
}

// renderLogEntry renders a single log entry
func (v *LogViewer) renderLogEntry(entry *parser.LogEntry, highlight bool) string {
	styles := ui.GetStyles()

	var parts []string

	// Line number
	if v.ShowLineNo && entry.LineNumber > 0 {
		lineNo := fmt.Sprintf("%4d", entry.LineNumber)
		if highlight {
			parts = append(parts, styles.Selected.Render(lineNo))
		} else {
			parts = append(parts, styles.Muted.Render(lineNo))
		}
	}

	// Timestamp
	if v.ShowTimestamp && !entry.Timestamp.IsZero() {
		timestamp := entry.Timestamp.Format("15:04:05.000")
		if highlight {
			parts = append(parts, styles.Selected.Render(timestamp))
		} else {
			parts = append(parts, styles.Muted.Render(timestamp))
		}
	}

	// Level
	if v.ShowLevel {
		level := v.formatLevel(entry.Level)
		if highlight {
			parts = append(parts, styles.Selected.Render(level))
		} else {
			parts = append(parts, level)
		}
	}

	// Service
	if v.ShowService && entry.Service != "" {
		service := fmt.Sprintf("[%s]", entry.Service)
		if highlight {
			parts = append(parts, styles.Selected.Render(service))
		} else {
			parts = append(parts, styles.Muted.Render(service))
		}
	}

	// Message
	message := v.formatMessage(entry.Message, highlight)
	parts = append(parts, message)

	line := strings.Join(parts, " ")

	// Apply overall highlighting if this is the current entry
	if highlight {
		return styles.Highlight.Width(v.Width - 4).Render(line)
	}

	return line
}

// formatLevel formats and colors log levels
func (v *LogViewer) formatLevel(level parser.LogLevel) string {
	styles := ui.GetStyles()

	levelStr := strings.ToUpper(level.String())
	levelStr = fmt.Sprintf("%-5s", levelStr) // Pad to 5 characters

	switch level {
	case parser.LevelError:
		return styles.Error.Render(levelStr)
	case parser.LevelWarn:
		return styles.Warning.Render(levelStr)
	case parser.LevelInfo:
		return styles.Info.Render(levelStr)
	case parser.LevelDebug:
		return styles.Muted.Render(levelStr)
	default:
		return styles.Body.Render(levelStr)
	}
}

// formatMessage formats the log message with highlighting
func (v *LogViewer) formatMessage(message string, isCurrentEntry bool) string {
	styles := ui.GetStyles()

	// Apply syntax highlighting for highlight terms
	if len(v.Highlight) > 0 {
		for _, term := range v.Highlight {
			if term == "" {
				continue
			}

			// Create case-insensitive regex
			pattern := regexp.QuoteMeta(term)
			re, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				continue
			}

			// Replace matches with highlighted version
			message = re.ReplaceAllStringFunc(message, func(match string) string {
				if isCurrentEntry {
					return styles.Selected.Bold(true).Render(match)
				}
				return styles.Highlight.Bold(true).Render(match)
			})
		}
	}

	return message
}

// DetailViewer represents a detailed view of a specific item
type DetailViewer struct {
	Title   string
	Content []DetailSection
	Width   int
	Height  int
}

// DetailSection represents a section in the detail view
type DetailSection struct {
	Title   string
	Content []string
	Style   string // "info", "warning", "error", "success"
}

// NewDetailViewer creates a new detail viewer
func NewDetailViewer(title string, width, height int) *DetailViewer {
	return &DetailViewer{
		Title:  title,
		Width:  width,
		Height: height,
	}
}

// AddSection adds a section to the detail view
func (d *DetailViewer) AddSection(section DetailSection) {
	d.Content = append(d.Content, section)
}

// Clear clears all content
func (d *DetailViewer) Clear() {
	d.Content = d.Content[:0]
}

// Render renders the detail viewer
func (d *DetailViewer) Render() string {
	styles := ui.GetStyles()

	content := make([]string, 0, len(d.Content)+5)

	// Add title
	content = append(content, styles.Header.Render(d.Title), "")

	// Render sections
	for _, section := range d.Content {
		content = append(content, d.renderSection(section)...)
		content = append(content, "")
	}

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)
	return styles.Panel.Width(d.Width).Render(joined)
}

// renderSection renders a detail section
func (d *DetailViewer) renderSection(section DetailSection) []string {
	styles := ui.GetStyles()

	lines := make([]string, 0, len(section.Content)+5)

	// Section title
	var titleStyle lipgloss.Style
	switch section.Style {
	case "success":
		titleStyle = styles.Success
	case "warning":
		titleStyle = styles.Warning
	case "error":
		titleStyle = styles.Error
	case "info":
		titleStyle = styles.Info
	default:
		titleStyle = styles.Subheader
	}

	lines = append(lines, titleStyle.Bold(true).Render(section.Title))

	// Section content
	for _, line := range section.Content {
		lines = append(lines, styles.Body.Render("  "+line))
	}

	return lines
}

// Helper functions
