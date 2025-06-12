package analyzer

import (
	"time"

	"github.com/yildizm/LogSum/internal/parser"
)

// Analysis represents the result of log analysis
type Analysis struct {
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
	TotalEntries int            `json:"total_entries"`
	ErrorCount   int            `json:"error_count"`
	WarnCount    int            `json:"warn_count"`
	Patterns     []PatternMatch `json:"patterns"`
	Insights     []Insight      `json:"insights"`
	Timeline     *Timeline      `json:"timeline,omitempty"`
}

// PatternMatch represents a matched pattern in logs
type PatternMatch struct {
	Pattern   *parser.Pattern    `json:"pattern"`
	Matches   []*parser.LogEntry `json:"matches"`
	Count     int                `json:"count"`
	FirstSeen time.Time          `json:"first_seen"`
	LastSeen  time.Time          `json:"last_seen"`
}

// Insight represents an analysis insight
type Insight struct {
	Type        InsightType        `json:"type"`
	Severity    parser.LogLevel    `json:"severity"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Evidence    []*parser.LogEntry `json:"evidence"`
	Confidence  float64            `json:"confidence"`
}

// InsightType categorizes insights
type InsightType string

const (
	InsightTypeErrorSpike  InsightType = "error_spike"
	InsightTypePerformance InsightType = "performance"
	InsightTypeAnomaly     InsightType = "anomaly"
	InsightTypeRootCause   InsightType = "root_cause"
)

// Timeline represents temporal analysis
type Timeline struct {
	Buckets    []TimeBucket  `json:"buckets"`
	BucketSize time.Duration `json:"bucket_size"`
}

// TimeBucket represents a time window of log data
type TimeBucket struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	EntryCount int       `json:"entry_count"`
	ErrorCount int       `json:"error_count"`
	WarnCount  int       `json:"warn_count"`
}
