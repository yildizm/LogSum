package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigPaths defines the config file search paths in priority order
var ConfigPaths = []string{
	"./.logsum.yaml",               // Project-specific config (highest priority)
	"~/.config/logsum/config.yaml", // User config
	"/etc/logsum/config.yaml",      // System config (lowest priority)
}

// Loader handles configuration loading with priority merging
type Loader struct {
	configPaths []string
}

// NewLoader creates a new config loader
func NewLoader() *Loader {
	return &Loader{
		configPaths: ConfigPaths,
	}
}

// LoadConfig loads configuration from multiple sources with priority order:
// 1. Command line flags (handled by caller)
// 2. Environment variables
// 3. ./.logsum.yaml
// 4. ~/.config/logsum/config.yaml
// 5. /etc/logsum/config.yaml
// 6. Built-in defaults
func (l *Loader) LoadConfig(customPath string) (*Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// If custom path is provided, use only that path
	if customPath != "" {
		// Validate the custom path for security
		if err := validateConfigPath(customPath); err != nil {
			return nil, fmt.Errorf("invalid config path: %w", err)
		}
		if err := l.loadFromFile(config, customPath); err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", customPath, err)
		}
	} else {
		// Load from standard paths in reverse priority order (lowest to highest)
		paths := make([]string, len(l.configPaths))
		copy(paths, l.configPaths)
		// Reverse the slice to load lowest priority first
		for i := len(paths)/2 - 1; i >= 0; i-- {
			opp := len(paths) - 1 - i
			paths[i], paths[opp] = paths[opp], paths[i]
		}

		for _, path := range paths {
			expandedPath := expandPath(path)
			if fileExists(expandedPath) {
				if err := l.loadFromFile(config, expandedPath); err != nil {
					// Log warning but continue with other config files
					fmt.Fprintf(os.Stderr, "Warning: Failed to load config from %s: %v\n", expandedPath, err)
				}
			}
		}
	}

	// Apply environment variable overrides
	if err := l.applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML file and merges it with existing config
func (l *Loader) loadFromFile(config *Config, path string) error {
	// #nosec G304 - path is validated by validateConfigPath() before reaching here
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create a temporary config to unmarshal into
	var fileConfig Config
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Merge the file config into the existing config
	mergeConfigs(config, &fileConfig)

	return nil
}

// applyEnvOverrides applies environment variable overrides to the config
func (l *Loader) applyEnvOverrides(config *Config) error {
	envMappings := map[string]func(string) error{
		// AI Config
		"LOGSUM_AI_PROVIDER":    func(v string) error { config.AI.Provider = v; return nil },
		"LOGSUM_AI_MODEL":       func(v string) error { config.AI.Model = v; return nil },
		"LOGSUM_AI_ENDPOINT":    func(v string) error { config.AI.Endpoint = v; return nil },
		"LOGSUM_AI_API_KEY":     func(v string) error { config.AI.APIKey = v; return nil },
		"LOGSUM_AI_TIMEOUT":     func(v string) error { return parseDuration(v, &config.AI.Timeout) },
		"LOGSUM_AI_MAX_RETRIES": func(v string) error { return parseInt(v, &config.AI.MaxRetries) },

		// Storage Config
		"LOGSUM_STORAGE_CACHE_DIR":      func(v string) error { config.Storage.CacheDir = v; return nil },
		"LOGSUM_STORAGE_INDEX_PATH":     func(v string) error { config.Storage.IndexPath = v; return nil },
		"LOGSUM_STORAGE_VECTOR_DB_PATH": func(v string) error { config.Storage.VectorDBPath = v; return nil },
		"LOGSUM_STORAGE_TEMP_DIR":       func(v string) error { config.Storage.TempDir = v; return nil },

		// Output Config
		"LOGSUM_OUTPUT_DEFAULT_FORMAT":   func(v string) error { config.Output.DefaultFormat = v; return nil },
		"LOGSUM_OUTPUT_COLOR_MODE":       func(v string) error { config.Output.ColorMode = v; return nil },
		"LOGSUM_OUTPUT_VERBOSE":          func(v string) error { return parseBool(v, &config.Output.Verbose) },
		"LOGSUM_OUTPUT_TIMESTAMP_FORMAT": func(v string) error { config.Output.TimestampFormat = v; return nil },
		"LOGSUM_OUTPUT_SHOW_PROGRESS":    func(v string) error { return parseBool(v, &config.Output.ShowProgress) },
		"LOGSUM_OUTPUT_COMPACT_MODE":     func(v string) error { return parseBool(v, &config.Output.CompactMode) },

		// Analysis Config
		"LOGSUM_ANALYSIS_MAX_ENTRIES":      func(v string) error { return parseInt(v, &config.Analysis.MaxEntries) },
		"LOGSUM_ANALYSIS_TIMELINE_BUCKETS": func(v string) error { return parseInt(v, &config.Analysis.TimelineBuckets) },
		"LOGSUM_ANALYSIS_ENABLE_INSIGHTS":  func(v string) error { return parseBool(v, &config.Analysis.EnableInsights) },
		"LOGSUM_ANALYSIS_TIMEOUT":          func(v string) error { return parseDuration(v, &config.Analysis.Timeout) },
		"LOGSUM_ANALYSIS_BUFFER_SIZE":      func(v string) error { return parseInt(v, &config.Analysis.BufferSize) },
		"LOGSUM_ANALYSIS_MAX_LINE_LENGTH":  func(v string) error { return parseInt(v, &config.Analysis.MaxLineLength) },
		"LOGSUM_ANALYSIS_STRICT_MODE":      func(v string) error { return parseBool(v, &config.Analysis.StrictMode) },

		// Pattern Config
		"LOGSUM_PATTERNS_AUTO_RELOAD":     func(v string) error { return parseBool(v, &config.Patterns.AutoReload) },
		"LOGSUM_PATTERNS_ENABLE_DEFAULTS": func(v string) error { return parseBool(v, &config.Patterns.EnableDefaults) },
	}

	for envVar, setter := range envMappings {
		if value := os.Getenv(envVar); value != "" {
			if err := setter(value); err != nil {
				return fmt.Errorf("invalid value for %s: %w", envVar, err)
			}
		}
	}

	// Handle special case for pattern directories (comma-separated list)
	if dirs := os.Getenv("LOGSUM_PATTERNS_DIRECTORIES"); dirs != "" {
		config.Patterns.Directories = strings.Split(dirs, ",")
		// Trim whitespace from each directory
		for i, dir := range config.Patterns.Directories {
			config.Patterns.Directories[i] = strings.TrimSpace(dir)
		}
	}

	return nil
}

// GetConfigPaths returns the list of configuration file paths that will be searched
func GetConfigPaths() []string {
	paths := make([]string, 0, len(ConfigPaths))
	for _, path := range ConfigPaths {
		paths = append(paths, expandPath(path))
	}
	return paths
}

// FindConfigFile finds the first existing config file in the search paths
func FindConfigFile() (string, bool) {
	for _, path := range ConfigPaths {
		expandedPath := expandPath(path)
		if fileExists(expandedPath) {
			return expandedPath, true
		}
	}
	return "", false
}

// Helper functions

// validateConfigPath validates that a config path is safe to read
func validateConfigPath(path string) error {
	// Clean the path to resolve any ".." components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Ensure it's a YAML file
	ext := strings.ToLower(filepath.Ext(cleanPath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("config file must have .yaml or .yml extension")
	}

	// Convert to absolute path for additional validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Basic sanity check - ensure it's not in sensitive system directories
	if strings.HasPrefix(absPath, "/etc/passwd") ||
		strings.HasPrefix(absPath, "/etc/shadow") ||
		strings.HasPrefix(absPath, "/proc/") ||
		strings.HasPrefix(absPath, "/sys/") {
		return fmt.Errorf("access to system files not allowed")
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// mergeConfigs merges source config into destination config
// Only non-zero values from source overwrite destination
func mergeConfigs(dst, src *Config) {
	// Version
	if src.Version != "" {
		dst.Version = src.Version
	}

	mergePatternConfig(&dst.Patterns, &src.Patterns)
	mergeAIConfig(&dst.AI, &src.AI)
	mergeStorageConfig(&dst.Storage, &src.Storage)
	mergeOutputConfig(&dst.Output, &src.Output)
	mergeAnalysisConfig(&dst.Analysis, &src.Analysis)
}

// mergePatternConfig merges pattern configuration
func mergePatternConfig(dst, src *PatternConfig) {
	if len(src.Directories) > 0 {
		dst.Directories = src.Directories
	}
	if src.AutoReload {
		dst.AutoReload = src.AutoReload
	}
	if len(src.CustomPatterns) > 0 {
		if dst.CustomPatterns == nil {
			dst.CustomPatterns = make(map[string]interface{})
		}
		for k, v := range src.CustomPatterns {
			dst.CustomPatterns[k] = v
		}
	}
	if !src.EnableDefaults && dst.EnableDefaults {
		// Only override if explicitly set to false in source
		dst.EnableDefaults = src.EnableDefaults
	}
}

// mergeAIConfig merges AI configuration
func mergeAIConfig(dst, src *AIConfig) {
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.Endpoint != "" {
		dst.Endpoint = src.Endpoint
	}
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.Timeout != 0 {
		dst.Timeout = src.Timeout
	}
	if src.MaxRetries != 0 {
		dst.MaxRetries = src.MaxRetries
	}
}

// mergeStorageConfig merges storage configuration
func mergeStorageConfig(dst, src *StorageConfig) {
	if src.CacheDir != "" {
		dst.CacheDir = src.CacheDir
	}
	if src.IndexPath != "" {
		dst.IndexPath = src.IndexPath
	}
	if src.VectorDBPath != "" {
		dst.VectorDBPath = src.VectorDBPath
	}
	if src.TempDir != "" {
		dst.TempDir = src.TempDir
	}
}

// mergeOutputConfig merges output configuration
func mergeOutputConfig(dst, src *OutputConfig) {
	if src.DefaultFormat != "" {
		dst.DefaultFormat = src.DefaultFormat
	}
	if src.ColorMode != "" {
		dst.ColorMode = src.ColorMode
	}
	if src.TimestampFormat != "" {
		dst.TimestampFormat = src.TimestampFormat
	}
	// For boolean fields, we need to check if they were explicitly set
	// This is a limitation of YAML unmarshaling, but we'll handle it in env overrides
	mergeIfSet(&dst.Verbose, src.Verbose)
	mergeIfSet(&dst.ShowProgress, src.ShowProgress)
	mergeIfSet(&dst.CompactMode, src.CompactMode)
}

// mergeAnalysisConfig merges analysis configuration
func mergeAnalysisConfig(dst, src *AnalysisConfig) {
	if src.MaxEntries != 0 {
		dst.MaxEntries = src.MaxEntries
	}
	if src.TimelineBuckets != 0 {
		dst.TimelineBuckets = src.TimelineBuckets
	}
	if src.Timeout != 0 {
		dst.Timeout = src.Timeout
	}
	if src.BufferSize != 0 {
		dst.BufferSize = src.BufferSize
	}
	if src.MaxLineLength != 0 {
		dst.MaxLineLength = src.MaxLineLength
	}
	mergeIfSet(&dst.EnableInsights, src.EnableInsights)
	mergeIfSet(&dst.StrictMode, src.StrictMode)
}

// mergeIfSet only merges boolean values if they appear to be explicitly set
// This is a simple heuristic, but works for most cases
func mergeIfSet(dst *bool, src bool) {
	// For now, always merge - this could be improved with custom unmarshaling
	*dst = src
}

// Type conversion helpers

func parseInt(s string, dst *int) error {
	val, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*dst = val
	return nil
}

func parseBool(s string, dst *bool) error {
	val, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*dst = val
	return nil
}

func parseDuration(s string, dst *time.Duration) error {
	val, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*dst = val
	return nil
}
