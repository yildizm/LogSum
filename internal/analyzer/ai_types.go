package analyzer

import (
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/common"
)

// AIAnalysis represents AI-enhanced log analysis results
type AIAnalysis struct {
	// Base analysis data
	Analysis *Analysis `json:"analysis"`

	// AI-generated insights
	AISummary       string           `json:"ai_summary"`
	ErrorAnalysis   *ErrorAnalysis   `json:"error_analysis,omitempty"`
	RootCauses      []RootCause      `json:"root_causes,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`

	// Document context
	DocumentContext *DocumentContext `json:"document_context,omitempty"`

	// Analysis metadata
	AnalyzedAt     time.Time      `json:"analyzed_at"`
	Provider       string         `json:"provider"`
	Model          string         `json:"model"`
	TokenUsage     *ai.TokenUsage `json:"token_usage,omitempty"`
	ProcessingTime time.Duration  `json:"processing_time"`
}

// ErrorAnalysis contains AI analysis of errors
type ErrorAnalysis struct {
	Summary           string            `json:"summary"`
	CriticalErrors    []ErrorInsight    `json:"critical_errors"`
	ErrorPatterns     []ErrorPattern    `json:"error_patterns"`
	CorrelatedEvents  []CorrelatedEvent `json:"correlated_events,omitempty"`
	SeverityBreakdown map[string]int    `json:"severity_breakdown"`
	SourceCitations   []SourceCitation  `json:"source_citations,omitempty"`
}

// ErrorInsight represents AI analysis of a specific error
type ErrorInsight struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Severity    common.LogLevel    `json:"severity"`
	FirstSeen   time.Time          `json:"first_seen"`
	LastSeen    time.Time          `json:"last_seen"`
	Occurrences int                `json:"occurrences"`
	Evidence    []*common.LogEntry `json:"evidence"`
	Explanation string             `json:"explanation"`
	Impact      string             `json:"impact"`
	Confidence  float64            `json:"confidence"`
}

// ErrorPattern represents a pattern identified in errors
type ErrorPattern struct {
	Pattern     string             `json:"pattern"`
	Description string             `json:"description"`
	Frequency   int                `json:"frequency"`
	Examples    []*common.LogEntry `json:"examples"`
	Trend       TrendDirection     `json:"trend"`
}

// RootCause represents an AI-identified root cause
type RootCause struct {
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	Confidence      float64            `json:"confidence"`
	Evidence        []*common.LogEntry `json:"evidence"`
	Category        RootCauseCategory  `json:"category"`
	Impact          ImpactLevel        `json:"impact"`
	Timeline        []time.Time        `json:"timeline,omitempty"`
	SourceCitations []SourceCitation   `json:"source_citations,omitempty"`
}

// Recommendation represents an AI-generated recommendation
type Recommendation struct {
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Priority        RecommendationPriority `json:"priority"`
	Category        RecommendationCategory `json:"category"`
	ActionItems     []string               `json:"action_items"`
	Benefits        []string               `json:"benefits"`
	Effort          EffortLevel            `json:"effort"`
	RelatedIssues   []string               `json:"related_issues,omitempty"`
	SourceCitations []SourceCitation       `json:"source_citations,omitempty"`
}

// CorrelatedEvent represents events that correlate with errors
type CorrelatedEvent struct {
	EventType   string             `json:"event_type"`
	Description string             `json:"description"`
	Timestamp   time.Time          `json:"timestamp"`
	Correlation float64            `json:"correlation"`
	Evidence    []*common.LogEntry `json:"evidence"`
}

// TrendDirection indicates the trend of a pattern
type TrendDirection string

const (
	TrendIncreasing  TrendDirection = "increasing"
	TrendDecreasing  TrendDirection = "decreasing"
	TrendStable      TrendDirection = "stable"
	TrendFluctuating TrendDirection = "fluctuating"
)

// RootCauseCategory categorizes root causes
type RootCauseCategory string

const (
	RootCauseInfrastructure RootCauseCategory = "infrastructure"
	RootCauseApplication    RootCauseCategory = "application"
	RootCauseNetwork        RootCauseCategory = "network"
	RootCauseDatabase       RootCauseCategory = "database"
	RootCauseConfiguration  RootCauseCategory = "configuration"
	RootCausePerformance    RootCauseCategory = "performance"
	RootCauseSecurity       RootCauseCategory = "security"
	RootCauseExternal       RootCauseCategory = "external"
)

// ImpactLevel represents the impact level of an issue
type ImpactLevel string

const (
	ImpactCritical ImpactLevel = "critical"
	ImpactHigh     ImpactLevel = "high"
	ImpactMedium   ImpactLevel = "medium"
	ImpactLow      ImpactLevel = "low"
)

// RecommendationPriority represents the priority of a recommendation
type RecommendationPriority string

const (
	PriorityUrgent RecommendationPriority = "urgent"
	PriorityHigh   RecommendationPriority = "high"
	PriorityMedium RecommendationPriority = "medium"
	PriorityLow    RecommendationPriority = "low"
)

// RecommendationCategory categorizes recommendations
type RecommendationCategory string

const (
	CategoryMonitoring    RecommendationCategory = "monitoring"
	CategoryPerformance   RecommendationCategory = "performance"
	CategorySecurity      RecommendationCategory = "security"
	CategoryMaintenance   RecommendationCategory = "maintenance"
	CategoryConfiguration RecommendationCategory = "configuration"
	CategoryScaling       RecommendationCategory = "scaling"
	CategoryDevelopment   RecommendationCategory = "development"
)

// EffortLevel represents the effort required for a recommendation
type EffortLevel string

const (
	EffortMinimal     EffortLevel = "minimal"
	EffortLow         EffortLevel = "low"
	EffortMedium      EffortLevel = "medium"
	EffortHigh        EffortLevel = "high"
	EffortSignificant EffortLevel = "significant"
)

// DocumentContext contains relevant documentation context for AI analysis
type DocumentContext struct {
	CorrelatedDocuments []ContextDocument `json:"correlated_documents"`
	TotalDocuments      int               `json:"total_documents"`
	TokensUsed          int               `json:"tokens_used"`
	TruncatedContext    bool              `json:"truncated_context"`
	DirectErrorCount    int               `json:"direct_error_count,omitempty"` // Number of direct error correlations
}

// ContextDocument represents a document used for AI context
type ContextDocument struct {
	Title           string   `json:"title"`
	Path            string   `json:"path"`
	MatchedKeywords []string `json:"matched_keywords"`
	Score           float64  `json:"score"`
	Excerpt         string   `json:"excerpt"`
	RelevantSection string   `json:"relevant_section,omitempty"`
	Source          string   `json:"source,omitempty"`        // Source of correlation (pattern-correlation, direct-error-correlation)
	ErrorContext    string   `json:"error_context,omitempty"` // Additional context for error correlations
}

// SourceCitation represents a citation to source documentation
type SourceCitation struct {
	DocumentTitle string  `json:"document_title"`
	DocumentPath  string  `json:"document_path"`
	Section       string  `json:"section,omitempty"`
	Relevance     float64 `json:"relevance"`
	Quote         string  `json:"quote,omitempty"`
}

// AIAnalyzerOptions configures the AI analyzer
type AIAnalyzerOptions struct {
	// Provider to use for AI analysis
	Provider ai.Provider

	// MaxTokensPerRequest limits token usage per AI request
	MaxTokensPerRequest int

	// EnableErrorAnalysis enables detailed error analysis
	EnableErrorAnalysis bool

	// EnableRootCauseAnalysis enables root cause analysis
	EnableRootCauseAnalysis bool

	// EnableRecommendations enables recommendation generation
	EnableRecommendations bool

	// IncludeContext includes additional context in AI requests
	IncludeContext bool

	// EnableDocumentContext enables document correlation and context
	EnableDocumentContext bool

	// MaxContextTokens limits the tokens used for document context
	MaxContextTokens int

	// MinConfidence sets minimum confidence threshold for results
	MinConfidence float64

	// MaxConcurrentRequests limits concurrent AI requests
	MaxConcurrentRequests int
}
