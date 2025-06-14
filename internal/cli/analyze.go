package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/analyzer"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/formatter"
	"github.com/yildizm/LogSum/internal/ui"
	"github.com/yildizm/go-logparser"
)

var (
	analyzeFormat     string
	analyzePatterns   string
	analyzeFollow     bool
	analyzeTimeout    time.Duration
	analyzeMaxLines   int
	analyzeNoTUI      bool
	analyzeOutputFile string
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

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
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

	// Load patterns
	patterns := loadAnalysisPatterns()

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

// loadPatternsFromPath loads patterns from a file or directory
func loadPatternsFromPath(path string) ([]*common.Pattern, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	if info.IsDir() {
		return loadPatternsFromDirectory(path)
	}
	return loadPatternsFromFile(path)
}

// loadPatternsFromDirectory loads all pattern files from a directory
func loadPatternsFromDirectory(directory string) ([]*common.Pattern, error) {
	var patterns []*common.Pattern

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			filePatterns, err := loadPatternsFromFile(path)
			if err != nil {
				// Log warning but continue with other files
				return nil
			}
			patterns = append(patterns, filePatterns...)
		}
		return nil
	})

	return patterns, err
}

// loadPatternsFromFile loads patterns from a single YAML file
func loadPatternsFromFile(filename string) ([]*common.Pattern, error) {
	return common.LoadPatternsFromFile(filename)
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

// loadAnalysisPatterns loads patterns based on configuration
func loadAnalysisPatterns() []*common.Pattern {
	var patterns []*common.Pattern

	if analyzePatterns != "" {
		loadedPatterns, err := loadPatternsFromPath(analyzePatterns)
		if err != nil {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: failed to load patterns from %s: %v\n", analyzePatterns, err)
			}
		} else {
			patterns = loadedPatterns
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Loaded %d patterns\n", len(patterns))
			}
		}
	} else {
		// Use embedded default patterns
		loadedPatterns, err := common.LoadDefaultPatterns()
		if err != nil {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: failed to load default patterns: %v\n", err)
			}
		} else {
			patterns = loadedPatterns
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Loaded %d default patterns\n", len(patterns))
			}
		}
	}

	return patterns
}

// runAnalysisAndOutput performs analysis and outputs results
func runAnalysisAndOutput(ctx context.Context, entries []*common.LogEntry, patterns []*common.Pattern) error {
	// Determine if we should use TUI
	shouldUseTUI := !analyzeNoTUI && getOutputFormat() == "text" && !isVerbose()

	if shouldUseTUI {
		// Launch TUI
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Launching interactive terminal UI...\n")
		}
		return ui.InteractiveRun(entries, patterns)
	}

	// Perform CLI analysis
	analysis, err := performAnalysis(ctx, entries, patterns)
	if err != nil {
		return err
	}

	// Format and output results
	return formatAndOutputResults(analysis)
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
		fmt.Fprintf(os.Stderr, "Performing analysis...\n")
	}

	analysis, err := engine.Analyze(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	return analysis, nil
}

// formatAndOutputResults formats analysis results and handles output
func formatAndOutputResults(analysis *analyzer.Analysis) error {
	formatterInstance, err := getFormatter(getOutputFormat(), !noColor)
	if err != nil {
		return fmt.Errorf("failed to get formatter: %w", err)
	}

	output, err := formatterInstance.Format(analysis)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return handleOutputDestination(output)
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
