package analyzer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/go-promptfmt"
)

// DocumentCorrelator is an interface for correlating analysis results with documentation
type DocumentCorrelator interface {
	Correlate(ctx context.Context, analysis *common.Analysis) (interface{}, error)
	SetDocumentStore(store interface{}) error
}

// AIAnalyzer wraps the base analyzer with AI capabilities
type AIAnalyzer struct {
	baseAnalyzer Analyzer
	options      *AIAnalyzerOptions
	correlator   DocumentCorrelator
}

// NewAIAnalyzer creates a new AI-enhanced analyzer
func NewAIAnalyzer(baseAnalyzer Analyzer, options *AIAnalyzerOptions) *AIAnalyzer {
	if options == nil {
		options = &AIAnalyzerOptions{
			MaxTokensPerRequest:     2000,
			EnableErrorAnalysis:     true,
			EnableRootCauseAnalysis: true,
			EnableRecommendations:   true,
			IncludeContext:          true,
			EnableDocumentContext:   false, // Optional by default
			MaxContextTokens:        1000,
			MinConfidence:           0.6,
			MaxConcurrentRequests:   3,
		}
	}

	return &AIAnalyzer{
		baseAnalyzer: baseAnalyzer,
		options:      options,
		correlator:   nil, // Will be set via SetCorrelator
	}
}

// SetCorrelator sets the document correlator
func (a *AIAnalyzer) SetCorrelator(correlator DocumentCorrelator) {
	a.correlator = correlator
}

// SetDocumentStore configures the document store for context retrieval
func (a *AIAnalyzer) SetDocumentStore(store interface{}) error {
	if a.correlator == nil {
		return fmt.Errorf("correlator not initialized")
	}
	return a.correlator.SetDocumentStore(store)
}

// AnalyzeWithAI performs enhanced analysis using AI
func (a *AIAnalyzer) AnalyzeWithAI(ctx context.Context, entries []*common.LogEntry) (*AIAnalysis, error) {
	startTime := time.Now()

	// First perform base analysis
	baseAnalysis, err := a.baseAnalyzer.Analyze(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("base analysis failed: %w", err)
	}

	// Create AI analysis result
	aiAnalysis := &AIAnalysis{
		Analysis:       baseAnalysis,
		AnalyzedAt:     time.Now(),
		Provider:       a.options.Provider.Name(),
		ProcessingTime: time.Since(startTime),
	}

	// Get document context for AI analysis
	documentContext := a.getDocumentContext(ctx, baseAnalysis)
	aiAnalysis.DocumentContext = documentContext

	// Perform AI analysis tasks
	if err := a.performAIAnalysis(ctx, aiAnalysis, baseAnalysis, entries, documentContext); err != nil {
		return nil, err
	}

	aiAnalysis.ProcessingTime = time.Since(startTime)
	return aiAnalysis, nil
}

// getDocumentContext retrieves document context for AI analysis
func (a *AIAnalyzer) getDocumentContext(ctx context.Context, baseAnalysis *common.Analysis) *DocumentContext {
	if !a.options.EnableDocumentContext || a.correlator == nil {
		return nil
	}

	correlationResult, err := a.correlator.Correlate(ctx, baseAnalysis)
	if err != nil || correlationResult == nil {
		return nil
	}

	return a.buildDocumentContext(correlationResult)
}

// performAIAnalysis executes all AI analysis tasks
func (a *AIAnalyzer) performAIAnalysis(ctx context.Context, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) error {
	// Generate AI summary
	summary, err := a.generateSummary(ctx, baseAnalysis, entries, documentContext)
	if err != nil {
		return fmt.Errorf("failed to generate AI summary: %w", err)
	}
	aiAnalysis.AISummary = summary

	// Perform error analysis
	a.performErrorAnalysis(ctx, aiAnalysis, baseAnalysis, entries, documentContext)

	// Perform root cause analysis
	a.performRootCauseAnalysis(ctx, aiAnalysis, baseAnalysis, entries, documentContext)

	// Generate recommendations
	a.performRecommendationAnalysis(ctx, aiAnalysis, baseAnalysis, entries, documentContext)

	return nil
}

// performErrorAnalysis handles error analysis with source citations
func (a *AIAnalyzer) performErrorAnalysis(ctx context.Context, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) {
	if !a.options.EnableErrorAnalysis || baseAnalysis.ErrorCount == 0 {
		return
	}

	errorAnalysis, err := a.analyzeErrors(ctx, baseAnalysis, entries, documentContext)
	if err != nil {
		fmt.Printf("Error analysis failed: %v\n", err)
		return
	}

	if documentContext != nil {
		errorAnalysis.SourceCitations = a.extractCitations(documentContext)
	}
	aiAnalysis.ErrorAnalysis = errorAnalysis
}

// performRootCauseAnalysis handles root cause analysis with source citations
func (a *AIAnalyzer) performRootCauseAnalysis(ctx context.Context, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) {
	if !a.options.EnableRootCauseAnalysis {
		return
	}

	rootCauses, err := a.identifyRootCauses(ctx, baseAnalysis, entries, documentContext)
	if err != nil {
		fmt.Printf("Root cause analysis failed: %v\n", err)
		return
	}

	if documentContext != nil {
		citations := a.extractCitations(documentContext)
		for i := range rootCauses {
			rootCauses[i].SourceCitations = citations
		}
	}
	aiAnalysis.RootCauses = rootCauses
}

// performRecommendationAnalysis handles recommendation generation with source citations
func (a *AIAnalyzer) performRecommendationAnalysis(ctx context.Context, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) {
	if !a.options.EnableRecommendations {
		return
	}

	recommendations, err := a.generateRecommendations(ctx, baseAnalysis, entries, documentContext)
	if err != nil {
		fmt.Printf("Recommendation generation failed: %v\n", err)
		return
	}

	if documentContext != nil {
		citations := a.extractCitations(documentContext)
		for i := range recommendations {
			recommendations[i].SourceCitations = citations
		}
	}
	aiAnalysis.Recommendations = recommendations
}

// generateSummary creates an AI-generated summary of the analysis
func (a *AIAnalyzer) generateSummary(ctx context.Context, analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) (string, error) {
	prompt := a.buildSummaryPrompt(analysis, entries, docContext)

	req := &ai.CompletionRequest{
		Prompt:       prompt.String(),
		SystemPrompt: prompt.SystemPrompt,
		MaxTokens:    500,
		Temperature:  0.3,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	// Store token usage if available for tracking/billing
	_ = resp.Usage

	return strings.TrimSpace(resp.Content), nil
}

// analyzeErrors performs detailed AI analysis of errors
func (a *AIAnalyzer) analyzeErrors(ctx context.Context, analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) (*ErrorAnalysis, error) {
	errorEntries := a.extractErrorEntries(entries)
	if len(errorEntries) == 0 {
		return nil, nil
	}

	prompt := a.buildErrorAnalysisPrompt(errorEntries, analysis, docContext)

	req := &ai.CompletionRequest{
		Prompt:       prompt.String(),
		SystemPrompt: prompt.SystemPrompt,
		MaxTokens:    a.options.MaxTokensPerRequest,
		Temperature:  0.2,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse JSON response using go-promptfmt
	response := promptfmt.NewResponse(resp.Content)
	var errorAnalysis ErrorAnalysis
	parseResult := response.TryParseJSON(&errorAnalysis)
	if !parseResult.Success {
		// If JSON parsing fails, create a basic error analysis
		return &ErrorAnalysis{
			Summary: resp.Content,
			SeverityBreakdown: map[string]int{
				"error": analysis.ErrorCount,
				"warn":  analysis.WarnCount,
			},
		}, nil
	}

	return &errorAnalysis, nil
}

// identifyRootCauses uses AI to identify potential root causes
func (a *AIAnalyzer) identifyRootCauses(ctx context.Context, analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) ([]RootCause, error) {
	if analysis.ErrorCount == 0 {
		return nil, nil
	}

	prompt := a.buildRootCausePrompt(analysis, entries, docContext)

	req := &ai.CompletionRequest{
		Prompt:       prompt.String(),
		SystemPrompt: prompt.SystemPrompt,
		MaxTokens:    a.options.MaxTokensPerRequest,
		Temperature:  0.3,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse JSON response using go-promptfmt
	response := promptfmt.NewResponse(resp.Content)
	var rootCauses []RootCause
	parseResult := response.TryParseJSON(&rootCauses)
	if !parseResult.Success {
		// If parsing fails, return empty slice
		return nil, nil
	}

	// Filter by confidence threshold
	filtered := make([]RootCause, 0)
	for i := range rootCauses {
		if rootCauses[i].Confidence >= a.options.MinConfidence {
			filtered = append(filtered, rootCauses[i])
		}
	}

	return filtered, nil
}

// generateRecommendations creates actionable recommendations
func (a *AIAnalyzer) generateRecommendations(ctx context.Context, analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) ([]Recommendation, error) {
	prompt := a.buildRecommendationPrompt(analysis, entries, docContext)

	req := &ai.CompletionRequest{
		Prompt:       prompt.String(),
		SystemPrompt: prompt.SystemPrompt,
		MaxTokens:    a.options.MaxTokensPerRequest,
		Temperature:  0.4,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse JSON response using go-promptfmt
	response := promptfmt.NewResponse(resp.Content)
	var recommendations []Recommendation
	parseResult := response.TryParseJSON(&recommendations)
	if !parseResult.Success {
		return nil, nil
	}

	return recommendations, nil
}

// Helper methods for building prompts

func (a *AIAnalyzer) buildSummaryPrompt(analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) *promptfmt.Prompt {
	// Use LogSum-specific pattern for better structure
	logPattern := LogAnalysis().WithAnalysis(analysis)

	// Add error entries if available
	if analysis.ErrorCount > 0 {
		errorEntries := a.extractErrorEntries(entries)
		logPattern.WithErrorEntries(errorEntries).WithSampleSize(5)
	}

	prompt := logPattern.Build()

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			// Create a new prompt with the context added
			pb := promptfmt.New().
				System("You are a LogSum AI assistant specializing in log analysis. Provide structured insights about system health, errors, and operational patterns.").
				User("%s", prompt.String()).
				AddContext("documentation", contextSection)
			if prompt.JSONSchema != nil {
				pb.ExpectJSON(prompt.JSONSchema)
			}
			return pb.Build()
		}
	}

	return prompt
}

func (a *AIAnalyzer) buildErrorAnalysisPrompt(errorEntries []*common.LogEntry, analysis *common.Analysis, docContext *DocumentContext) *promptfmt.Prompt {
	// Use error analysis pattern from go-promptfmt
	errorPattern := promptfmt.ErrorAnalysis()

	// Build error samples
	sampleSize := minInt(10, len(errorEntries))
	errorSamples := "Sample Error Entries:\n"
	for i := 0; i < sampleSize; i++ {
		entry := errorEntries[i]
		errorSamples += fmt.Sprintf("[%s] %s: %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level,
			entry.Message)
	}

	// Build context
	contextInfo := fmt.Sprintf("LogSum analysis - Total Errors: %d, Time Range: %s to %s",
		analysis.ErrorCount,
		analysis.StartTime.Format(time.RFC3339),
		analysis.EndTime.Format(time.RFC3339))

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			contextInfo += "\n\n" + contextSection
		}
	}

	return errorPattern.
		WithError(errorSamples).
		WithContext(contextInfo).
		Build()
}

func (a *AIAnalyzer) buildRootCausePrompt(analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) *promptfmt.Prompt {
	// Use chain of thought pattern for root cause analysis
	cotPattern := promptfmt.ChainOfThought()

	// Build problem description
	problem := fmt.Sprintf("System experiencing %d errors, %d warnings. Analyze root causes.",
		analysis.ErrorCount, analysis.WarnCount)

	// Add patterns context
	if len(analysis.Patterns) > 0 {
		problem += "\n\nFrequent Patterns:\n"
		for i, pattern := range analysis.Patterns {
			if i >= 3 {
				break
			}
			problem += fmt.Sprintf("- %s (%d times)\n", pattern.Pattern.Name, pattern.Count)
		}
	}

	// Add error samples
	errorEntries := a.extractErrorEntries(entries)
	if len(errorEntries) > 0 {
		sampleSize := minInt(5, len(errorEntries))
		problem += "\n\nRecent Errors:\n"
		for i := 0; i < sampleSize; i++ {
			entry := errorEntries[i]
			problem += fmt.Sprintf("- %s\n", entry.Message)
		}
	}

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			problem += "\n\n" + contextSection
		}
	}

	return cotPattern.
		WithProblem(problem, "System Reliability").
		WithMaxSteps(5).
		Build()
}

func (a *AIAnalyzer) buildRecommendationPrompt(analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) *promptfmt.Prompt {
	pb := promptfmt.New().
		System("You are a DevOps consultant. Provide actionable recommendations based on log analysis in JSON format.").
		User("Based on this log analysis, provide actionable recommendations:\n\nSystem Health: %d errors, %d warnings out of %d total entries",
			analysis.ErrorCount, analysis.WarnCount, analysis.TotalEntries)

	// Add insights context
	if len(analysis.Insights) > 0 {
		insightsText := "Key Issues Identified:\n"
		for _, insight := range analysis.Insights {
			insightsText += fmt.Sprintf("- %s (%s)\n", insight.Title, insight.Type)
		}
		pb.AddContext("issues", insightsText)
	}

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			pb.AddContext("documentation", contextSection)
		}
	}

	// Expect structured JSON response
	type RecommendationResponse struct {
		Recommendations []struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Priority    string   `json:"priority"`
			Category    string   `json:"category"`
			ActionItems []string `json:"action_items"`
			Effort      string   `json:"effort"`
		} `json:"recommendations"`
	}

	return pb.ExpectJSON(&RecommendationResponse{}).Build()
}

// buildDocumentContext creates DocumentContext from correlation results
func (a *AIAnalyzer) buildDocumentContext(correlationResult interface{}) *DocumentContext {
	// This is a simple implementation that assumes the correlation result
	// is in the expected format. In a real implementation, you would
	// need to properly type-assert and convert the correlation result
	// to DocumentContext format.

	// For now, return nil to maintain compatibility
	// This method should be implemented based on the actual correlation
	// result structure from the correlation package
	return nil
}

// buildContextSection creates the context section for prompts
func (a *AIAnalyzer) buildContextSection(docContext *DocumentContext) string {
	if docContext == nil || len(docContext.CorrelatedDocuments) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Context: Relevant Documentation\n")
	builder.WriteString("===============================\n")

	for i, doc := range docContext.CorrelatedDocuments {
		if i >= 3 { // Limit to top 3 for brevity
			break
		}

		builder.WriteString(fmt.Sprintf("\n[%d] %s (Score: %.2f)\n", i+1, doc.Title, doc.Score))
		if doc.RelevantSection != "" {
			builder.WriteString(fmt.Sprintf("Section: %s\n", doc.RelevantSection))
		}
		if len(doc.MatchedKeywords) > 0 {
			builder.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(doc.MatchedKeywords, ", ")))
		}
		builder.WriteString(fmt.Sprintf("Content: %s\n", doc.Excerpt))
		builder.WriteString(fmt.Sprintf("Source: %s\n", doc.Path))
	}

	if docContext.TruncatedContext {
		builder.WriteString("\n[Note: Additional relevant documentation was found but truncated for brevity]\n")
	}

	return builder.String()
}

// limitText truncates text to specified character limit
func (a *AIAnalyzer) limitText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

// extractCitations converts document context to source citations
func (a *AIAnalyzer) extractCitations(docContext *DocumentContext) []SourceCitation {
	if docContext == nil || len(docContext.CorrelatedDocuments) == 0 {
		return nil
	}

	citations := make([]SourceCitation, 0, len(docContext.CorrelatedDocuments))

	for _, doc := range docContext.CorrelatedDocuments {
		citation := SourceCitation{
			DocumentTitle: doc.Title,
			DocumentPath:  doc.Path,
			Section:       doc.RelevantSection,
			Relevance:     doc.Score,
			Quote:         a.limitText(doc.Excerpt, 150), // Shorter quote for citations
		}
		citations = append(citations, citation)
	}

	return citations
}

func (a *AIAnalyzer) extractErrorEntries(entries []*common.LogEntry) []*common.LogEntry {
	var errorEntries []*common.LogEntry
	for _, entry := range entries {
		if entry.LogLevel == common.LevelError || entry.LogLevel == common.LevelFatal {
			errorEntries = append(errorEntries, entry)
		}
	}
	return errorEntries
}
