package config

import (
	"fmt"
	"time"
)

// Config holds the complete application configuration
type Config struct {
	Version  string         `yaml:"version" json:"version"`
	Patterns PatternConfig  `yaml:"patterns" json:"patterns"`
	AI       AIConfig       `yaml:"ai" json:"ai"`
	Storage  StorageConfig  `yaml:"storage" json:"storage"`
	Output   OutputConfig   `yaml:"output" json:"output"`
	Analysis AnalysisConfig `yaml:"analysis" json:"analysis"`
}

// PatternConfig configures pattern loading and processing
type PatternConfig struct {
	Directories    []string               `yaml:"directories" json:"directories"`
	AutoReload     bool                   `yaml:"auto_reload" json:"auto_reload"`
	CustomPatterns map[string]interface{} `yaml:"custom_patterns" json:"custom_patterns"`
	EnableDefaults bool                   `yaml:"enable_defaults" json:"enable_defaults"`
}

// AIConfig configures AI provider settings
type AIConfig struct {
	Provider   string        `yaml:"provider" json:"provider"`       // ollama|openai|anthropic
	Model      string        `yaml:"model" json:"model"`             // model name/identifier
	Endpoint   string        `yaml:"endpoint" json:"endpoint"`       // API endpoint URL
	APIKey     string        `yaml:"api_key" json:"api_key"`         // API key (support env var reference)
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`         // request timeout
	MaxRetries int           `yaml:"max_retries" json:"max_retries"` // retry count
}

// StorageConfig configures storage and caching
type StorageConfig struct {
	CacheDir     string `yaml:"cache_dir" json:"cache_dir"`           // directory for caches
	IndexPath    string `yaml:"index_path" json:"index_path"`         // document index location
	VectorDBPath string `yaml:"vector_db_path" json:"vector_db_path"` // vector storage location
	TempDir      string `yaml:"temp_dir" json:"temp_dir"`             // temporary file location
}

// OutputConfig configures output formatting and display
type OutputConfig struct {
	DefaultFormat   string `yaml:"default_format" json:"default_format"`     // json|text|markdown|csv
	ColorMode       string `yaml:"color_mode" json:"color_mode"`             // auto|always|never
	Verbose         bool   `yaml:"verbose" json:"verbose"`                   // default verbosity
	TimestampFormat string `yaml:"timestamp_format" json:"timestamp_format"` // time format string
	ShowProgress    bool   `yaml:"show_progress" json:"show_progress"`       // show progress bars
	CompactMode     bool   `yaml:"compact_mode" json:"compact_mode"`         // compact output mode
}

// AnalysisConfig configures analysis behavior
type AnalysisConfig struct {
	MaxEntries      int           `yaml:"max_entries" json:"max_entries"`
	TimelineBuckets int           `yaml:"timeline_buckets" json:"timeline_buckets"`
	EnableInsights  bool          `yaml:"enable_insights" json:"enable_insights"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
	BufferSize      int           `yaml:"buffer_size" json:"buffer_size"`
	MaxLineLength   int           `yaml:"max_line_length" json:"max_line_length"`
	StrictMode      bool          `yaml:"strict_mode" json:"strict_mode"`

	// Context timeout configurations
	VectorTimeout      time.Duration `yaml:"vector_timeout" json:"vector_timeout"`           // Vector operations timeout
	CorrelationTimeout time.Duration `yaml:"correlation_timeout" json:"correlation_timeout"` // Correlation analysis timeout
	IndexingTimeout    time.Duration `yaml:"indexing_timeout" json:"indexing_timeout"`       // Document indexing timeout
	CancelCheckPeriod  int           `yaml:"cancel_check_period" json:"cancel_check_period"` // Iterations between cancellation checks
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Patterns: PatternConfig{
			Directories:    []string{"./patterns", "./examples/patterns"},
			AutoReload:     false,
			CustomPatterns: make(map[string]interface{}),
			EnableDefaults: true,
		},
		AI: AIConfig{
			Provider:   "ollama",
			Model:      "llama3.2",
			Endpoint:   "http://localhost:11434",
			APIKey:     "",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		},
		Storage: StorageConfig{
			CacheDir:     "~/.cache/logsum",
			IndexPath:    "~/.cache/logsum/index.db",
			VectorDBPath: "~/.cache/logsum/vectors.db",
			TempDir:      "/tmp/logsum",
		},
		Output: OutputConfig{
			DefaultFormat:   "text",
			ColorMode:       "auto",
			Verbose:         false,
			TimestampFormat: "2006-01-02 15:04:05",
			ShowProgress:    true,
			CompactMode:     false,
		},
		Analysis: AnalysisConfig{
			MaxEntries:      100000,
			TimelineBuckets: 60,
			EnableInsights:  true,
			Timeout:         60 * time.Second,
			BufferSize:      4096,
			MaxLineLength:   1024 * 1024, // 1MB
			StrictMode:      false,

			// Context timeout defaults
			VectorTimeout:      30 * time.Second,  // Vector search operations
			CorrelationTimeout: 60 * time.Second,  // Error correlation analysis
			IndexingTimeout:    120 * time.Second, // Document indexing (longer for large sets)
			CancelCheckPeriod:  100,               // Check for cancellation every 100 iterations
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.validateAIConfig(); err != nil {
		return err
	}
	if err := c.validateOutputConfig(); err != nil {
		return err
	}
	if err := c.validateAnalysisConfig(); err != nil {
		return err
	}
	if err := c.validateTimeoutConfig(); err != nil {
		return err
	}
	return nil
}

// validateAIConfig validates AI-related configuration
func (c *Config) validateAIConfig() error {
	if c.AI.Provider != "" {
		validProviders := map[string]bool{
			"ollama":    true,
			"openai":    true,
			"anthropic": true,
		}
		if !validProviders[c.AI.Provider] {
			return fmt.Errorf("invalid AI provider: %s (must be one of: ollama, openai, anthropic)", c.AI.Provider)
		}
	}
	if c.AI.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	return nil
}

// validateOutputConfig validates output-related configuration
func (c *Config) validateOutputConfig() error {
	if c.Output.DefaultFormat != "" {
		validFormats := map[string]bool{
			"json":     true,
			"text":     true,
			"markdown": true,
			"csv":      true,
		}
		if !validFormats[c.Output.DefaultFormat] {
			return fmt.Errorf("invalid output format: %s (must be one of: json, text, markdown, csv)", c.Output.DefaultFormat)
		}
	}
	if c.Output.ColorMode != "" {
		validColorModes := map[string]bool{
			"auto":   true,
			"always": true,
			"never":  true,
		}
		if !validColorModes[c.Output.ColorMode] {
			return fmt.Errorf("invalid color mode: %s (must be one of: auto, always, never)", c.Output.ColorMode)
		}
	}
	return nil
}

// validateAnalysisConfig validates analysis-related configuration
func (c *Config) validateAnalysisConfig() error {
	if c.Analysis.MaxEntries < 1 {
		return fmt.Errorf("max_entries must be greater than 0")
	}
	if c.Analysis.TimelineBuckets < 1 {
		return fmt.Errorf("timeline_buckets must be greater than 0")
	}
	if c.Analysis.BufferSize < 1 {
		return fmt.Errorf("buffer_size must be greater than 0")
	}
	if c.Analysis.MaxLineLength < 1 {
		return fmt.Errorf("max_line_length must be greater than 0")
	}
	return nil
}

// validateTimeoutConfig validates timeout-related configuration
func (c *Config) validateTimeoutConfig() error {
	if c.Analysis.VectorTimeout < 0 {
		return fmt.Errorf("vector_timeout must be non-negative")
	}
	if c.Analysis.CorrelationTimeout < 0 {
		return fmt.Errorf("correlation_timeout must be non-negative")
	}
	if c.Analysis.IndexingTimeout < 0 {
		return fmt.Errorf("indexing_timeout must be non-negative")
	}
	if c.Analysis.CancelCheckPeriod < 1 {
		return fmt.Errorf("cancel_check_period must be greater than 0")
	}
	return nil
}
