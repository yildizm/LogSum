package config

import (
	"time"
)

// Config holds application configuration
type Config struct {
	// Parser settings
	Parser ParserConfig `yaml:"parser" json:"parser"`

	// Analyzer settings
	Analyzer AnalyzerConfig `yaml:"analyzer" json:"analyzer"`

	// UI settings
	UI UIConfig `yaml:"ui" json:"ui"`

	// Pattern settings
	Patterns PatternConfig `yaml:"patterns" json:"patterns"`
}

// ParserConfig configures parsing behavior
type ParserConfig struct {
	MaxLineLength int           `yaml:"max_line_length" json:"max_line_length"`
	BufferSize    int           `yaml:"buffer_size" json:"buffer_size"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	StrictMode    bool          `yaml:"strict_mode" json:"strict_mode"`
}

// AnalyzerConfig configures analysis behavior
type AnalyzerConfig struct {
	MaxEntries      int           `yaml:"max_entries" json:"max_entries"`
	TimelineBuckets int           `yaml:"timeline_buckets" json:"timeline_buckets"`
	EnableInsights  bool          `yaml:"enable_insights" json:"enable_insights"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
}

// UIConfig configures UI behavior
type UIConfig struct {
	Theme        string `yaml:"theme" json:"theme"`
	ShowProgress bool   `yaml:"show_progress" json:"show_progress"`
	CompactMode  bool   `yaml:"compact_mode" json:"compact_mode"`
	ColorOutput  bool   `yaml:"color_output" json:"color_output"`
}

// PatternConfig configures pattern loading
type PatternConfig struct {
	Directory      string   `yaml:"directory" json:"directory"`
	Files          []string `yaml:"files" json:"files"`
	EnableDefaults bool     `yaml:"enable_defaults" json:"enable_defaults"`
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	// Implementation to be added
	return DefaultConfig(), nil
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Parser: ParserConfig{
			MaxLineLength: 1024 * 1024, // 1MB
			BufferSize:    4096,
			Timeout:       30 * time.Second,
			StrictMode:    false,
		},
		Analyzer: AnalyzerConfig{
			MaxEntries:      100000,
			TimelineBuckets: 60,
			EnableInsights:  true,
			Timeout:         60 * time.Second,
		},
		UI: UIConfig{
			Theme:        "default",
			ShowProgress: true,
			CompactMode:  false,
			ColorOutput:  true,
		},
		Patterns: PatternConfig{
			Directory:      "configs/patterns",
			EnableDefaults: true,
		},
	}
}
