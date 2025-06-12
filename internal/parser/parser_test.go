package parser

import (
	"strings"
	"testing"
)

func TestJSONParser(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *LogEntry)
	}{
		{
			name:  "standard JSON log",
			input: `{"timestamp":"2024-01-02T15:04:05Z","level":"ERROR","message":"Database connection failed","service":"api"}`,
			validate: func(t *testing.T, e *LogEntry) {
				if e.Level != LevelError {
					t.Errorf("want level ERROR, got %v", e.Level)
				}
				if e.Message != "Database connection failed" {
					t.Errorf("want message 'Database connection failed', got %s", e.Message)
				}
				if e.Service != "api" {
					t.Errorf("want service 'api', got %s", e.Service)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil && entry != nil {
				tt.validate(t, entry)
			}
		})
	}
}

func TestLogfmtParser(t *testing.T) {
	parser := NewLogfmtParser()

	tests := []struct {
		name     string
		input    string
		validate func(*testing.T, *LogEntry)
	}{
		{
			name:  "standard logfmt",
			input: `time=2024-01-02T15:04:05Z level=error msg="Connection timeout" service=worker duration=1.23`,
			validate: func(t *testing.T, e *LogEntry) {
				if e.Level != LevelError {
					t.Errorf("want level ERROR, got %v", e.Level)
				}
				if e.Message != "Connection timeout" {
					t.Errorf("want message 'Connection timeout', got %s", e.Message)
				}
				if e.Metadata["duration"] != "1.23" {
					t.Errorf("want duration=1.23, got %s", e.Metadata["duration"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, _ := parser.Parse(tt.input)
			if tt.validate != nil && entry != nil {
				tt.validate(t, entry)
			}
		})
	}
}

func TestTextParser(t *testing.T) {
	parser := NewTextParser()

	tests := []struct {
		name     string
		input    string
		validate func(*testing.T, *LogEntry)
	}{
		{
			name:  "bracketed level",
			input: `2024-01-02 15:04:05 [ERROR] Failed to connect to database`,
			validate: func(t *testing.T, e *LogEntry) {
				if e.Level != LevelError {
					t.Errorf("want level ERROR, got %v", e.Level)
				}
				if !strings.Contains(e.Message, "Failed to connect") {
					t.Errorf("message should contain 'Failed to connect'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, _ := parser.Parse(tt.input)
			if tt.validate != nil && entry != nil {
				tt.validate(t, entry)
			}
		})
	}
}

func TestFormatDetection(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name    string
		samples []string
		want    string
	}{
		{
			name: "JSON logs",
			samples: []string{
				`{"level":"info","msg":"test"}`,
				`{"level":"error","msg":"test2"}`,
			},
			want: "json",
		},
		{
			name: "logfmt logs",
			samples: []string{
				`level=info msg="test" time=2024-01-02T15:04:05Z`,
				`level=error msg="test2" time=2024-01-02T15:04:06Z`,
			},
			want: "logfmt",
		},
		{
			name: "text logs",
			samples: []string{
				`[INFO] Starting application`,
				`[ERROR] Connection failed`,
			},
			want: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := factory.DetectFormat(tt.samples)
			if err != nil {
				t.Errorf("DetectFormat() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
