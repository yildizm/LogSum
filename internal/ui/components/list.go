package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
)

// ListItem represents an item in a list
type ListItem struct {
	ID          string
	Title       string
	Description string
	Status      string
	Icon        string
	Data        interface{} // Store associated data
}

// List represents a navigable list component
type List struct {
	Title         string
	Items         []ListItem
	Selected      int
	Focused       bool
	Width         int
	Height        int
	ShowNumbers   bool
	ShowIcons     bool
	Searchable    bool
	searchQuery   string
	filteredItems []int // Indices of filtered items
}

// NewList creates a new list component
func NewList(title string, width, height int) *List {
	return &List{
		Title:       title,
		Width:       width,
		Height:      height,
		ShowNumbers: true,
		ShowIcons:   true,
		Searchable:  true,
	}
}

// AddItem adds an item to the list
func (l *List) AddItem(item *ListItem) {
	l.Items = append(l.Items, *item)
	l.updateFilter()
}

// SetItems sets all items in the list
func (l *List) SetItems(items []ListItem) {
	l.Items = items
	l.Selected = 0
	l.updateFilter()
}

// SetFocused sets the focus state of the list
func (l *List) SetFocused(focused bool) {
	l.Focused = focused
}

// GetSelectedItem returns the currently selected item
func (l *List) GetSelectedItem() *ListItem {
	if len(l.filteredItems) == 0 || l.Selected >= len(l.filteredItems) {
		return nil
	}
	index := l.filteredItems[l.Selected]
	if index >= len(l.Items) {
		return nil
	}
	return &l.Items[index]
}

// MoveUp moves selection up
func (l *List) MoveUp() {
	if l.Selected > 0 {
		l.Selected--
	}
}

// MoveDown moves selection down
func (l *List) MoveDown() {
	if l.Selected < len(l.filteredItems)-1 {
		l.Selected++
	}
}

// SetSearch sets the search query and filters items
func (l *List) SetSearch(query string) {
	l.searchQuery = query
	l.Selected = 0
	l.updateFilter()
}

// updateFilter updates the filtered items based on search query
func (l *List) updateFilter() {
	l.filteredItems = l.filteredItems[:0]

	for i, item := range l.Items {
		if l.searchQuery == "" || l.matchesSearch(&item, l.searchQuery) {
			l.filteredItems = append(l.filteredItems, i)
		}
	}
}

// matchesSearch checks if an item matches the search query
func (l *List) matchesSearch(item *ListItem, query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(strings.ToLower(item.Title), query) ||
		strings.Contains(strings.ToLower(item.Description), query) ||
		strings.Contains(strings.ToLower(item.ID), query)
}

// Render renders the list
func (l *List) Render() string {
	primaryColor := lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}
	secondaryColor := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	selectedColor := lipgloss.AdaptiveColor{Light: "#DBEAFE", Dark: "#1E3A8A"}

	headerStyle := lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	focusedStyle := lipgloss.NewStyle().Background(selectedColor).Foreground(primaryColor)
	normalStyle := lipgloss.NewStyle().Foreground(secondaryColor)

	var content []string

	// Add title
	title := headerStyle.Render(l.Title)
	if l.Focused {
		title = focusedStyle.Render(title)
	}
	content = append(content, title)

	// Add search indicator
	if l.searchQuery != "" {
		searchText := fmt.Sprintf("Search: %s (%d results)", l.searchQuery, len(l.filteredItems))
		content = append(content, normalStyle.Render(searchText))
	}

	content = append(content, "")

	// Calculate visible range
	maxVisible := l.Height - 4 // Account for title and spacing
	if maxVisible < 1 {
		maxVisible = 1
	}

	startIndex := 0
	if l.Selected >= maxVisible {
		startIndex = l.Selected - maxVisible + 1
	}

	endIndex := startIndex + maxVisible
	if endIndex > len(l.filteredItems) {
		endIndex = len(l.filteredItems)
	}

	// Render visible items
	for i := startIndex; i < endIndex; i++ {
		itemIndex := l.filteredItems[i]
		item := l.Items[itemIndex]

		isSelected := i == l.Selected
		content = append(content, l.renderItem(&item, i+1, isSelected))
	}

	// Add scrolling indicator
	if len(l.filteredItems) > maxVisible {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)", startIndex+1, endIndex, len(l.filteredItems))
		content = append(content, "", normalStyle.Render(scrollInfo))
	}

	joined := lipgloss.JoinVertical(lipgloss.Left, content...)

	panelStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(secondaryColor)

	if l.Focused {
		return focusedStyle.Width(l.Width).Render(joined)
	}

	return panelStyle.Width(l.Width).Render(joined)
}

// renderItem renders a single list item
func (l *List) renderItem(item *ListItem, number int, selected bool) string {
	primaryColor := lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}
	secondaryColor := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	selectedColor := lipgloss.AdaptiveColor{Light: "#DBEAFE", Dark: "#1E3A8A"}
	successColor := lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"}
	warningColor := lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"}
	errorColor := lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"}

	var parts []string

	// Add number
	if l.ShowNumbers {
		numStr := fmt.Sprintf("%2d.", number)
		parts = append(parts, numStr)
	}

	// Add icon
	if l.ShowIcons && item.Icon != "" {
		parts = append(parts, item.Icon)
	}

	// Add title
	title := item.Title
	if item.Description != "" {
		title += " - " + item.Description
	}
	parts = append(parts, title)

	line := strings.Join(parts, " ")

	// Apply styling based on status
	var style lipgloss.Style
	if selected {
		style = lipgloss.NewStyle().Background(selectedColor).Foreground(primaryColor)
	} else {
		style = lipgloss.NewStyle().Foreground(secondaryColor)
		// Apply status color
		switch item.Status {
		case "success":
			style = style.Foreground(successColor)
		case "warning":
			style = style.Foreground(warningColor)
		case "error":
			style = style.Foreground(errorColor)
		case "info":
			style = style.Foreground(primaryColor)
		}
	}

	return style.Width(l.Width - 4).Render(line)
}

// PatternList creates a list component for patterns
func NewPatternList(patterns []analyzer.PatternMatch, width, height int) *List {
	list := NewList("Patterns Found", width, height)

	for _, pattern := range patterns {
		status := "info"
		icon := "🔍"

		// Determine status based on pattern type
		switch pattern.Pattern.Type {
		case common.PatternTypeError:
			status = "error"
			icon = "❌"
		case common.PatternTypeAnomaly:
			status = "warning"
			icon = "⚠️"
		case common.PatternTypePerformance:
			status = "warning"
			icon = "⚠️"
		case common.PatternTypeSecurity:
			status = "error"
			icon = "❌"
		}

		description := fmt.Sprintf("%d matches", pattern.Count)
		if !pattern.FirstSeen.IsZero() {
			description += fmt.Sprintf(" (first: %s)", pattern.FirstSeen.Format("15:04:05"))
		}

		item := ListItem{
			ID:          pattern.Pattern.ID,
			Title:       pattern.Pattern.Name,
			Description: description,
			Status:      status,
			Icon:        icon,
			Data:        pattern,
		}

		list.AddItem(&item)
	}

	return list
}

// InsightList creates a list component for insights
func NewInsightList(insights []analyzer.Insight, width, height int) *List {
	list := NewList("Insights", width, height)

	for i, insight := range insights {
		status := "info"
		icon := "💡"

		// Determine status based on severity
		switch insight.Severity {
		case common.LevelError:
			status = "error"
			icon = "❌"
		case common.LevelWarn:
			status = "warning"
			icon = "⚠️"
		case common.LevelInfo:
			status = "info"
			icon = "ℹ️"
		}

		description := fmt.Sprintf("%.0f%% confidence", insight.Confidence*100)
		if len(insight.Evidence) > 0 {
			description += fmt.Sprintf(" (%d entries)", len(insight.Evidence))
		}

		item := ListItem{
			ID:          fmt.Sprintf("insight-%d", i),
			Title:       insight.Title,
			Description: description,
			Status:      status,
			Icon:        icon,
			Data:        insight,
		}

		list.AddItem(&item)
	}

	return list
}

// LogList creates a list component for log entries
func NewLogList(entries []*common.LogEntry, width, height int) *List {
	list := NewList("Log Entries", width, height)
	list.ShowNumbers = false // Line numbers are more relevant for logs

	for i, entry := range entries {
		status := "info"
		icon := "ℹ️"

		// Determine status based on log level
		switch entry.LogLevel {
		case common.LevelError:
			status = "error"
			icon = "❌"
		case common.LevelWarn:
			status = "warning"
			icon = "⚠️"
		case common.LevelInfo:
			status = "info"
			icon = "ℹ️"
		case common.LevelDebug:
			status = "debug"
			icon = "🐛"
		}

		// Format timestamp and line
		timestamp := ""
		if !entry.Timestamp.IsZero() {
			timestamp = entry.Timestamp.Format("15:04:05")
		}

		title := entry.Message
		if len(title) > 60 {
			title = title[:57] + "..."
		}

		description := ""
		if timestamp != "" {
			description = timestamp
		}
		if entry.Service != "" {
			if description != "" {
				description += " "
			}
			description += "[" + entry.Service + "]"
		}
		if entry.LineNumber > 0 {
			if description != "" {
				description += " "
			}
			description += fmt.Sprintf("L%d", entry.LineNumber)
		}

		item := ListItem{
			ID:          fmt.Sprintf("log-%d", i),
			Title:       title,
			Description: description,
			Status:      status,
			Icon:        icon,
			Data:        entry,
		}

		list.AddItem(&item)
	}

	return list
}
