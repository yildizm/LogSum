package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test that defaults are set correctly
	if cfg.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", cfg.Version)
	}

	if cfg.AI.Provider != "ollama" {
		t.Errorf("Expected AI provider ollama, got %s", cfg.AI.Provider)
	}

	if cfg.Output.DefaultFormat != "text" {
		t.Errorf("Expected output format text, got %s", cfg.Output.DefaultFormat)
	}

	if cfg.Analysis.MaxEntries != 100000 {
		t.Errorf("Expected max entries 100000, got %d", cfg.Analysis.MaxEntries)
	}

	if len(cfg.Patterns.Directories) != 2 {
		t.Errorf("Expected 2 pattern directories, got %d", len(cfg.Patterns.Directories))
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid AI provider",
			config: &Config{
				AI: AIConfig{Provider: "invalid"},
			},
			wantErr: true,
			errMsg:  "invalid AI provider: invalid (must be one of: ollama, openai, anthropic)",
		},
		{
			name: "invalid output format",
			config: &Config{
				Output: OutputConfig{DefaultFormat: "invalid"},
			},
			wantErr: true,
			errMsg:  "invalid output format: invalid (must be one of: json, text, markdown, csv)",
		},
		{
			name: "invalid color mode",
			config: &Config{
				Output: OutputConfig{ColorMode: "invalid"},
			},
			wantErr: true,
			errMsg:  "invalid color mode: invalid (must be one of: auto, always, never)",
		},
		{
			name: "invalid max entries",
			config: &Config{
				Analysis: AnalysisConfig{MaxEntries: 0},
			},
			wantErr: true,
			errMsg:  "max_entries must be greater than 0",
		},
		{
			name: "invalid timeline buckets",
			config: &Config{
				Analysis: AnalysisConfig{
					MaxEntries:      100,
					TimelineBuckets: 0,
				},
			},
			wantErr: true,
			errMsg:  "timeline_buckets must be greater than 0",
		},
		{
			name: "negative max retries",
			config: &Config{
				AI: AIConfig{MaxRetries: -1},
				Analysis: AnalysisConfig{
					MaxEntries:      100,
					TimelineBuckets: 10,
					BufferSize:      1024,
					MaxLineLength:   1024,
				},
			},
			wantErr: true,
			errMsg:  "max_retries must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigMerging(t *testing.T) {
	// Create base config
	dst := DefaultConfig()
	dst.AI.Provider = "ollama"
	dst.Output.DefaultFormat = "text"

	// Create source config to merge
	src := &Config{
		AI: AIConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Output: OutputConfig{
			DefaultFormat: "json",
			Verbose:       true,
		},
	}

	// Merge configs
	mergeConfigs(dst, src)

	// Check that values were merged correctly
	if dst.AI.Provider != "openai" {
		t.Errorf("Expected AI provider openai, got %s", dst.AI.Provider)
	}
	if dst.AI.Model != "gpt-4" {
		t.Errorf("Expected AI model gpt-4, got %s", dst.AI.Model)
	}
	if dst.Output.DefaultFormat != "json" {
		t.Errorf("Expected output format json, got %s", dst.Output.DefaultFormat)
	}
	if !dst.Output.Verbose {
		t.Errorf("Expected verbose to be true")
	}

	// Check that unset values in source don't override destination
	if dst.AI.Timeout != 30*time.Second {
		t.Errorf("Expected AI timeout to remain 30s, got %v", dst.AI.Timeout)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "relative path",
			input:    "./config.yaml",
			expected: "./config.yaml",
		},
		{
			name:     "absolute path",
			input:    "/etc/logsum/config.yaml",
			expected: "/etc/logsum/config.yaml",
		},
		{
			name:     "home directory path",
			input:    "~/.config/logsum/config.yaml",
			expected: "~/.config/logsum/config.yaml", // Will be expanded in real usage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if tt.input == "~/.config/logsum/config.yaml" {
				// For tilde expansion, just check it's different from input
				if result == tt.input {
					t.Errorf("Expected path to be expanded, but got same path")
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestGetConfigPaths(t *testing.T) {
	paths := GetConfigPaths()
	if len(paths) != 3 {
		t.Errorf("Expected 3 config paths, got %d", len(paths))
	}

	expectedPaths := []string{
		"./.logsum.yaml",
		"~/.config/logsum/config.yaml",
		"/etc/logsum/config.yaml",
	}

	for i, expectedPath := range expectedPaths {
		if i < len(paths) {
			// For paths with ~, just check that expansion occurred
			if expectedPath == "~/.config/logsum/config.yaml" {
				if paths[i] == expectedPath {
					t.Errorf("Expected path %s to be expanded", expectedPath)
				}
			} else {
				if paths[i] != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, paths[i])
				}
			}
		}
	}
}
