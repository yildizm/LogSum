package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/config"
	"github.com/yildizm/LogSum/internal/emoji"
	"github.com/yildizm/LogSum/internal/logger"
)

var (
	cfgFile   string
	verbose   bool
	noColor   bool
	noEmoji   bool
	outputFmt string

	// Global config instance
	globalConfig *config.Config

	// Global logger instance
	mainLogger *logger.Logger
)

// NewRootCommand creates the root command
func NewRootCommand(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "logsum",
		Short: "High-Performance Log Analysis Tool",
		Long: `LogSum is a high-performance log analysis tool that automatically detects patterns,
identifies anomalies, and provides insights from your log data.

It supports multiple log formats (JSON, logfmt, plain text) and can analyze logs
from files or stdin with real-time monitoring capabilities.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load configuration
			loader := config.NewLoader()
			cfg, err := loader.LoadConfig(cfgFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
				cfg = config.DefaultConfig()
			}
			globalConfig = cfg

			// Override config with command line flags if they were explicitly set
			if cmd.Flag("verbose").Changed {
				globalConfig.Output.Verbose = verbose
			} else {
				verbose = globalConfig.Output.Verbose
			}

			if cmd.Flag("output").Changed {
				globalConfig.Output.DefaultFormat = outputFmt
			} else {
				outputFmt = globalConfig.Output.DefaultFormat
			}

			if cmd.Flag("no-color").Changed {
				if noColor {
					globalConfig.Output.ColorMode = "never"
				}
			} else {
				noColor = globalConfig.Output.ColorMode == "never"
			}

			// Auto-disable emojis on Windows if not explicitly set
			if runtime.GOOS == "windows" && !cmd.Flag("no-emoji").Changed {
				noEmoji = true
			}
			// Set emoji state for all components
			emoji.SetEmojiDisabled(noEmoji)
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&noEmoji, "no-emoji", false, "disable emoji output (useful for Windows terminals)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "text", "output format (text, json, markdown)")

	// Add subcommands
	rootCmd.AddCommand(newAnalyzeCommand())
	rootCmd.AddCommand(newPatternsCommand())
	rootCmd.AddCommand(newWatchCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newMonitorCommand())
	rootCmd.AddCommand(newVersionCommand(version, commit, date))

	return rootCmd
}

func newVersionCommand(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display version number, build commit, date, and runtime information",
		Run: func(cmd *cobra.Command, args []string) {
			displayVersion := version
			displayCommit := commit
			displayDate := date

			if version == "dev" || version == "" {
				displayVersion = "development"
			}
			if commit == "none" || commit == "" {
				displayCommit = "local-build"
			}
			if date == "unknown" || date == "" {
				displayDate = "local-build"
			}

			fmt.Printf("LogSum %s (%s) built on %s\n", displayVersion, displayCommit, displayDate)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}

// Global helpers
func isVerbose() bool {
	return verbose
}

// GetLogger returns a logger for the given component
func GetLogger(component string) *logger.Logger {
	if mainLogger == nil {
		mainLogger = logger.NewWithCallback("main", isVerbose)
	}
	return mainLogger.WithComponent(component)
}

func getOutputFormat() string {
	return outputFmt
}

func isEmojiDisabled() bool {
	return noEmoji
}

// GetGlobalConfig returns the loaded global configuration
func GetGlobalConfig() *config.Config {
	if globalConfig == nil {
		return config.DefaultConfig()
	}
	return globalConfig
}
