package ui

import (
	"github.com/yildizm/LogSum/internal/analyzer"
)

// View represents different UI views
type View int

const (
	ViewAnalyzing View = iota
	ViewResults
	ViewDetails
	ViewHelp
)

// Model holds the UI state
type Model struct {
	CurrentView View
	Analysis    *analyzer.Analysis
	Selected    int
	Width       int
	Height      int
	Error       error
}
