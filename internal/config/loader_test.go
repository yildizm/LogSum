package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if len(loader.configPaths) != 3 {
		t.Errorf("Expected 3 config paths, got %d", len(loader.configPaths))
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	loader := NewLoader()

	// Test loading with no config files (should use defaults)
	cfg, err := loader.LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Verify it's using defaults
	if cfg.AI.Provider != "ollama" {
		t.Errorf("Expected default AI provider ollama, got %s", cfg.AI.Provider)
	}
	if cfg.Output.DefaultFormat != "text" {
		t.Errorf("Expected default output format text, got %s", cfg.Output.DefaultFormat)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `version: "1.0"
ai:
  provider: "openai"
  model: "gpt-4"
  timeout: 60s
output:
  default_format: "json"
  verbose: true
analysis:
  max_entries: 50000
`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	// Verify the config was loaded correctly
	if cfg.AI.Provider != "openai" {
		t.Errorf("Expected AI provider openai, got %s", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gpt-4" {
		t.Errorf("Expected AI model gpt-4, got %s", cfg.AI.Model)
	}
	if cfg.AI.Timeout != 60*time.Second {
		t.Errorf("Expected AI timeout 60s, got %v", cfg.AI.Timeout)
	}
	if cfg.Output.DefaultFormat != "json" {
		t.Errorf("Expected output format json, got %s", cfg.Output.DefaultFormat)
	}
	if !cfg.Output.Verbose {
		t.Errorf("Expected verbose to be true")
	}
	if cfg.Analysis.MaxEntries != 50000 {
		t.Errorf("Expected max entries 50000, got %d", cfg.Analysis.MaxEntries)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	invalidConfigContent := `version: "1.0"
ai:
  provider: "openai"
  model: "gpt-4"
  timeout: 60s
  # Invalid YAML - missing closing quote
output:
  default_format: "json
  verbose: true
`

	err := os.WriteFile(configPath, []byte(invalidConfigContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	loader := NewLoader()
	_, err = loader.LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error loading invalid YAML config, but got none")
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	// Set environment variables
	envVars := map[string]string{
		"LOGSUM_AI_PROVIDER":          "anthropic",
		"LOGSUM_AI_MODEL":             "claude-3",
		"LOGSUM_OUTPUT_VERBOSE":       "true",
		"LOGSUM_ANALYSIS_MAX_ENTRIES": "25000",
		"LOGSUM_PATTERNS_DIRECTORIES": "dir1,dir2,dir3",
	}

	// Set environment variables
	for key, value := range envVars {
		_ = os.Setenv(key, value)
	}

	// Clean up environment variables after test
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	loader := NewLoader()
	cfg := DefaultConfig()

	err := loader.applyEnvOverrides(cfg)
	if err != nil {
		t.Fatalf("Failed to apply env overrides: %v", err)
	}

	// Check that environment variables were applied
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("Expected AI provider anthropic, got %s", cfg.AI.Provider)
	}
	if cfg.AI.Model != "claude-3" {
		t.Errorf("Expected AI model claude-3, got %s", cfg.AI.Model)
	}
	if !cfg.Output.Verbose {
		t.Errorf("Expected verbose to be true")
	}
	if cfg.Analysis.MaxEntries != 25000 {
		t.Errorf("Expected max entries 25000, got %d", cfg.Analysis.MaxEntries)
	}
	if len(cfg.Patterns.Directories) != 3 {
		t.Errorf("Expected 3 pattern directories, got %d", len(cfg.Patterns.Directories))
	}
	expectedDirs := []string{"dir1", "dir2", "dir3"}
	for i, expectedDir := range expectedDirs {
		if i < len(cfg.Patterns.Directories) && cfg.Patterns.Directories[i] != expectedDir {
			t.Errorf("Expected pattern directory %s, got %s", expectedDir, cfg.Patterns.Directories[i])
		}
	}
}

func TestApplyEnvOverridesInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid int", "LOGSUM_ANALYSIS_MAX_ENTRIES", "not-a-number"},
		{"invalid bool", "LOGSUM_OUTPUT_VERBOSE", "not-a-bool"},
		{"invalid duration", "LOGSUM_AI_TIMEOUT", "not-a-duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(tt.envVar, tt.value)
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			loader := NewLoader()
			cfg := DefaultConfig()

			err := loader.applyEnvOverrides(cfg)
			if err == nil {
				t.Error("Expected error for invalid env var value, but got none")
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	var duration time.Duration

	err := parseDuration("30s", &duration)
	if err != nil {
		t.Errorf("Failed to parse duration: %v", err)
	}
	if duration != 30*time.Second {
		t.Errorf("Expected 30s, got %v", duration)
	}

	err = parseDuration("invalid", &duration)
	if err == nil {
		t.Error("Expected error for invalid duration, but got none")
	}
}

func TestParseInt(t *testing.T) {
	var value int

	err := parseInt("42", &value)
	if err != nil {
		t.Errorf("Failed to parse int: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}

	err = parseInt("not-a-number", &value)
	if err == nil {
		t.Error("Expected error for invalid int, but got none")
	}
}

func TestParseBool(t *testing.T) {
	var value bool

	err := parseBool("true", &value)
	if err != nil {
		t.Errorf("Failed to parse bool: %v", err)
	}
	if !value {
		t.Errorf("Expected true, got %v", value)
	}

	err = parseBool("false", &value)
	if err != nil {
		t.Errorf("Failed to parse bool: %v", err)
	}
	if value {
		t.Errorf("Expected false, got %v", value)
	}

	err = parseBool("not-a-bool", &value)
	if err == nil {
		t.Error("Expected error for invalid bool, but got none")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test when no config file exists
	_, found := FindConfigFile()
	if found {
		t.Error("Expected no config file to be found, but one was found")
	}

	// Create a temporary config file in current directory
	tempConfigPath := "./.logsum.yaml"
	err := os.WriteFile(tempConfigPath, []byte("version: 1.0"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer func() { _ = os.Remove(tempConfigPath) }()

	configPath, found := FindConfigFile()
	if !found {
		t.Error("Expected config file to be found, but none was found")
	}
	if configPath != tempConfigPath {
		t.Errorf("Expected config path %s, got %s", tempConfigPath, configPath)
	}
}

func TestFileExists(t *testing.T) {
	// Test with non-existent file
	if fileExists("/path/that/does/not/exist") {
		t.Error("Expected file to not exist, but fileExists returned true")
	}

	// Create a temporary file
	tempFile := filepath.Join(t.TempDir(), "test-file")
	err := os.WriteFile(tempFile, []byte("test"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if !fileExists(tempFile) {
		t.Error("Expected file to exist, but fileExists returned false")
	}
}

func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid yaml file",
			path:    "config.yaml",
			wantErr: false,
		},
		{
			name:    "valid yml file",
			path:    "config.yml",
			wantErr: false,
		},
		{
			name:    "path traversal attempt",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal not allowed",
		},
		{
			name:    "non-yaml file",
			path:    "config.txt",
			wantErr: true,
			errMsg:  "config file must have .yaml or .yml extension",
		},
		{
			name:    "system file access",
			path:    "/etc/passwd.yaml",
			wantErr: true,
			errMsg:  "access to system files not allowed",
		},
		{
			name:    "proc filesystem access",
			path:    "/proc/version.yaml",
			wantErr: true,
			errMsg:  "access to system files not allowed",
		},
		{
			name:    "relative path with valid extension",
			path:    "./configs/app.yaml",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
