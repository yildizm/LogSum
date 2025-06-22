package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/LogSum/internal/formatter"
	"github.com/yildizm/LogSum/internal/monitor"
	"github.com/yildizm/LogSum/internal/ui"
	"github.com/yildizm/go-logparser"
)

var (
	analyzeFormat      string
	analyzePatterns    string
	analyzeFollow      bool
	analyzeTimeout     time.Duration
	analyzeMaxLines    int
	analyzeNoTUI       bool
	analyzeOutputFile  string
	analyzeDocsPath    string
	analyzeCorrelate   bool
	analyzeAI          bool
	analyzeMonitor     bool
	analyzeMonitorFile string
)

func newAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [file]",
		Short: "Analyze log files or stdin",
		Long: `Analyze log files for patterns, anomalies, and insights.

If no file is specified, reads from stdin. Supports auto-detection of log formats
or manual format specification.

Examples:
  logsum analyze app.log
  logsum analyze --format json access.log
  logsum analyze --ai app.log
  logsum analyze --ai --docs ./docs/ app.log
  logsum analyze --monitor app.log
  logsum analyze --ai --monitor --monitor-file metrics.json app.log
  cat app.log | logsum analyze
  logsum analyze --patterns ./patterns/ app.log`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAnalyze,
	}

	cmd.Flags().StringVarP(&analyzeFormat, "format", "f", "auto", "log format (auto, json, logfmt, text)")
	cmd.Flags().StringVarP(&analyzePatterns, "patterns", "p", "", "pattern file or directory")
	cmd.Flags().BoolVar(&analyzeFollow, "follow", false, "follow file for new entries")
	cmd.Flags().DurationVar(&analyzeTimeout, "timeout", 30*time.Second, "analysis timeout")
	cmd.Flags().IntVar(&analyzeMaxLines, "max-lines", 100000, "maximum lines to analyze")
	cmd.Flags().BoolVar(&analyzeNoTUI, "no-tui", false, "disable terminal UI, output to stdout")
	cmd.Flags().StringVar(&analyzeOutputFile, "output-file", "", "save output to file instead of stdout")
	cmd.Flags().StringVar(&analyzeDocsPath, "docs", "", "path to documentation directory for correlation")
	cmd.Flags().BoolVar(&analyzeCorrelate, "correlate", false, "enable error-documentation correlation")
	cmd.Flags().BoolVar(&analyzeAI, "ai", false, "enable AI-powered analysis with LLM integration")
	cmd.Flags().BoolVar(&analyzeMonitor, "monitor", false, "enable real-time performance monitoring during analysis")
	cmd.Flags().StringVar(&analyzeMonitorFile, "monitor-file", "", "save monitoring metrics to file (optional)")

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Get configuration
	cfg := GetGlobalConfig()

	// Use config values if flags weren't explicitly set
	if !cmd.Flag("timeout").Changed {
		analyzeTimeout = cfg.Analysis.Timeout
	}
	if !cmd.Flag("max-lines").Changed {
		analyzeMaxLines = cfg.Analysis.MaxEntries
	}

	ctx, cancel := context.WithTimeout(context.Background(), analyzeTimeout)
	defer cancel()

	// Get input reader
	reader, _, cleanup, err := setupInputReader(args)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Read and parse logs
	entries, err := readAndParseInput(reader)
	if err != nil {
		return err
	}

	// Load patterns using pattern loader
	patternLoader := NewPatternLoader()
	patterns := patternLoader.LoadAnalysisPatterns()

	// Run analysis
	return runAnalysisAndOutput(ctx, entries, patterns)
}

func readLines(reader io.Reader, maxLines int) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	lineCount := 0
	for scanner.Scan() && lineCount < maxLines {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
			lineCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return lines, fmt.Errorf("scanner error: %w", err)
	}

	return lines, nil
}

// setupInputReader sets up the input reader based on command args
func setupInputReader(args []string) (reader io.Reader, source string, cleanup func(), err error) {
	if len(args) == 0 {
		// Read from stdin
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Reading from stdin...\n")
		}
		return os.Stdin, "", nil, nil
	}

	// Read from file
	filename := args[0]

	// Validate and sanitize file path for security
	if err := validateFilePath(filename); err != nil {
		return nil, "", nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Clean the path to handle Windows path separators and trailing slashes
	cleanPath := filepath.Clean(filename)

	// #nosec G304 - path is validated above
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	cleanup = func() {
		if err := file.Close(); err != nil && isVerbose() {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Analyzing file: %s\n", cleanPath)
	}

	return file, cleanPath, cleanup, nil
}

// readAndParseInput reads and parses log entries from the input reader
func readAndParseInput(reader io.Reader) ([]*common.LogEntry, error) {
	// Read all lines
	lines, err := readLines(reader, analyzeMaxLines)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("no log entries found")
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Read %d lines\n", len(lines))
	}

	// Parse logs
	var entries []*common.LogEntry
	if analyzeFormat == "auto" {
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Auto-detecting format...\n")
		}
		p := logparser.New()
		logEntries, err := p.ParseString(strings.Join(lines, "\n"))
		if err == nil {
			entries = make([]*common.LogEntry, len(logEntries))
			for i, entry := range logEntries {
				entries[i] = common.ConvertToCommonLogEntry(&entry, i+1)
			}
		}
	} else {
		entries, err = parseWithSpecificFormat(lines)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse logs: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid log entries found")
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Parsed %d log entries\n", len(entries))
	}

	return entries, nil
}

// parseWithSpecificFormat parses lines with a specific format
func parseWithSpecificFormat(lines []string) ([]*common.LogEntry, error) {
	var format logparser.Format
	switch analyzeFormat {
	case "json":
		format = logparser.FormatJSON
	case "logfmt":
		format = logparser.FormatLogfmt
	case "text":
		format = logparser.FormatText
	default:
		return nil, fmt.Errorf("unknown format %s. Available formats: json, logfmt, text", analyzeFormat)
	}

	p := logparser.NewWithFormat(format)
	logEntries, err := p.ParseString(strings.Join(lines, "\n"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse logs: %w", err)
	}

	entries := make([]*common.LogEntry, len(logEntries))
	for i, entry := range logEntries {
		entries[i] = common.ConvertToCommonLogEntry(&entry, i+1)
	}

	return entries, nil
}

// runAnalysisAndOutput performs analysis and outputs results.
// This is the main orchestrator that determines whether to use TUI or CLI mode
// and coordinates the analysis pipeline.
func runAnalysisAndOutput(ctx context.Context, entries []*common.LogEntry, patterns []*common.Pattern) error {
	if shouldUseTUIMode() {
		return runTUIAnalysis(entries, patterns)
	}
	return runCLIAnalysis(ctx, entries, patterns)
}

// shouldUseTUIMode determines if the terminal UI should be used based on flags and output settings.
func shouldUseTUIMode() bool {
	return !analyzeNoTUI && getOutputFormat() == "text" && !isVerbose()
}

// runTUIAnalysis launches the interactive terminal UI for log analysis.
func runTUIAnalysis(entries []*common.LogEntry, patterns []*common.Pattern) error {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Launching interactive terminal UI...\n")
	}
	return ui.InteractiveRun(entries, patterns)
}

// runCLIAnalysis performs command-line analysis with optional correlation and outputs results.
func runCLIAnalysis(ctx context.Context, entries []*common.LogEntry, patterns []*common.Pattern) error {
	// Setup monitoring if enabled
	var metricsCollector monitor.Collector
	if analyzeMonitor {
		var err error
		metricsCollector, err = setupMonitoring(ctx)
		if err != nil {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: failed to setup monitoring: %v\n", err)
			}
		} else {
			defer func() {
				if err := stopMonitoring(metricsCollector); err != nil && isVerbose() {
					fmt.Fprintf(os.Stderr, "Warning: failed to stop monitoring: %v\n", err)
				}
			}()
		}
	}

	// Perform main analysis
	analysis, err := performAnalysisWithMonitoring(ctx, entries, patterns, metricsCollector)
	if err != nil {
		return err
	}

	// Perform correlation if enabled OR if AI is enabled (AI always shows correlation summary)
	var correlationResult *correlation.CorrelationResult
	if analyzeCorrelate || analyzeAI {
		var err error
		correlationResult, err = performCorrelationWithMonitoring(ctx, analysis, metricsCollector)
		if err != nil {
			logCorrelationWarning(err)
		}
	}

	// Format and output results
	return formatAndOutputResults(analysis, correlationResult)
}

// performCorrelationIfEnabled runs document correlation if docs path is available and correlation is needed.
func performCorrelationIfEnabled(ctx context.Context, analysis *analyzer.Analysis) (*correlation.CorrelationResult, error) {
	if !analyzeCorrelate || analyzeDocsPath == "" {
		return nil, nil
	}
	return performCorrelation(ctx, analysis)
}

// logCorrelationWarning logs correlation errors in verbose mode to avoid cluttering output.
func logCorrelationWarning(err error) {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Warning: correlation failed: %v\n", err)
	}
}

// performAnalysis runs the analysis engine with patterns
func performAnalysis(ctx context.Context, entries []*common.LogEntry, patterns []*common.Pattern) (*analyzer.Analysis, error) {
	engine := analyzer.NewEngine()
	if len(patterns) > 0 {
		if err := engine.SetPatterns(patterns); err != nil {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: failed to set patterns: %v\n", err)
			}
		}
	}

	if isVerbose() {
		if analyzeAI {
			fmt.Fprintf(os.Stderr, "Performing AI-enhanced analysis...\n")
		} else {
			fmt.Fprintf(os.Stderr, "Performing analysis...\n")
		}
	}

	// Use AI analyzer if --ai flag is set
	if analyzeAI {
		return performAIAnalysis(ctx, engine, entries)
	}

	// Standard analysis
	analysis, err := engine.Analyze(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	return analysis, nil
}

// performCorrelation runs document correlation on analysis results
func performCorrelation(ctx context.Context, analysis *analyzer.Analysis) (*correlation.CorrelationResult, error) {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Setting up document correlation...\n")
	}

	// Create document store
	store, err := setupDocumentStore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup document store: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil && isVerbose() {
			fmt.Fprintf(os.Stderr, "Warning: failed to close document store: %v\n", err)
		}
	}()

	// Create correlator
	correlator := correlation.NewCorrelator()
	if err := correlator.SetDocumentStore(store); err != nil {
		return nil, fmt.Errorf("failed to configure correlator: %w", err)
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Running correlation analysis...\n")
	}

	// Perform correlation
	result, err := correlator.Correlate(ctx, analysis)
	if err != nil {
		return nil, fmt.Errorf("correlation failed: %w", err)
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Found %d correlations out of %d patterns\n",
			result.CorrelatedPatterns, result.TotalPatterns)
	}

	return result, nil
}

// setupDocumentStore creates and populates a document store
func setupDocumentStore(ctx context.Context) (docstore.DocumentStore, error) {
	// Create memory-based document store
	store := docstore.NewMemoryStore()
	scanner := docstore.NewMarkdownScanner()
	indexer := docstore.NewMemoryIndexer()

	docStore := docstore.NewStore(store, scanner, indexer)

	// Scan and index documents from the specified path
	patterns := []string{"*.md", "*.txt", "*.rst"}
	if err := docStore.ProcessDirectoryWithContext(ctx, analyzeDocsPath, patterns, nil); err != nil {
		return nil, fmt.Errorf("failed to index documents: %w", err)
	}

	// Debug: Check how many documents were loaded (commented out for production)
	if isVerbose() {
		stats, err := store.Stats()
		if err == nil {
			fmt.Fprintf(os.Stderr, "Loaded %d documents from %s (total sections: %d, total size: %d bytes)\n",
				stats.DocumentCount, analyzeDocsPath, stats.SectionCount, stats.TotalSize)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to get document stats: %v\n", err)
		}
	}

	return store, nil
}

// formatAndOutputResults formats analysis results and handles output
func formatAndOutputResults(analysis *analyzer.Analysis, correlationResult *correlation.CorrelationResult) error {
	formatterInstance, err := getFormatter(getOutputFormat(), !noColor)
	if err != nil {
		return fmt.Errorf("failed to get formatter: %w", err)
	}

	output, err := formatterInstance.Format(analysis)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Append correlation results if available
	if correlationResult != nil {
		correlationOutput := formatCorrelationResults(correlationResult)
		output = append(output, correlationOutput...)
	}

	return handleOutputDestination(output)
}

// formatCorrelationResults formats correlation results as text.
// Returns formatted output showing document correlations with patterns and direct errors.
func formatCorrelationResults(result *correlation.CorrelationResult) []byte {
	if result == nil || (len(result.Correlations) == 0 && len(result.DirectCorrelations) == 0) {
		return []byte("\n--- Document Correlations ---\nNo correlations found.\n")
	}

	var output strings.Builder
	writeCorrelationHeader(&output, result)

	// Write pattern-based correlations
	if len(result.Correlations) > 0 {
		writeCorrelationDetails(&output, result.Correlations)
	}

	// Write direct error correlations
	if len(result.DirectCorrelations) > 0 {
		writeDirectCorrelationDetails(&output, result.DirectCorrelations)
	}

	return []byte(output.String())
}

// writeCorrelationHeader writes the header section of correlation results.
func writeCorrelationHeader(output *strings.Builder, result *correlation.CorrelationResult) {
	output.WriteString("\n--- Document Correlations ---\n")

	totalCorrelations := result.CorrelatedPatterns + result.CorrelatedErrors
	switch {
	case result.TotalPatterns > 0 && result.TotalErrors > 0:
		fmt.Fprintf(output, "Found %d correlations (%d patterns, %d direct errors)\n\n",
			totalCorrelations, result.CorrelatedPatterns, result.CorrelatedErrors)
	case result.TotalPatterns > 0:
		fmt.Fprintf(output, "Found %d correlations out of %d patterns\n\n",
			result.CorrelatedPatterns, result.TotalPatterns)
	case result.TotalErrors > 0:
		fmt.Fprintf(output, "Found %d direct error correlations\n\n",
			result.CorrelatedErrors)
	}
}

// writeCorrelationDetails writes detailed information for each correlation.
func writeCorrelationDetails(output *strings.Builder, correlations []*correlation.PatternCorrelation) {
	for i, correlation := range correlations {
		writePatternInfo(output, i+1, correlation)
		writeDocumentMatches(output, correlation.DocumentMatches)
		output.WriteString("\n")
	}
}

// writePatternInfo writes pattern information for a single correlation.
func writePatternInfo(output *strings.Builder, index int, patternCorrelation *correlation.PatternCorrelation) {
	fmt.Fprintf(output, "Pattern %d: %s\n", index, patternCorrelation.Pattern.Name)
	fmt.Fprintf(output, "  Description: %s\n", patternCorrelation.Pattern.Description)
	fmt.Fprintf(output, "  Keywords: %s\n", strings.Join(patternCorrelation.Keywords, ", "))
	fmt.Fprintf(output, "  Match Count: %d\n", patternCorrelation.MatchCount)
	output.WriteString("  Related Documents:\n")
}

// writeDocumentMatches writes document match information, limited to top 3 documents.
func writeDocumentMatches(output *strings.Builder, documentMatches []*correlation.DocumentMatch) {
	const maxDocuments = 3
	for j, docMatch := range documentMatches {
		if j >= maxDocuments {
			break
		}
		writeDocumentMatch(output, j+1, docMatch)
	}
}

// writeDocumentMatch writes a single document match with score and details.
func writeDocumentMatch(output *strings.Builder, index int, docMatch *correlation.DocumentMatch) {
	fmt.Fprintf(output, "    %d. %s (Score: %.2f)\n",
		index, docMatch.Document.Title, docMatch.Score)
	fmt.Fprintf(output, "       Path: %s\n", docMatch.Document.Path)
	fmt.Fprintf(output, "       Keywords: %s\n",
		strings.Join(docMatch.MatchedKeywords, ", "))
}

// writeDirectCorrelationDetails writes detailed information for each direct error correlation.
func writeDirectCorrelationDetails(output *strings.Builder, correlations []*correlation.ErrorCorrelation) {
	if len(correlations) > 0 {
		output.WriteString("Direct Error Correlations:\n\n")
	}

	for i, correlation := range correlations {
		writeErrorInfo(output, i+1, correlation)
		writeDocumentMatches(output, correlation.DocumentMatches)
		output.WriteString("\n")
	}
}

// writeErrorInfo writes error information for a single direct correlation.
func writeErrorInfo(output *strings.Builder, index int, errorCorrelation *correlation.ErrorCorrelation) {
	fmt.Fprintf(output, "Error %d: %s\n", index, errorCorrelation.ErrorType)
	fmt.Fprintf(output, "  Message: %s\n", errorCorrelation.Error.Message)
	fmt.Fprintf(output, "  Keywords: %s\n", strings.Join(errorCorrelation.Keywords, ", "))
	fmt.Fprintf(output, "  Match Count: %d\n", errorCorrelation.MatchCount)
	fmt.Fprintf(output, "  Confidence: %.2f\n", errorCorrelation.Confidence)
	output.WriteString("  Related Documents:\n")
}

// handleOutputDestination writes output to file or stdout
func handleOutputDestination(output []byte) error {
	if analyzeOutputFile != "" {
		if err := validateOutputFilePath(analyzeOutputFile); err != nil {
			return fmt.Errorf("invalid output file path: %w", err)
		}

		if err := writeOutputBytesToFile(output, analyzeOutputFile); err != nil {
			return fmt.Errorf("failed to write output to file: %w", err)
		}

		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Output saved to: %s\n", analyzeOutputFile)
		}
	} else {
		fmt.Print(string(output))
	}

	return nil
}

func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty file path")
	}

	cleanPath := filepath.Clean(path)

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", cleanPath)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", cleanPath)
	}

	return nil
}

func validateOutputFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty file path")
	}
	return nil
}

// getFormatter returns the appropriate formatter for the given format
func getFormatter(format string, color bool) (formatter.Formatter, error) {
	switch format {
	case "json":
		return formatter.NewJSON(), nil
	case "markdown", "md":
		return formatter.NewMarkdown(), nil
	case "csv":
		return formatter.NewCSV(), nil
	case "text", "terminal", "":
		return formatter.NewTerminal(color), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// writeOutputBytesToFile writes output to a file with proper error handling
func writeOutputBytesToFile(output []byte, filePath string) error {
	// Validate file path again for security
	cleanPath := filepath.Clean(filePath)

	// Create or truncate the file
	file, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && isVerbose() {
			fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", closeErr)
		}
	}()

	// Write the output
	if _, err := file.Write(output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	// Sync to ensure data is written
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync output file: %w", err)
	}

	return nil
}

// setupMonitoring creates and starts a metrics collector for real-time monitoring
func setupMonitoring(ctx context.Context) (monitor.Collector, error) {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "📊 Starting real-time monitoring...\n")
	}

	// Create monitor configuration
	config := monitor.MonitorConfig{
		CollectionInterval:      500 * time.Millisecond, // High frequency for real-time feel
		RetentionPeriod:         1 * time.Hour,          // Keep data for analysis duration
		MaxDataPoints:           1000,                   // Reasonable limit for analysis
		BufferSize:              100,                    // Small buffer for responsiveness
		EnableMemoryMetrics:     true,
		EnableCPUMetrics:        true,
		EnableProcessingMetrics: true,
	}

	collector := monitor.NewWithConfig(config)

	// Start monitoring
	if err := collector.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start metrics collection: %w", err)
	}

	// Show initial monitoring message
	if analyzeMonitor && !isVerbose() {
		fmt.Fprintf(os.Stderr, "📊 Monitoring enabled - performance metrics will be displayed\n")
	}

	// Start real-time display if monitoring is enabled
	if analyzeMonitor {
		go showRealTimeMetrics(ctx, collector)
	}

	return collector, nil
}

// stopMonitoring gracefully stops monitoring and optionally exports metrics
func stopMonitoring(collector monitor.Collector) error {
	if collector == nil {
		return nil
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "📊 Stopping monitoring...\n")
	}

	// Stop the collector
	if err := collector.Stop(); err != nil {
		return fmt.Errorf("failed to stop monitoring: %w", err)
	}

	// Clear the real-time display line
	if analyzeMonitor {
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Export metrics if file specified
	if analyzeMonitorFile != "" {
		if err := exportMonitoringMetrics(collector, analyzeMonitorFile); err != nil {
			return fmt.Errorf("failed to export metrics: %w", err)
		}
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "📊 Metrics exported to: %s\n", analyzeMonitorFile)
		}
	}

	return nil
}

// performAnalysisWithMonitoring wraps the analysis with monitoring if collector is available
func performAnalysisWithMonitoring(ctx context.Context, entries []*common.LogEntry, patterns []*common.Pattern, collector monitor.Collector) (*analyzer.Analysis, error) {
	if collector != nil {
		// Track the entire analysis operation
		var analysis *analyzer.Analysis
		err := collector.TrackOperationWithError(monitor.OperationAnalyze, func() error {
			// Record input metrics
			metric := monitor.Metric{
				Name:      "analysis.input_lines",
				Type:      monitor.MetricTypeGauge,
				Value:     float64(len(entries)),
				Timestamp: time.Now(),
				Labels: map[string]string{
					"operation": "analyze",
				},
			}
			if err := collector.RecordMetric(&metric); err != nil && isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: failed to record input metrics: %v\n", err)
			}

			var err error
			analysis, err = performAnalysis(ctx, entries, patterns)
			return err
		})
		return analysis, err
	}

	// Fallback to regular analysis if no monitoring
	return performAnalysis(ctx, entries, patterns)
}

// performCorrelationWithMonitoring wraps correlation with monitoring if collector is available
func performCorrelationWithMonitoring(ctx context.Context, analysis *analyzer.Analysis, collector monitor.Collector) (*correlation.CorrelationResult, error) {
	if collector != nil {
		// Track correlation operation
		var result *correlation.CorrelationResult
		err := collector.TrackOperationWithError(monitor.OperationPattern, func() error {
			var err error
			result, err = performCorrelationIfEnabled(ctx, analysis)
			return err
		})
		return result, err
	}

	// Fallback to regular correlation if no monitoring
	return performCorrelationIfEnabled(ctx, analysis)
}

// exportMonitoringMetrics exports collected metrics to a file
func exportMonitoringMetrics(collector monitor.Collector, filename string) error {
	if collector == nil {
		return fmt.Errorf("no metrics collector available")
	}

	// Get final snapshot of metrics
	snapshot := collector.GetSnapshot()

	// Convert to JSON
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Write to file using existing helper
	return writeOutputBytesToFile(data, filename)
}

// showRealTimeMetrics displays live performance metrics during analysis
func showRealTimeMetrics(ctx context.Context, collector monitor.Collector) {
	if collector == nil {
		return
	}

	ticker := time.NewTicker(2 * time.Second) // Update every 2 seconds
	defer ticker.Stop()

	var lastSnapshot monitor.MetricsSnapshot
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !collector.IsRunning() {
				return
			}

			snapshot := collector.GetSnapshot()
			displayMetricsUpdate(&snapshot, &lastSnapshot, startTime)
			lastSnapshot = snapshot
		}
	}
}

// displayMetricsUpdate shows a concise real-time metrics update
func displayMetricsUpdate(current, last *monitor.MetricsSnapshot, startTime time.Time) {
	elapsed := time.Since(startTime)

	// Clear previous line and show updated metrics
	fmt.Fprintf(os.Stderr, "\r📊 [%v] Memory: %s | CPU: %d cores, %d goroutines | Operations: %d",
		formatDuration(elapsed),
		formatBytes(current.Memory.CurrentAlloc),
		current.CPU.NumCPU,
		current.CPU.NumGoroutines,
		len(current.Operations))

	// Show specific operation performance if available
	if len(current.Operations) > 0 {
		var totalDuration time.Duration
		var totalOps int64
		for _, op := range current.Operations {
			// Access timing information directly from OperationMetrics
			totalDuration += time.Duration(op.TotalTime)
			totalOps += op.Count
		}

		if totalDuration > 0 && totalOps > 0 {
			avgDuration := totalDuration / time.Duration(totalOps)
			fmt.Fprintf(os.Stderr, " | Ops: %d, Avg: %v", totalOps, formatDuration(avgDuration))
		}
	}

	// Add performance indicators
	if current.Memory.CurrentAlloc > 100*1024*1024 { // > 100MB
		fmt.Fprintf(os.Stderr, " ⚠️")
	}

	if current.CPU.NumGoroutines > 50 {
		fmt.Fprintf(os.Stderr, " 🔄")
	}
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration in a compact way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
