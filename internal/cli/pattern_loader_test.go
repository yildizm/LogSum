package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/config"
)

func TestNewPatternLoader(t *testing.T) {
	loader := NewPatternLoader()
	if loader == nil {
		t.Fatal("NewPatternLoader() should return a valid PatternLoader instance")
	}
}

func TestConvertSeverity(t *testing.T) {
	loader := NewPatternLoader()

	tests := []struct {
		input    string
		expected common.LogLevel
	}{
		{"debug", common.LevelDebug},
		{"DEBUG", common.LevelDebug},
		{"info", common.LevelInfo},
		{"INFO", common.LevelInfo},
		{"warn", common.LevelWarn},
		{"warning", common.LevelWarn},
		{"WARN", common.LevelWarn},
		{"error", common.LevelError},
		{"ERROR", common.LevelError},
		{"fatal", common.LevelFatal},
		{"FATAL", common.LevelFatal},
		{"unknown", common.LevelInfo}, // default
		{"", common.LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := loader.convertSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("convertSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertSinglePattern(t *testing.T) {
	loader := NewPatternLoader()

	tests := []struct {
		name        string
		patternName string
		patternMap  map[string]interface{}
		expectNil   bool
		expectedID  string
	}{
		{
			name:        "valid pattern",
			patternName: "test-pattern",
			patternMap: map[string]interface{}{
				"pattern":     "ERROR.*database",
				"severity":    "error",
				"description": "Database error pattern",
			},
			expectNil:  false,
			expectedID: "test-pattern",
		},
		{
			name:        "missing regex",
			patternName: "no-regex",
			patternMap: map[string]interface{}{
				"severity":    "error",
				"description": "Pattern without regex",
			},
			expectNil: true,
		},
		{
			name:        "empty regex",
			patternName: "empty-regex",
			patternMap: map[string]interface{}{
				"pattern":     "",
				"severity":    "error",
				"description": "Pattern with empty regex",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.convertSinglePattern(tt.patternName, tt.patternMap)

			if tt.expectNil && result != nil {
				t.Errorf("convertSinglePattern() should return nil for %s", tt.name)
			}
			if !tt.expectNil && result == nil {
				t.Errorf("convertSinglePattern() should not return nil for %s", tt.name)
			}
			if result != nil && result.ID != tt.expectedID {
				t.Errorf("convertSinglePattern() ID = %v, want %v", result.ID, tt.expectedID)
			}
		})
	}
}

func TestConvertCustomPatterns(t *testing.T) {
	loader := NewPatternLoader()

	customPatterns := map[string]interface{}{
		"valid-pattern": map[string]interface{}{
			"pattern":     "ERROR.*test",
			"severity":    "error",
			"description": "Test error pattern",
		},
		"invalid-pattern": "not-a-map", // This should be skipped
		"no-regex-pattern": map[string]interface{}{
			"severity":    "warn",
			"description": "Pattern without regex", // This should be skipped
		},
	}

	result := loader.convertCustomPatterns(customPatterns)

	// Should only get 1 valid pattern
	if len(result) != 1 {
		t.Errorf("convertCustomPatterns() should return 1 pattern, got %d", len(result))
	}

	if len(result) > 0 {
		pattern := result[0]
		if pattern.ID != "valid-pattern" {
			t.Errorf("Expected pattern ID 'valid-pattern', got %s", pattern.ID)
		}
		if pattern.Regex != "ERROR.*test" {
			t.Errorf("Expected regex 'ERROR.*test', got %s", pattern.Regex)
		}
	}
}

func TestLoadPatternsFromConfig(t *testing.T) {
	loader := NewPatternLoader()

	// Test with empty config
	emptyConfig := &config.Config{
		Patterns: config.PatternConfig{
			Directories:    []string{},
			CustomPatterns: map[string]interface{}{},
			EnableDefaults: false,
		},
	}

	result := loader.loadPatternsFromConfig(emptyConfig)
	if len(result) != 0 {
		t.Errorf("Expected no patterns for empty config, got %d", len(result))
	}

	// Test with default patterns enabled
	defaultConfig := &config.Config{
		Patterns: config.PatternConfig{
			Directories:    []string{},
			CustomPatterns: map[string]interface{}{},
			EnableDefaults: true,
		},
	}

	result = loader.loadPatternsFromConfig(defaultConfig)
	// Should load default patterns (exact count depends on implementation)
	if len(result) == 0 {
		t.Errorf("Expected default patterns to be loaded, got %d", len(result))
	}
}

func TestLoadPatternsFromDirectories(t *testing.T) {
	loader := NewPatternLoader()

	// Test with non-existent directories
	nonExistentDirs := []string{"/non/existent/path1", "/non/existent/path2"}
	result := loader.loadPatternsFromDirectories(nonExistentDirs)

	if len(result) != 0 {
		t.Errorf("Expected no patterns from non-existent directories, got %d", len(result))
	}

	// Test with empty slice
	emptyDirs := []string{}
	result = loader.loadPatternsFromDirectories(emptyDirs)

	if len(result) != 0 {
		t.Errorf("Expected no patterns from empty directories list, got %d", len(result))
	}
}

func TestLoadAnalysisPatterns(t *testing.T) {
	loader := NewPatternLoader()

	// Save original values
	oldAnalyzePatterns := analyzePatterns
	oldGlobalConfig := globalConfig

	defer func() {
		analyzePatterns = oldAnalyzePatterns
		globalConfig = oldGlobalConfig
	}()

	// Test with flag set to non-existent path
	analyzePatterns = "/non/existent/pattern/file.yaml"
	globalConfig = &config.Config{
		Patterns: config.PatternConfig{
			EnableDefaults: false,
		},
	}

	result := loader.LoadAnalysisPatterns()
	// Should return empty since path doesn't exist
	if len(result) != 0 {
		t.Errorf("Expected no patterns for non-existent flag path, got %d", len(result))
	}

	// Test with no flag (should use config)
	analyzePatterns = ""
	globalConfig = &config.Config{
		Patterns: config.PatternConfig{
			EnableDefaults: true,
		},
	}

	result = loader.LoadAnalysisPatterns()
	// Should load default patterns
	if len(result) == 0 {
		t.Errorf("Expected default patterns when no flag set, got %d", len(result))
	}
}

func TestLoadPatternsFromPath(t *testing.T) {
	loader := NewPatternLoader()

	// Test with non-existent path
	_, err := loader.loadPatternsFromPath("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test-pattern-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	// Write a simple YAML pattern (array format)
	content := `- id: test-pattern
  name: Test Pattern
  regex: "ERROR.*test"
  severity: 3
  description: Test pattern for unit tests
`
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading from file
	patterns, err := loader.loadPatternsFromPath(tempFile.Name())
	if err != nil {
		t.Errorf("loadPatternsFromPath() error = %v", err)
	}

	// Note: The actual loading depends on common.LoadPatternsFromFile implementation
	// This test mainly verifies the path handling logic
	_ = patterns // Use the variable to avoid unused warning
}

func TestLoadPatternsFromDirectory(t *testing.T) {
	loader := NewPatternLoader()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-patterns-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create some test files
	yamlFile := filepath.Join(tempDir, "test.yaml")
	nonYamlFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(yamlFile, []byte("# test yaml"), 0o600); err != nil {
		t.Fatalf("Failed to create yaml file: %v", err)
	}
	if err := os.WriteFile(nonYamlFile, []byte("test text"), 0o600); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Test loading from directory
	patterns, err := loader.loadPatternsFromDirectory(tempDir)
	if err != nil {
		t.Errorf("loadPatternsFromDirectory() error = %v", err)
	}

	// The function should attempt to load .yaml files
	// Actual loading success depends on file content and common.LoadPatternsFromFile
	_ = patterns // Use the variable to avoid unused warning
}

// Test verbose logging behavior
func TestVerboseLogging(t *testing.T) {
	// Save original verbose state
	oldVerbose := verbose
	defer func() {
		verbose = oldVerbose
	}()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	// Enable verbose mode
	verbose = true

	loader := NewPatternLoader()

	// Test convert single pattern with missing regex (should log warning)
	pattern := loader.convertSinglePattern("test", map[string]any{
		"description": "test pattern without regex",
	})

	if pattern != nil {
		t.Error("Expected nil pattern for missing regex")
	}

	// Close writer and read output
	if err := w.Close(); err != nil {
		t.Errorf("Failed to close pipe writer: %v", err)
	}
	output := make([]byte, 1000)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if !strings.Contains(outputStr, "Warning: pattern test has no regex") {
		t.Error("Expected verbose warning for pattern without regex")
	}
}
