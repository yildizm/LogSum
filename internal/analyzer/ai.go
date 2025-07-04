package analyzer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yildizm/LogSum/internal/ai"
	"github.com/yildizm/LogSum/internal/common"
	corrpkg "github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/go-promptfmt"
	"golang.org/x/sync/semaphore"
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
	semaphore    *semaphore.Weighted // For limiting concurrent AI requests
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

	// Set default values for zero-valued fields
	if options.MaxConcurrentRequests == 0 {
		options.MaxConcurrentRequests = 3
	}
	if options.MaxTokensPerRequest == 0 {
		options.MaxTokensPerRequest = 2000
	}
	if options.MaxContextTokens == 0 {
		options.MaxContextTokens = 1000
	}

	return &AIAnalyzer{
		baseAnalyzer: baseAnalyzer,
		options:      options,
		correlator:   nil, // Will be set via SetCorrelator
		semaphore:    semaphore.NewWeighted(int64(options.MaxConcurrentRequests)),
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

	// Perform AI operations concurrently with request limiting
	return a.executeConcurrentAnalysis(ctx, aiAnalysis, baseAnalysis, entries, documentContext)
}

// executeConcurrentAnalysis runs AI analysis tasks concurrently with proper synchronization
func (a *AIAnalyzer) executeConcurrentAnalysis(ctx context.Context, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Error analysis
	if a.options.EnableErrorAnalysis && baseAnalysis.ErrorCount > 0 {
		wg.Add(1)
		go a.performErrorAnalysis(ctx, &wg, &mu, aiAnalysis, baseAnalysis, entries, documentContext)
	}

	// Root cause analysis
	if a.options.EnableRootCauseAnalysis {
		wg.Add(1)
		go a.performRootCauseAnalysis(ctx, &wg, &mu, aiAnalysis, baseAnalysis, entries, documentContext)
	}

	// Recommendations
	if a.options.EnableRecommendations {
		wg.Add(1)
		go a.performRecommendationAnalysis(ctx, &wg, &mu, aiAnalysis, baseAnalysis, entries, documentContext)
	}

	// Wait for all concurrent operations to complete
	wg.Wait()
	return nil
}

// performErrorAnalysis handles error analysis in a goroutine
func (a *AIAnalyzer) performErrorAnalysis(ctx context.Context, wg *sync.WaitGroup, mu *sync.Mutex, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) {
	a.runConcurrentAnalysis(ctx, wg, "Error analysis", func() error {
		errorAnalysis, err := a.analyzeErrors(ctx, baseAnalysis, entries, documentContext)
		if err != nil {
			return err
		}

		if documentContext != nil {
			errorAnalysis.SourceCitations = a.extractCitations(documentContext)
		}

		mu.Lock()
		aiAnalysis.ErrorAnalysis = errorAnalysis
		mu.Unlock()
		return nil
	})
}

// runConcurrentAnalysis is a helper to reduce duplication in analysis methods
func (a *AIAnalyzer) runConcurrentAnalysis(ctx context.Context, wg *sync.WaitGroup, taskName string, workFn func() error) {
	defer wg.Done()
	if err := a.semaphore.Acquire(ctx, 1); err != nil {
		return
	}
	defer a.semaphore.Release(1)

	if err := workFn(); err != nil {
		fmt.Printf("%s failed: %v\n", taskName, err)
	}
}

// performRootCauseAnalysis handles root cause analysis in a goroutine
func (a *AIAnalyzer) performRootCauseAnalysis(ctx context.Context, wg *sync.WaitGroup, mu *sync.Mutex, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) {
	runSliceAnalysis(a, ctx, wg, mu, "Root cause analysis",
		a.identifyRootCauses,
		func(result []RootCause) { aiAnalysis.RootCauses = result },
		func(items []RootCause, citations []SourceCitation) {
			for i := range items {
				items[i].SourceCitations = citations
			}
		},
		baseAnalysis, entries, documentContext)
}

// performRecommendationAnalysis handles recommendation generation in a goroutine
func (a *AIAnalyzer) performRecommendationAnalysis(ctx context.Context, wg *sync.WaitGroup, mu *sync.Mutex, aiAnalysis *AIAnalysis, baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext) {
	runSliceAnalysis(a, ctx, wg, mu, "Recommendation generation",
		a.generateRecommendations,
		func(result []Recommendation) { aiAnalysis.Recommendations = result },
		func(items []Recommendation, citations []SourceCitation) {
			for i := range items {
				items[i].SourceCitations = citations
			}
		},
		baseAnalysis, entries, documentContext)
}

// runSliceAnalysis is a generic helper for slice-based analysis operations
func runSliceAnalysis[T any](
	a *AIAnalyzer, ctx context.Context, wg *sync.WaitGroup, mu *sync.Mutex, taskName string,
	analyzeFunc func(context.Context, *common.Analysis, []*common.LogEntry, *DocumentContext) ([]T, error),
	setResult func([]T),
	setCitations func([]T, []SourceCitation),
	baseAnalysis *common.Analysis, entries []*common.LogEntry, documentContext *DocumentContext,
) {
	a.runConcurrentAnalysis(ctx, wg, taskName, func() error {
		result, err := analyzeFunc(ctx, baseAnalysis, entries, documentContext)
		if err != nil {
			return err
		}

		if documentContext != nil {
			citations := a.extractCitations(documentContext)
			setCitations(result, citations)
		}

		mu.Lock()
		setResult(result)
		mu.Unlock()
		return nil
	})
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
	// Build a human-readable summary prompt (not JSON)
	pb := promptfmt.New().
		System("You are a LogSum AI assistant specializing in log analysis. Provide a clear, human-readable summary of the log analysis. Focus on key insights, error patterns, and actionable recommendations.")

	// Build analysis context
	contextInfo := fmt.Sprintf("Log Analysis Summary - Total Entries: %d, Errors: %d, Warnings: %d, Time Range: %s to %s",
		analysis.TotalEntries,
		analysis.ErrorCount,
		analysis.WarnCount,
		analysis.StartTime.Format(time.RFC3339),
		analysis.EndTime.Format(time.RFC3339))

	// Add error samples if available
	if analysis.ErrorCount > 0 {
		errorEntries := a.extractErrorEntries(entries)
		sampleSize := min(3, len(errorEntries))
		if sampleSize > 0 {
			contextInfo += "\n\nKey Error Samples:\n"
			for i := 0; i < sampleSize; i++ {
				entry := errorEntries[i]
				contextInfo += fmt.Sprintf("- [%s] %s: %s\n",
					entry.Timestamp.Format(time.RFC3339),
					entry.Level,
					entry.Message)
			}
		}
	}

	// Add document context if available
	if docContext != nil && len(docContext.CorrelatedDocuments) > 0 {
		contextSection := a.buildContextSection(docContext)
		if contextSection != "" {
			contextInfo += "\n\n" + contextSection
		}
	}

	// Add patterns if available
	if len(analysis.Patterns) > 0 {
		contextInfo += "\n\nDetected Patterns:\n"
		for i, pattern := range analysis.Patterns {
			if i >= 3 {
				break
			}
			contextInfo += fmt.Sprintf("- %s (%d occurrences)\n", pattern.Pattern.Name, pattern.Count)
		}
	}

	// Add insights if available
	if len(analysis.Insights) > 0 {
		contextInfo += "\n\nKey Insights:\n"
		for i, insight := range analysis.Insights {
			if i >= 3 {
				break
			}
			contextInfo += fmt.Sprintf("- %s (%s, %.1f%% confidence)\n",
				insight.Title, insight.Type, insight.Confidence*100)
		}
	}

	return pb.
		User("Please provide a concise, human-readable summary of this log analysis. Focus on:\n1. Overall system health\n2. Key errors and their potential impact\n3. Notable patterns or trends\n4. Immediate recommendations\n\nAnalysis Data:\n%s", contextInfo).
		Build()
}

func (a *AIAnalyzer) buildErrorAnalysisPrompt(errorEntries []*common.LogEntry, analysis *common.Analysis, docContext *DocumentContext) *promptfmt.Prompt {
	// Use error analysis pattern from go-promptfmt
	errorPattern := promptfmt.ErrorAnalysis()

	// Build error samples
	sampleSize := min(10, len(errorEntries))
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
		sampleSize := min(5, len(errorEntries))
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

// buildDocumentContext creates DocumentContext from correlation results including direct error correlations
func (a *AIAnalyzer) buildDocumentContext(correlationResult interface{}) *DocumentContext {
	// Type-assert to CorrelationResult
	result, ok := correlationResult.(*corrpkg.CorrelationResult)
	if !ok || result == nil {
		return nil
	}

	var contextDocs []ContextDocument
	tokenCount := 0
	maxTokens := a.options.MaxContextTokens
	if maxTokens == 0 {
		maxTokens = 4000 // Default max tokens
	}

	// Convert pattern-based correlation results to document context
	for _, correlation := range result.Correlations {
		for _, docMatch := range correlation.DocumentMatches {
			if tokenCount >= maxTokens {
				break
			}

			// Create excerpt from document content
			excerpt := a.limitText(docMatch.Document.Content, 200)

			contextDoc := ContextDocument{
				Title:           docMatch.Document.Title,
				Path:            docMatch.Document.Path,
				MatchedKeywords: docMatch.MatchedKeywords,
				Score:           docMatch.Score,
				Excerpt:         excerpt,
				RelevantSection: docMatch.Highlighted,
				Source:          "pattern-correlation", // Identify source of correlation
			}

			contextDocs = append(contextDocs, contextDoc)
			// Rough token estimate (4 chars per token)
			tokenCount += len(excerpt) / 4
		}
	}

	// NEW: Convert direct error correlation results to document context
	for _, errorCorrelation := range result.DirectCorrelations {
		if tokenCount >= maxTokens {
			break
		}

		for _, docMatch := range errorCorrelation.DocumentMatches {
			if tokenCount >= maxTokens {
				break
			}

			// Create enhanced excerpt that includes error context
			excerpt := a.limitText(docMatch.Document.Content, 200)

			// Create a more detailed context document for error correlations
			contextDoc := ContextDocument{
				Title:           docMatch.Document.Title,
				Path:            docMatch.Document.Path,
				MatchedKeywords: append(docMatch.MatchedKeywords, errorCorrelation.Keywords...),
				Score:           docMatch.Score,
				Excerpt:         excerpt,
				RelevantSection: docMatch.Highlighted,
				Source:          "direct-error-correlation",            // Identify as direct error correlation
				ErrorContext:    a.buildErrorContext(errorCorrelation), // NEW: Add error-specific context
			}

			contextDocs = append(contextDocs, contextDoc)
			// Rough token estimate (4 chars per token)
			tokenCount += len(excerpt) / 4
		}
	}

	// Sort documents by score (highest first) to prioritize most relevant
	for i := 0; i < len(contextDocs)-1; i++ {
		for j := i + 1; j < len(contextDocs); j++ {
			if contextDocs[i].Score < contextDocs[j].Score {
				contextDocs[i], contextDocs[j] = contextDocs[j], contextDocs[i]
			}
		}
	}

	// Check if we have any actual correlations
	if len(contextDocs) == 0 {
		return nil
	}

	return &DocumentContext{
		CorrelatedDocuments: contextDocs,
		TotalDocuments:      len(contextDocs),
		TokensUsed:          tokenCount,
		TruncatedContext:    tokenCount >= maxTokens,
		DirectErrorCount:    len(result.DirectCorrelations), // NEW: Track direct error correlations
	}
}

// buildErrorContext creates contextual information for direct error correlations
func (a *AIAnalyzer) buildErrorContext(errorCorrelation *corrpkg.ErrorCorrelation) string {
	if errorCorrelation == nil {
		return ""
	}

	var contextBuilder strings.Builder

	// Error type and occurrence info
	contextBuilder.WriteString(fmt.Sprintf("Error Type: %s", errorCorrelation.ErrorType))
	if errorCorrelation.MatchCount > 1 {
		contextBuilder.WriteString(fmt.Sprintf(" (%d occurrences)", errorCorrelation.MatchCount))
	}

	// Confidence level
	contextBuilder.WriteString(fmt.Sprintf(", Confidence: %.2f", errorCorrelation.Confidence))

	// Keywords used for correlation
	if len(errorCorrelation.Keywords) > 0 {
		contextBuilder.WriteString(fmt.Sprintf(", Keywords: %s", strings.Join(errorCorrelation.Keywords, ", ")))
	}

	// Sample error message (truncated)
	if errorCorrelation.Error != nil && errorCorrelation.Error.Message != "" {
		sampleMessage := a.limitText(errorCorrelation.Error.Message, 100)
		contextBuilder.WriteString(fmt.Sprintf(", Sample: %s", sampleMessage))
	}

	return contextBuilder.String()
}

// buildContextSection creates the context section for prompts
func (a *AIAnalyzer) buildContextSection(docContext *DocumentContext) string {
	if docContext == nil || len(docContext.CorrelatedDocuments) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Context: Relevant Documentation\n")
	builder.WriteString("===============================\n")

	// Add summary of correlation types
	if docContext.DirectErrorCount > 0 {
		builder.WriteString(fmt.Sprintf("Direct error correlations found: %d\n", docContext.DirectErrorCount))
	}

	for i := range docContext.CorrelatedDocuments {
		if i >= 5 { // Limit to top 5 for comprehensive context
			break
		}
		doc := &docContext.CorrelatedDocuments[i]

		builder.WriteString(fmt.Sprintf("\n[%d] %s (Score: %.2f)", i+1, doc.Title, doc.Score))

		// Indicate correlation source
		if doc.Source != "" {
			switch doc.Source {
			case "direct-error-correlation":
				builder.WriteString(" [Direct Error Correlation]")
			case "pattern-correlation":
				builder.WriteString(" [Pattern Correlation]")
			}
		}
		builder.WriteString("\n")

		if doc.RelevantSection != "" {
			builder.WriteString(fmt.Sprintf("Section: %s\n", doc.RelevantSection))
		}
		if len(doc.MatchedKeywords) > 0 {
			builder.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(doc.MatchedKeywords, ", ")))
		}

		// Add error context for direct error correlations
		if doc.ErrorContext != "" {
			builder.WriteString(fmt.Sprintf("Error Context: %s\n", doc.ErrorContext))
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

	for i := range docContext.CorrelatedDocuments {
		doc := &docContext.CorrelatedDocuments[i]
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
