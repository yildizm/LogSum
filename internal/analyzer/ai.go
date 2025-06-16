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

// AIAnalyzer wraps the base analyzer with AI capabilities
type AIAnalyzer struct {
	baseAnalyzer Analyzer
	options      *AIAnalyzerOptions
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
			MinConfidence:           0.6,
			MaxConcurrentRequests:   3,
		}
	}

	return &AIAnalyzer{
		baseAnalyzer: baseAnalyzer,
		options:      options,
	}
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

	// Generate AI summary
	summary, err := a.generateSummary(ctx, baseAnalysis, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI summary: %w", err)
	}
	aiAnalysis.AISummary = summary

	// Perform error analysis if enabled and errors exist
	if a.options.EnableErrorAnalysis && baseAnalysis.ErrorCount > 0 {
		errorAnalysis, err := a.analyzeErrors(ctx, baseAnalysis, entries)
		if err != nil {
			// Log error but don't fail the entire analysis
			fmt.Printf("Error analysis failed: %v\n", err)
		} else {
			aiAnalysis.ErrorAnalysis = errorAnalysis
		}
	}

	// Perform root cause analysis if enabled
	if a.options.EnableRootCauseAnalysis {
		rootCauses, err := a.identifyRootCauses(ctx, baseAnalysis, entries)
		if err != nil {
			fmt.Printf("Root cause analysis failed: %v\n", err)
		} else {
			aiAnalysis.RootCauses = rootCauses
		}
	}

	// Generate recommendations if enabled
	if a.options.EnableRecommendations {
		recommendations, err := a.generateRecommendations(ctx, baseAnalysis, entries)
		if err != nil {
			fmt.Printf("Recommendation generation failed: %v\n", err)
		} else {
			aiAnalysis.Recommendations = recommendations
		}
	}

	aiAnalysis.ProcessingTime = time.Since(startTime)
	return aiAnalysis, nil
}

// generateSummary creates an AI-generated summary of the analysis
func (a *AIAnalyzer) generateSummary(ctx context.Context, analysis *Analysis, entries []*common.LogEntry) (string, error) {
	prompt := a.buildSummaryPrompt(analysis, entries)

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
func (a *AIAnalyzer) analyzeErrors(ctx context.Context, analysis *Analysis, entries []*common.LogEntry) (*ErrorAnalysis, error) {
	errorEntries := a.extractErrorEntries(entries)
	if len(errorEntries) == 0 {
		return nil, nil
	}

	prompt := a.buildErrorAnalysisPrompt(errorEntries, analysis)

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
func (a *AIAnalyzer) identifyRootCauses(ctx context.Context, analysis *Analysis, entries []*common.LogEntry) ([]RootCause, error) {
	if analysis.ErrorCount == 0 {
		return nil, nil
	}

	prompt := a.buildRootCausePrompt(analysis, entries)

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
	for _, rc := range rootCauses {
		if rc.Confidence >= a.options.MinConfidence {
			filtered = append(filtered, rc)
		}
	}

	return filtered, nil
}

// generateRecommendations creates actionable recommendations
func (a *AIAnalyzer) generateRecommendations(ctx context.Context, analysis *Analysis, entries []*common.LogEntry) ([]Recommendation, error) {
	prompt := a.buildRecommendationPrompt(analysis, entries)

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

func (a *AIAnalyzer) buildSummaryPrompt(analysis *Analysis, entries []*common.LogEntry) string {
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

	builder.WriteString("\nProvide a 2-3 sentence summary focusing on the most critical findings and overall system health.")

	return builder.String()
}

func (a *AIAnalyzer) buildErrorAnalysisPrompt(errorEntries []*common.LogEntry, analysis *Analysis) string {
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

	builder.WriteString("\nProvide JSON response with: summary, critical_errors[], error_patterns[], severity_breakdown{}")

	return builder.String()
}

func (a *AIAnalyzer) buildRootCausePrompt(analysis *Analysis, entries []*common.LogEntry) string {
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

	builder.WriteString("\nProvide JSON array of root causes with: title, description, confidence (0-1), category, impact")

	return builder.String()
}

func (a *AIAnalyzer) buildRecommendationPrompt(analysis *Analysis, entries []*common.LogEntry) string {
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

	builder.WriteString("\nProvide JSON array of recommendations with: title, description, priority, category, action_items[], effort")

	return builder.String()
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
