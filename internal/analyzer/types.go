package analyzer

import (
	"github.com/yildizm/LogSum/internal/common"
)

// Re-export common types for backward compatibility
type Analysis = common.Analysis
type PatternMatch = common.PatternMatch
type Insight = common.Insight
type InsightType = common.InsightType
type Timeline = common.Timeline
type TimeBucket = common.TimeBucket

// Re-export constants
const (
	InsightTypeErrorSpike  = common.InsightTypeErrorSpike
	InsightTypePerformance = common.InsightTypePerformance
	InsightTypeAnomaly     = common.InsightTypeAnomaly
	InsightTypeRootCause   = common.InsightTypeRootCause
)
