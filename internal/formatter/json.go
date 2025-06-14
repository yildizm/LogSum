package formatter

import (
	"encoding/json"
	"time"

	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
)

// jsonFormatter formats output as JSON
type jsonFormatter struct{}

// NewJSON creates a new JSON formatter
func NewJSON() Formatter {
	return &jsonFormatter{}
}

func (f *jsonFormatter) Format(analysis *analyzer.Analysis) ([]byte, error) {
	// Create enhanced JSON structure as specified in TASK-007
	output := &EnhancedJSONOutput{
		Summary:  createSummary(analysis),
		Patterns: createPatternOutputs(analysis.Patterns),
		Insights: createInsightOutputs(analysis.Insights),
		Timeline: createTimelineOutput(analysis.Timeline),
	}

	return json.MarshalIndent(output, "", "  ")
}

// Enhanced JSON output structures as specified in TASK-007

// EnhancedJSONOutput represents the enhanced JSON structure
type EnhancedJSONOutput struct {
	Summary  *SummaryOutput   `json:"summary"`
	Patterns []*PatternOutput `json:"patterns"`
	Insights []*InsightOutput `json:"insights"`
	Timeline *TimelineOutput  `json:"timeline,omitempty"`
}

// SummaryOutput represents the summary section
type SummaryOutput struct {
	TotalEntries int        `json:"total_entries"`
	ErrorCount   int        `json:"error_count"`
	WarningCount int        `json:"warning_count"`
	TimeRange    *TimeRange `json:"time_range,omitempty"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration string    `json:"duration"`
}

// PatternOutput represents enhanced pattern match output
type PatternOutput struct {
	Pattern       *common.Pattern    `json:"pattern"`
	Matches       int                `json:"matches"`
	FirstSeen     time.Time          `json:"first_seen,omitempty"`
	LastSeen      time.Time          `json:"last_seen,omitempty"`
	SampleEntries []*common.LogEntry `json:"sample_entries,omitempty"`
}

// InsightOutput represents enhanced insight output
type InsightOutput struct {
	Type          string  `json:"type"`
	Severity      int     `json:"severity"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Confidence    float64 `json:"confidence"`
	EvidenceCount int     `json:"evidence_count"`
}

// TimelineOutput represents timeline data
type TimelineOutput struct {
	BucketSize string                `json:"bucket_size"`
	Buckets    []analyzer.TimeBucket `json:"buckets"`
}

// Helper functions for enhanced JSON output

// createSummary creates enhanced summary output
func createSummary(analysis *analyzer.Analysis) *SummaryOutput {
	summary := &SummaryOutput{
		TotalEntries: analysis.TotalEntries,
		ErrorCount:   analysis.ErrorCount,
		WarningCount: analysis.WarnCount,
	}

	// Add time range if available
	if !analysis.StartTime.IsZero() && !analysis.EndTime.IsZero() {
		summary.TimeRange = &TimeRange{
			Start:    analysis.StartTime,
			End:      analysis.EndTime,
			Duration: analysis.EndTime.Sub(analysis.StartTime).String(),
		}
	}

	return summary
}

// createPatternOutputs creates enhanced pattern outputs with sample entries
func createPatternOutputs(patterns []analyzer.PatternMatch) []*PatternOutput {
	outputs := make([]*PatternOutput, 0, len(patterns))

	for _, match := range patterns {
		output := &PatternOutput{
			Pattern:   match.Pattern,
			Matches:   match.Count,
			FirstSeen: match.FirstSeen,
			LastSeen:  match.LastSeen,
		}

		// Add first 3 sample entries as specified in TASK-007
		if len(match.Matches) > 0 {
			sampleCount := 3
			if len(match.Matches) < sampleCount {
				sampleCount = len(match.Matches)
			}
			output.SampleEntries = match.Matches[:sampleCount]
		}

		outputs = append(outputs, output)
	}

	return outputs
}

// createInsightOutputs creates enhanced insight outputs
func createInsightOutputs(insights []analyzer.Insight) []*InsightOutput {
	outputs := make([]*InsightOutput, 0, len(insights))

	for _, insight := range insights {
		output := &InsightOutput{
			Type:          string(insight.Type),
			Severity:      int(insight.Severity),
			Title:         insight.Title,
			Description:   insight.Description,
			Confidence:    insight.Confidence,
			EvidenceCount: len(insight.Evidence),
		}

		outputs = append(outputs, output)
	}

	return outputs
}

// createTimelineOutput creates enhanced timeline output
func createTimelineOutput(timeline *analyzer.Timeline) *TimelineOutput {
	if timeline == nil {
		return nil
	}

	return &TimelineOutput{
		BucketSize: timeline.BucketSize.String(),
		Buckets:    timeline.Buckets,
	}
}
