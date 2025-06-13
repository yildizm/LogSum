package analyzer

import (
	"context"
	"time"

	"github.com/yildizm/LogSum/internal/common"
)

// Analyzer performs log analysis
type Analyzer interface {
	// Analyze performs analysis on log entries
	Analyze(ctx context.Context, entries []*common.LogEntry) (*Analysis, error)

	// AddPattern adds a pattern for detection
	AddPattern(pattern *common.Pattern) error

	// SetPatterns sets all patterns for detection
	SetPatterns(patterns []*common.Pattern) error
}

// Engine provides different analysis strategies
type Engine interface {
	Analyzer

	// WithTimeline enables timeline analysis
	WithTimeline(bucketSize time.Duration) Engine

	// WithPatterns loads patterns from a source
	WithPatterns(source string) (Engine, error)

	// WithInsights enables insight generation
	WithInsights() Engine
}
