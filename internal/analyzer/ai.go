package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/common"
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
		Prompt:       prompt,
		SystemPrompt: "You are a log analysis expert. Provide concise, actionable summaries of log analysis results.",
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
		Prompt:       prompt,
		SystemPrompt: "You are an expert in error analysis. Analyze log errors and provide structured insights in JSON format.",
		MaxTokens:    a.options.MaxTokensPerRequest,
		Temperature:  0.2,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response
	var errorAnalysis ErrorAnalysis
	if err := json.Unmarshal([]byte(resp.Content), &errorAnalysis); err != nil {
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
		Prompt:       prompt,
		SystemPrompt: "You are a system reliability expert. Identify root causes of issues from log data and provide structured analysis in JSON format.",
		MaxTokens:    a.options.MaxTokensPerRequest,
		Temperature:  0.3,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	var rootCauses []RootCause
	if err := json.Unmarshal([]byte(resp.Content), &rootCauses); err != nil {
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
		Prompt:       prompt,
		SystemPrompt: "You are a DevOps consultant. Provide actionable recommendations based on log analysis in JSON format.",
		MaxTokens:    a.options.MaxTokensPerRequest,
		Temperature:  0.4,
	}

	resp, err := a.options.Provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	var recommendations []Recommendation
	if err := json.Unmarshal([]byte(resp.Content), &recommendations); err != nil {
		return nil, nil
	}

	return recommendations, nil
}

// Helper methods for building prompts

func (a *AIAnalyzer) buildSummaryPrompt(analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) string {
	var builder strings.Builder

	builder.WriteString("Analyze this log summary and provide a concise overview:\n\n")
	builder.WriteString(fmt.Sprintf("Time Range: %s to %s\n",
		analysis.StartTime.Format(time.RFC3339),
		analysis.EndTime.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Total Entries: %d\n", analysis.TotalEntries))
	builder.WriteString(fmt.Sprintf("Errors: %d, Warnings: %d\n", analysis.ErrorCount, analysis.WarnCount))

	if len(analysis.Patterns) > 0 {
		builder.WriteString("\nTop Patterns:\n")
		for i, pattern := range analysis.Patterns {
			if i >= 5 { // Limit to top 5 patterns
				break
			}
			builder.WriteString(fmt.Sprintf("- %s (%d occurrences)\n",
				pattern.Pattern.Name, pattern.Count))
		}
	}

	if len(analysis.Insights) > 0 {
		builder.WriteString("\nKey Insights:\n")
		for _, insight := range analysis.Insights {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", insight.Title, insight.Description))
		}
	}

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			builder.WriteString("\n\n")
			builder.WriteString(contextSection)
		}
	}

	builder.WriteString("\nProvide a 2-3 sentence summary focusing on the most critical findings and overall system health.")

	return builder.String()
}

func (a *AIAnalyzer) buildErrorAnalysisPrompt(errorEntries []*common.LogEntry, analysis *common.Analysis, docContext *DocumentContext) string {
	var builder strings.Builder

	builder.WriteString("Analyze these error logs and provide structured insights in JSON format:\n\n")

	// Include sample error entries (limit to avoid token overflow)
	sampleSize := minInt(10, len(errorEntries))
	builder.WriteString("Sample Error Entries:\n")
	for i := 0; i < sampleSize; i++ {
		entry := errorEntries[i]
		builder.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level,
			entry.Message))
	}

	builder.WriteString(fmt.Sprintf("\nTotal Errors: %d\n", analysis.ErrorCount))
	builder.WriteString(fmt.Sprintf("Time Range: %s to %s\n",
		analysis.StartTime.Format(time.RFC3339),
		analysis.EndTime.Format(time.RFC3339)))

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			builder.WriteString("\n\n")
			builder.WriteString(contextSection)
		}
	}

	builder.WriteString("\nProvide JSON response with: summary, critical_errors[], error_patterns[], severity_breakdown{}")

	return builder.String()
}

func (a *AIAnalyzer) buildRootCausePrompt(analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) string {
	var builder strings.Builder

	builder.WriteString("Identify potential root causes from this log analysis. Respond in JSON format:\n\n")

	// Include context about errors and patterns
	builder.WriteString(fmt.Sprintf("Error Count: %d\n", analysis.ErrorCount))
	builder.WriteString(fmt.Sprintf("Warning Count: %d\n", analysis.WarnCount))

	if len(analysis.Patterns) > 0 {
		builder.WriteString("\nFrequent Patterns:\n")
		for i, pattern := range analysis.Patterns {
			if i >= 3 {
				break
			}
			builder.WriteString(fmt.Sprintf("- %s (%d times)\n", pattern.Pattern.Name, pattern.Count))
		}
	}

	// Include recent error samples
	errorEntries := a.extractErrorEntries(entries)
	if len(errorEntries) > 0 {
		sampleSize := minInt(5, len(errorEntries))
		builder.WriteString("\nRecent Errors:\n")
		for i := 0; i < sampleSize; i++ {
			entry := errorEntries[i]
			builder.WriteString(fmt.Sprintf("- %s\n", entry.Message))
		}
	}

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			builder.WriteString("\n\n")
			builder.WriteString(contextSection)
		}
	}

	builder.WriteString("\nProvide JSON array of root causes with: title, description, confidence (0-1), category, impact")

	return builder.String()
}

func (a *AIAnalyzer) buildRecommendationPrompt(analysis *common.Analysis, entries []*common.LogEntry, docContext *DocumentContext) string {
	var builder strings.Builder

	builder.WriteString("Based on this log analysis, provide actionable recommendations in JSON format:\n\n")

	builder.WriteString(fmt.Sprintf("System Health: %d errors, %d warnings out of %d total entries\n",
		analysis.ErrorCount, analysis.WarnCount, analysis.TotalEntries))

	if len(analysis.Insights) > 0 {
		builder.WriteString("\nKey Issues Identified:\n")
		for _, insight := range analysis.Insights {
			builder.WriteString(fmt.Sprintf("- %s (%s)\n", insight.Title, insight.Type))
		}
	}

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			builder.WriteString("\n\n")
			builder.WriteString(contextSection)
		}
	}

	builder.WriteString("\nProvide JSON array of recommendations with: title, description, priority, category, action_items[], effort")

	return builder.String()
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
