package common

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

//go:embed embedded_patterns.yaml
var defaultPatternsYAML []byte

// LoadPatternsFromFile loads patterns from a single YAML file
func LoadPatternsFromFile(filename string) ([]*Pattern, error) {
	// Validate and sanitize file path for security
	if err := validatePatternFilePath(filename); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// #nosec G304 - path is validated above
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try to parse as single pattern first
	var pattern Pattern
	if err := yaml.Unmarshal(data, &pattern); err == nil && pattern.ID != "" {
		return []*Pattern{&pattern}, nil
	}

	// Try to parse as array of patterns
	var patterns []*Pattern
	if err := yaml.Unmarshal(data, &patterns); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return patterns, nil
}

// validatePatternFilePath validates that a pattern file path is safe to read
func validatePatternFilePath(path string) error {
	// Check for empty path
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty file path")
	}

	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts beyond reasonable bounds
	if strings.Contains(cleanPath, "..") && !isConfigPath(cleanPath) {
		return fmt.Errorf("path traversal not allowed")
	}

	// For pattern files, allow YAML/YML extensions
	ext := strings.ToLower(filepath.Ext(cleanPath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("pattern files must have .yaml or .yml extension")
	}

	return nil
}

// isConfigPath checks if the path is a legitimate config path (for default patterns)
func isConfigPath(path string) bool {
	// Allow relative paths to configs directory for default patterns
	return strings.HasPrefix(path, "configs/") ||
		strings.HasPrefix(path, "./configs/") ||
		strings.HasPrefix(path, "../configs/")
}

// LoadDefaultPatterns loads embedded default patterns
func LoadDefaultPatterns() ([]*Pattern, error) {
	var patterns []*Pattern
	if err := yaml.Unmarshal(defaultPatternsYAML, &patterns); err != nil {
		return nil, fmt.Errorf("failed to parse embedded default patterns: %w", err)
	}
	return patterns, nil
}

// LoadPatternsWithFallback loads embedded default patterns
func LoadPatternsWithFallback(directory string) ([]*Pattern, error) {
	// Always use embedded default patterns
	return LoadDefaultPatterns()
}
