package config

import (
	"testing"
	"time"
)

func TestTimeoutConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	// Test that timeout defaults are set correctly
	if config.Analysis.VectorTimeout != 30*time.Second {
		t.Errorf("Expected VectorTimeout to be 30s, got %v", config.Analysis.VectorTimeout)
	}

	if config.Analysis.CorrelationTimeout != 60*time.Second {
		t.Errorf("Expected CorrelationTimeout to be 60s, got %v", config.Analysis.CorrelationTimeout)
	}

	if config.Analysis.IndexingTimeout != 120*time.Second {
		t.Errorf("Expected IndexingTimeout to be 120s, got %v", config.Analysis.IndexingTimeout)
	}

	if config.Analysis.CancelCheckPeriod != 100 {
		t.Errorf("Expected CancelCheckPeriod to be 100, got %v", config.Analysis.CancelCheckPeriod)
	}
}

func TestTimeoutValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid timeouts",
			config: &Config{
				Analysis: AnalysisConfig{
					MaxEntries:         100,
					TimelineBuckets:    10,
					BufferSize:         1024,
					MaxLineLength:      1024,
					VectorTimeout:      30 * time.Second,
					CorrelationTimeout: 60 * time.Second,
					IndexingTimeout:    120 * time.Second,
					CancelCheckPeriod:  100,
				},
			},
			wantErr: false,
		},
		{
			name: "negative vector timeout",
			config: &Config{
				Analysis: AnalysisConfig{
					MaxEntries:         100,
					TimelineBuckets:    10,
					BufferSize:         1024,
					MaxLineLength:      1024,
					VectorTimeout:      -1 * time.Second,
					CorrelationTimeout: 60 * time.Second,
					IndexingTimeout:    120 * time.Second,
					CancelCheckPeriod:  100,
				},
			},
			wantErr: true,
			errMsg:  "vector_timeout must be non-negative",
		},
		{
			name: "negative correlation timeout",
			config: &Config{
				Analysis: AnalysisConfig{
					MaxEntries:         100,
					TimelineBuckets:    10,
					BufferSize:         1024,
					MaxLineLength:      1024,
					VectorTimeout:      30 * time.Second,
					CorrelationTimeout: -1 * time.Second,
					IndexingTimeout:    120 * time.Second,
					CancelCheckPeriod:  100,
				},
			},
			wantErr: true,
			errMsg:  "correlation_timeout must be non-negative",
		},
		{
			name: "zero cancel check period",
			config: &Config{
				Analysis: AnalysisConfig{
					MaxEntries:         100,
					TimelineBuckets:    10,
					BufferSize:         1024,
					MaxLineLength:      1024,
					VectorTimeout:      30 * time.Second,
					CorrelationTimeout: 60 * time.Second,
					IndexingTimeout:    120 * time.Second,
					CancelCheckPeriod:  0,
				},
			},
			wantErr: true,
			errMsg:  "cancel_check_period must be greater than 0",
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
