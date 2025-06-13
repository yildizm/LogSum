package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/go-logparser"
)

var (
	watchPatterns string
)

func newWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch [file]",
		Short: "Watch log files for real-time analysis",
		Long: `Monitor log files for changes and analyze new entries in real-time.

Uses file system notifications to detect changes and processes new log lines
as they are written to the file. Press Ctrl+C to stop watching.

Examples:
  logsum watch app.log
  logsum watch --patterns ./patterns/ access.log`,
		Args: cobra.ExactArgs(1),
		RunE: runWatch,
	}

	cmd.Flags().StringVarP(&watchPatterns, "patterns", "p", "", "pattern file or directory")

	return cmd
}

func runWatch(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Setup file watcher
	watcher, file, cleanup, err := setupFileWatcher(filename)
	if err != nil {
		return err
	}
	defer cleanup()

	// Run watch loop
	return runWatchLoop(watcher, file)
}

func processNewLines(file *os.File, detectedParser logparser.Parser) (logparser.Parser, error) {
	scanner := bufio.NewScanner(file)

	var newLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return detectedParser, fmt.Errorf("scanner error: %w", err)
	}

	if len(newLines) == 0 {
		return detectedParser, nil
	}

	// Auto-detect parser on first lines if not already detected
	if detectedParser == nil {
		detectedParser = logparser.New()
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Created auto-detecting parser\n")
		}
	}

	// Process new lines in batch
	if len(newLines) > 0 {
		entries, err := detectedParser.ParseString(strings.Join(newLines, "\n"))
		if err != nil {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Failed to parse lines: %v\n", err)
			}
			return detectedParser, nil
		}

		// Simple real-time output - just show important entries
		for i, entry := range entries {
			commonEntry := common.ConvertToCommonLogEntry(&entry, i)
			if commonEntry.LogLevel >= common.LevelWarn {
				timestamp := entry.Timestamp.Format("15:04:05")
				fmt.Printf("[%s] %s: %s\n", timestamp, commonEntry.LogLevel.String(), entry.Message)
			}
		}
	}

	return detectedParser, nil
}

// cleanupWatcher safely closes watcher with error logging
func cleanupWatcher(watcher *fsnotify.Watcher) {
	if err := watcher.Close(); err != nil && isVerbose() {
		fmt.Fprintf(os.Stderr, "Warning: failed to close watcher: %v\n", err)
	}
}

// cleanupFile safely closes file with error logging
func cleanupFile(file *os.File) {
	if err := file.Close(); err != nil && isVerbose() {
		fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
	}
}

// createWatcher creates and configures a new file system watcher
func createWatcher(filename string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	if err := watcher.Add(filename); err != nil {
		cleanupWatcher(watcher)
		return nil, fmt.Errorf("failed to watch file: %w", err)
	}

	return watcher, nil
}

// openWatchFile opens and prepares file for watching
func openWatchFile(filename string) (*os.File, error) {
	// #nosec G304 - path is validated by caller
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		cleanupFile(file)
		return nil, fmt.Errorf("failed to seek to end of file: %w", err)
	}

	return file, nil
}

// setupFileWatcher creates and configures file watcher
func setupFileWatcher(filename string) (*fsnotify.Watcher, *os.File, func(), error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("file does not exist: %s", filename)
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Watching file: %s\n", filename)
		fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop...\n\n")
	}

	// Validate file path for security
	if err := validateWatchFilePath(filename); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Create watcher
	watcher, err := createWatcher(filename)
	if err != nil {
		return nil, nil, nil, err
	}

	// Open file
	file, err := openWatchFile(filename)
	if err != nil {
		cleanupWatcher(watcher)
		return nil, nil, nil, err
	}

	cleanup := func() {
		cleanupWatcher(watcher)
		cleanupFile(file)
	}

	return watcher, file, cleanup, nil
}

// runWatchLoop runs the main watch loop with signal handling
func runWatchLoop(watcher *fsnotify.Watcher, file *os.File) error {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Create parser for auto-detection
	var detectedParser logparser.Parser

	// Watch loop
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-signals:
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "\nReceived interrupt signal, stopping...\n")
			}
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}
			updatedParser, err := handleWatchEvent(event, file, detectedParser)
			if err != nil {
				if isVerbose() {
					fmt.Fprintf(os.Stderr, "Error handling event: %v\n", err)
				}
			} else {
				detectedParser = updatedParser
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			}
		}
	}
}

// handleWatchEvent processes file system events
func handleWatchEvent(event fsnotify.Event, file *os.File, detectedParser logparser.Parser) (logparser.Parser, error) {
	// Only process write events
	if event.Op&fsnotify.Write == fsnotify.Write {
		updatedParser, err := processNewLines(file, detectedParser)
		if err != nil {
			return detectedParser, fmt.Errorf("error processing new lines: %w", err)
		}
		return updatedParser, nil
	}
	return detectedParser, nil
}

// validateWatchFilePath validates that a file path is safe to watch
func validateWatchFilePath(path string) error {
	// Check for empty path
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty file path")
	}

	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// For watch operations, ensure the file exists and is a regular file
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("cannot watch directory, must be a file")
	}

	return nil
}
