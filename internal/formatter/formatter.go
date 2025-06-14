package formatter

import "github.com/yildizm/LogSum/internal/analyzer"

// Formatter defines the interface for output formatting
type Formatter interface {
	Format(analysis *analyzer.Analysis) ([]byte, error)
}
