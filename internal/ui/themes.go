package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents a color theme for the TUI
type Theme struct {
	Name string

	// Primary colors
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Accent    lipgloss.AdaptiveColor

	// Semantic colors
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Info    lipgloss.AdaptiveColor

	// UI colors
	Border     lipgloss.AdaptiveColor
	Background lipgloss.AdaptiveColor
	Foreground lipgloss.AdaptiveColor
	Muted      lipgloss.AdaptiveColor
	Highlight  lipgloss.AdaptiveColor

	// Special colors
	Insight  lipgloss.AdaptiveColor
	Progress lipgloss.AdaptiveColor
	Selected lipgloss.AdaptiveColor
}

// buildTheme creates a theme with the given colors
func buildTheme(name string, primary, secondary, accent, success, warning, errorColor, info, border, background, foreground, muted, highlight, insight, progress, selected [2]string) Theme {
	return Theme{
		Name:       name,
		Primary:    lipgloss.AdaptiveColor{Light: primary[0], Dark: primary[1]},
		Secondary:  lipgloss.AdaptiveColor{Light: secondary[0], Dark: secondary[1]},
		Accent:     lipgloss.AdaptiveColor{Light: accent[0], Dark: accent[1]},
		Success:    lipgloss.AdaptiveColor{Light: success[0], Dark: success[1]},
		Warning:    lipgloss.AdaptiveColor{Light: warning[0], Dark: warning[1]},
		Error:      lipgloss.AdaptiveColor{Light: errorColor[0], Dark: errorColor[1]},
		Info:       lipgloss.AdaptiveColor{Light: info[0], Dark: info[1]},
		Border:     lipgloss.AdaptiveColor{Light: border[0], Dark: border[1]},
		Background: lipgloss.AdaptiveColor{Light: background[0], Dark: background[1]},
		Foreground: lipgloss.AdaptiveColor{Light: foreground[0], Dark: foreground[1]},
		Muted:      lipgloss.AdaptiveColor{Light: muted[0], Dark: muted[1]},
		Highlight:  lipgloss.AdaptiveColor{Light: highlight[0], Dark: highlight[1]},
		Insight:    lipgloss.AdaptiveColor{Light: insight[0], Dark: insight[1]},
		Progress:   lipgloss.AdaptiveColor{Light: progress[0], Dark: progress[1]},
		Selected:   lipgloss.AdaptiveColor{Light: selected[0], Dark: selected[1]},
	}
}

// Available themes
var (
	DefaultTheme = buildTheme("default",
		[2]string{"#1E40AF", "#3B82F6"}, [2]string{"#6B7280", "#9CA3AF"}, [2]string{"#7C3AED", "#A855F7"},
		[2]string{"#059669", "#10B981"}, [2]string{"#D97706", "#F59E0B"}, [2]string{"#DC2626", "#EF4444"},
		[2]string{"#0891B2", "#06B6D4"}, [2]string{"#D1D5DB", "#374151"}, [2]string{"#FFFFFF", "#111827"},
		[2]string{"#111827", "#F9FAFB"}, [2]string{"#6B7280", "#9CA3AF"}, [2]string{"#FEF3C7", "#1F2937"},
		[2]string{"#7C3AED", "#A855F7"}, [2]string{"#059669", "#10B981"}, [2]string{"#DBEAFE", "#1E3A8A"})

	HighContrastTheme = buildTheme("high-contrast",
		[2]string{"#000000", "#FFFFFF"}, [2]string{"#666666", "#BBBBBB"}, [2]string{"#000080", "#8080FF"},
		[2]string{"#006600", "#00FF00"}, [2]string{"#CC6600", "#FFAA00"}, [2]string{"#CC0000", "#FF4444"},
		[2]string{"#0066CC", "#4499FF"}, [2]string{"#000000", "#FFFFFF"}, [2]string{"#FFFFFF", "#000000"},
		[2]string{"#000000", "#FFFFFF"}, [2]string{"#666666", "#BBBBBB"}, [2]string{"#FFFF00", "#444444"},
		[2]string{"#800080", "#FF80FF"}, [2]string{"#006600", "#00FF00"}, [2]string{"#CCCCCC", "#333333"})

	MinimalTheme = buildTheme("minimal",
		[2]string{"#2D3748", "#E2E8F0"}, [2]string{"#718096", "#A0AEC0"}, [2]string{"#4A5568", "#CBD5E0"},
		[2]string{"#2F855A", "#68D391"}, [2]string{"#C05621", "#F6AD55"}, [2]string{"#C53030", "#FC8181"},
		[2]string{"#2B6CB0", "#63B3ED"}, [2]string{"#E2E8F0", "#2D3748"}, [2]string{"#FFFFFF", "#1A202C"},
		[2]string{"#2D3748", "#F7FAFC"}, [2]string{"#A0AEC0", "#718096"}, [2]string{"#F7FAFC", "#2D3748"},
		[2]string{"#553C9A", "#B794F6"}, [2]string{"#2F855A", "#68D391"}, [2]string{"#EDF2F7", "#2D3748"})
)

// Current active theme
var currentTheme = DefaultTheme

// GetTheme returns the current active theme
func GetTheme() Theme {
	return currentTheme
}

// SetTheme sets the active theme
func SetTheme(theme *Theme) {
	currentTheme = *theme
}

// SetThemeByName sets the theme by name
func SetThemeByName(name string) bool {
	switch name {
	case "default":
		SetTheme(&DefaultTheme)
		return true
	case "high-contrast":
		SetTheme(&HighContrastTheme)
		return true
	case "minimal":
		SetTheme(&MinimalTheme)
		return true
	default:
		return false
	}
}

// IsColorDisabled checks if colors should be disabled
func IsColorDisabled() bool {
	return os.Getenv("NO_COLOR") != ""
}

// GetAvailableThemes returns list of available theme names
func GetAvailableThemes() []string {
	return []string{"default", "high-contrast", "minimal"}
}

// Styled text helpers that respect color settings
func (t *Theme) StyledText(text string, style *lipgloss.Style) string {
	if IsColorDisabled() {
		return text
	}
	return style.Render(text)
}

// Common styles based on current theme
func GetStyles() *Styles {
	theme := GetTheme()

	return &Styles{
		Theme: theme,

		// Base styles
		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1),

		Header: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),

		Subheader: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true),

		Body: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		Muted: lipgloss.NewStyle().
			Foreground(theme.Muted),

		// Status styles
		Success: lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),

		Info: lipgloss.NewStyle().
			Foreground(theme.Info),

		// Interactive styles
		Selected: lipgloss.NewStyle().
			Background(theme.Selected).
			Foreground(theme.Primary).
			Bold(true),

		Focused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Padding(0, 1),

		// Layout styles
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(1, 2),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border).
			Padding(0, 1),

		// Special styles
		Progress: lipgloss.NewStyle().
			Foreground(theme.Progress).
			Bold(true),

		Insight: lipgloss.NewStyle().
			Foreground(theme.Insight).
			Bold(true),

		Highlight: lipgloss.NewStyle().
			Background(theme.Highlight).
			Foreground(theme.Primary),

		// List styles
		ListItem: lipgloss.NewStyle().
			Padding(0, 2),

		ListSelected: lipgloss.NewStyle().
			Background(theme.Selected).
			Foreground(theme.Primary).
			Padding(0, 2).
			Bold(true),
	}
}

// Styles contains all the styled components
type Styles struct {
	Theme Theme

	// Base styles
	Title     lipgloss.Style
	Header    lipgloss.Style
	Subheader lipgloss.Style
	Body      lipgloss.Style
	Muted     lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Interactive styles
	Selected lipgloss.Style
	Focused  lipgloss.Style

	// Layout styles
	Box   lipgloss.Style
	Panel lipgloss.Style

	// Special styles
	Progress  lipgloss.Style
	Insight   lipgloss.Style
	Highlight lipgloss.Style

	// List styles
	ListItem     lipgloss.Style
	ListSelected lipgloss.Style
}
