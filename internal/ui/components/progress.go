package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar represents a progress bar component
type ProgressBar struct {
	Width     int
	Current   int
	Total     int
	StartTime time.Time
	ShowETA   bool
	Label     string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int) *ProgressBar {
	return &ProgressBar{
		Width:     width,
		StartTime: time.Now(),
		ShowETA:   true,
	}
}

// SetProgress updates the progress
func (p *ProgressBar) SetProgress(current, total int) {
	p.Current = current
	p.Total = total
}

// SetLabel sets the progress label
func (p *ProgressBar) SetLabel(label string) {
	p.Label = label
}

// Render renders the progress bar
func (p *ProgressBar) Render() string {
	// Define styles locally to avoid import cycle
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

	if p.Total == 0 {
		return p.renderIndeterminate()
	}

	percentage := float64(p.Current) / float64(p.Total)
	if percentage > 1.0 {
		percentage = 1.0
	}

	// Calculate filled width
	filledWidth := int(float64(p.Width) * percentage)
	emptyWidth := p.Width - filledWidth

	// Create progress bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", emptyWidth)

	bar := progressStyle.Render(filled) + mutedStyle.Render(empty)

	// Create percentage text
	percentText := fmt.Sprintf("%.1f%%", percentage*100)

	// Create ETA text
	etaText := ""
	if p.ShowETA && p.Current > 0 && percentage > 0 {
		elapsed := time.Since(p.StartTime)
		estimated := time.Duration(float64(elapsed) / percentage)
		remaining := estimated - elapsed
		if remaining > 0 {
			etaText = fmt.Sprintf(" ETA: %s", p.formatDuration(remaining))
		}
	}

	// Create status text
	statusText := fmt.Sprintf("%d/%d %s%s", p.Current, p.Total, percentText, etaText)

	result := fmt.Sprintf("[%s] %s", bar, statusText)

	if p.Label != "" {
		result = p.Label + "\n" + result
	}

	return result
}

// renderIndeterminate renders an indeterminate progress bar
func (p *ProgressBar) renderIndeterminate() string {
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

	// Create a moving animation
	elapsed := time.Since(p.StartTime)
	frame := int(elapsed.Milliseconds()/100) % p.Width

	var bar strings.Builder
	for i := 0; i < p.Width; i++ {
		if i >= frame && i < frame+3 {
			bar.WriteString(progressStyle.Render("█"))
		} else {
			bar.WriteString(mutedStyle.Render("░"))
		}
	}

	result := fmt.Sprintf("[%s] Processing...", bar.String())

	if p.Label != "" {
		result = p.Label + "\n" + result
	}

	return result
}

// formatDuration formats a duration for display
func (p *ProgressBar) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// Spinner represents a spinning progress indicator
type Spinner struct {
	Frame     int
	StartTime time.Time
	Label     string
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		StartTime: time.Now(),
	}
}

// SetLabel sets the spinner label
func (s *Spinner) SetLabel(label string) {
	s.Label = label
}

// Tick advances the spinner animation
func (s *Spinner) Tick() {
	spinnerFrames := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	s.Frame = (s.Frame + 1) % len(spinnerFrames)
}

// Render renders the spinner
func (s *Spinner) Render() string {
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)

	spinnerFrames := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	char := string(spinnerFrames[s.Frame])
	spinner := progressStyle.Render(char)

	if s.Label != "" {
		return fmt.Sprintf("%s %s", spinner, s.Label)
	}

	return spinner
}

// LoadingIndicator combines multiple loading states
type LoadingIndicator struct {
	spinner     *Spinner
	progressBar *ProgressBar
	useSpinner  bool
	message     string
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator(width int) *LoadingIndicator {
	return &LoadingIndicator{
		spinner:     NewSpinner(),
		progressBar: NewProgressBar(width),
		useSpinner:  true,
	}
}

// SetMessage sets the loading message
func (l *LoadingIndicator) SetMessage(message string) {
	l.message = message
	l.spinner.SetLabel(message)
	l.progressBar.SetLabel(message)
}

// SetProgress switches to progress bar mode
func (l *LoadingIndicator) SetProgress(current, total int) {
	l.useSpinner = false
	l.progressBar.SetProgress(current, total)
}

// UseSpinner switches to spinner mode
func (l *LoadingIndicator) UseSpinner() {
	l.useSpinner = true
}

// Tick advances the animation
func (l *LoadingIndicator) Tick() {
	if l.useSpinner {
		l.spinner.Tick()
	}
}

// Render renders the loading indicator
func (l *LoadingIndicator) Render() string {
	if l.useSpinner {
		return l.spinner.Render()
	}
	return l.progressBar.Render()
}
