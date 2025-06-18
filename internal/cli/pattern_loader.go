package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/config"
)

// PatternLoader handles loading and processing of log analysis patterns from various sources.
type PatternLoader struct{}

// NewPatternLoader creates a new pattern loader instance.
func NewPatternLoader() *PatternLoader {
	return &PatternLoader{}
}

// LoadAnalysisPatterns loads patterns based on configuration precedence:
// 1. Command line flag patterns (highest priority)
// 2. Config file directory patterns
// 3. Config file custom patterns
// 4. Default embedded patterns (lowest priority)
func (pl *PatternLoader) LoadAnalysisPatterns() []*common.Pattern {
	cfg := GetGlobalConfig()

	// Check if patterns flag was explicitly set
	if analyzePatterns != "" {
		return pl.loadPatternsFromFlag()
	}

	return pl.loadPatternsFromConfig(cfg)
}

// loadPatternsFromFlag loads patterns from the command line flag.
func (pl *PatternLoader) loadPatternsFromFlag() []*common.Pattern {
	loadedPatterns, err := pl.loadPatternsFromPath(analyzePatterns)
	if err != nil {
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Warning: failed to load patterns from %s: %v\n", analyzePatterns, err)
		}
		return nil
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Loaded %d patterns from flag\n", len(loadedPatterns))
	}
	return loadedPatterns
}

// loadPatternsFromConfig loads patterns from configuration settings.
func (pl *PatternLoader) loadPatternsFromConfig(cfg *config.Config) []*common.Pattern {
	var patterns []*common.Pattern

	// Load patterns from configured directories
	patterns = append(patterns, pl.loadPatternsFromDirectories(cfg.Patterns.Directories)...)

	// Add custom patterns from config
	if len(cfg.Patterns.CustomPatterns) > 0 {
		customPatterns := pl.convertCustomPatterns(cfg.Patterns.CustomPatterns)
		patterns = append(patterns, customPatterns...)
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Loaded %d custom patterns from config\n", len(customPatterns))
		}
	}

	// Use embedded default patterns if enabled and no patterns loaded
	if cfg.Patterns.EnableDefaults && len(patterns) == 0 {
		patterns = pl.loadDefaultPatterns()
	}

	return patterns
}

// loadPatternsFromDirectories loads patterns from configured directories.
func (pl *PatternLoader) loadPatternsFromDirectories(directories []string) []*common.Pattern {
	var patterns []*common.Pattern

	for _, dir := range directories {
		if _, err := os.Stat(dir); err == nil {
			loadedPatterns, err := pl.loadPatternsFromPath(dir)
			if err != nil {
				if isVerbose() {
					fmt.Fprintf(os.Stderr, "Warning: failed to load patterns from %s: %v\n", dir, err)
				}
			} else {
				patterns = append(patterns, loadedPatterns...)
				if isVerbose() {
					fmt.Fprintf(os.Stderr, "Loaded %d patterns from %s\n", len(loadedPatterns), dir)
				}
			}
		}
	}

	return patterns
}

// loadDefaultPatterns loads embedded default patterns.
func (pl *PatternLoader) loadDefaultPatterns() []*common.Pattern {
	loadedPatterns, err := common.LoadDefaultPatterns()
	if err != nil {
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Warning: failed to load default patterns: %v\n", err)
		}
		return nil
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Loaded %d default patterns\n", len(loadedPatterns))
	}
	return loadedPatterns
}

// loadPatternsFromPath loads patterns from a file or directory.
func (pl *PatternLoader) loadPatternsFromPath(path string) ([]*common.Pattern, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	if info.IsDir() {
		return pl.loadPatternsFromDirectory(path)
	}
	return pl.loadPatternsFromFile(path)
}

// loadPatternsFromDirectory loads all pattern files from a directory.
func (pl *PatternLoader) loadPatternsFromDirectory(directory string) ([]*common.Pattern, error) {
	var patterns []*common.Pattern

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			filePatterns, err := pl.loadPatternsFromFile(path)
			if err != nil {
				// Log specific error instead of silently continuing
				if isVerbose() {
					fmt.Fprintf(os.Stderr, "Warning: failed to load pattern file %s: %v\n", path, err)
				}
				return nil // Continue with other files
			}
			patterns = append(patterns, filePatterns...)
		}
		return nil
	})

	return patterns, err
}

// loadPatternsFromFile loads patterns from a single YAML file.
func (pl *PatternLoader) loadPatternsFromFile(filename string) ([]*common.Pattern, error) {
	return common.LoadPatternsFromFile(filename)
}

// convertCustomPatterns converts config custom patterns to common.Pattern format.
func (pl *PatternLoader) convertCustomPatterns(customPatterns map[string]interface{}) []*common.Pattern {
	var patterns []*common.Pattern

	for name, patternData := range customPatterns {
		patternMap, ok := patternData.(map[string]interface{})
		if !ok {
			if isVerbose() {
				fmt.Fprintf(os.Stderr, "Warning: invalid pattern data for %s\n", name)
			}
			continue
		}

		pattern := pl.convertSinglePattern(name, patternMap)
		if pattern != nil {
			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

// convertSinglePattern converts a single pattern map to common.Pattern.
func (pl *PatternLoader) convertSinglePattern(name string, patternMap map[string]interface{}) *common.Pattern {
	pattern := &common.Pattern{
		ID:   name,
		Name: name,
	}

	if patternStr, ok := patternMap["pattern"].(string); ok {
		pattern.Regex = patternStr
	}
	if severityStr, ok := patternMap["severity"].(string); ok {
		pattern.Severity = pl.convertSeverity(severityStr)
	}
	if description, ok := patternMap["description"].(string); ok {
		pattern.Description = description
	}

	if pattern.Regex != "" {
		return pattern
	}

	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Warning: pattern %s has no regex, skipping\n", name)
	}
	return nil
}

// convertSeverity converts string severity to LogLevel.
func (pl *PatternLoader) convertSeverity(severityStr string) common.LogLevel {
	switch strings.ToLower(severityStr) {
	case "debug":
		return common.LevelDebug
	case "info":
		return common.LevelInfo
	case "warn", "warning":
		return common.LevelWarn
	case "error":
		return common.LevelError
	case "fatal":
		return common.LevelFatal
	default:
		return common.LevelInfo
	}
}
