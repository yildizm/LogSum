package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/config"
	"gopkg.in/yaml.v3"
)

// newConfigCommand creates the config command with subcommands
func newConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage LogSum configuration",
		Long: `Manage LogSum configuration files and settings.
		
The config command provides subcommands for initializing, viewing,
validating, and managing configuration files.`,
	}

	// Add subcommands
	configCmd.AddCommand(newConfigInitCommand())
	configCmd.AddCommand(newConfigShowCommand())
	configCmd.AddCommand(newConfigValidateCommand())
	configCmd.AddCommand(newConfigPathCommand())

	return configCmd
}

// newConfigInitCommand creates the config init subcommand
func newConfigInitCommand() *cobra.Command {
	var (
		outputPath string
		minimal    bool
		force      bool
	)

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long: `Initialize a new LogSum configuration file with default values.
		
By default, creates a full configuration file with all options and comments.
Use --minimal for a compact configuration with only essential settings.`,
		Example: `  # Create full config in current directory
  logsum config init

  # Create minimal config
  logsum config init --minimal

  # Create config at specific path
  logsum config init --output ~/.config/logsum/config.yaml

  # Overwrite existing config
  logsum config init --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine output path
			if outputPath == "" {
				outputPath = ".logsum.yaml"
			}

			// Check if file exists and not forcing
			if !force && fileExists(outputPath) {
				return fmt.Errorf("config file already exists at %s (use --force to overwrite)", outputPath)
			}

			// Create directory if needed
			dir := filepath.Dir(outputPath)
			if dir != "." && dir != "/" {
				if err := os.MkdirAll(dir, 0o750); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", dir, err)
				}
			}

			// Get config content
			var content string
			if minimal {
				content = config.MinimalSampleConfig()
			} else {
				content = config.SampleConfig()
			}

			// Write config file
			if err := os.WriteFile(outputPath, []byte(content), 0o600); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			fmt.Printf("✅ Configuration file created at: %s\n", outputPath)
			if minimal {
				fmt.Println("📄 Created minimal configuration with essential settings")
			} else {
				fmt.Println("📄 Created full configuration with all options and documentation")
			}

			return nil
		},
	}

	initCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for config file (default: .logsum.yaml)")
	initCmd.Flags().BoolVarP(&minimal, "minimal", "m", false, "create minimal configuration")
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing config file")

	return initCmd
}

// newConfigShowCommand creates the config show subcommand
func newConfigShowCommand() *cobra.Command {
	var (
		format     string
		configPath string
	)

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		Long: `Display the current effective configuration after loading from all sources.
		
Shows the merged configuration from all sources including defaults,
config files, and environment variable overrides.`,
		Example: `  # Show config in YAML format
  logsum config show

  # Show config in JSON format
  logsum config show --format json

  # Show config from specific file
  logsum config show --config /path/to/config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			loader := config.NewLoader()
			cfg, err := loader.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Format and display configuration
			switch format {
			case "json":
				data, err := json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal config to JSON: %w", err)
				}
				fmt.Println(string(data))
			case "yaml":
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return fmt.Errorf("failed to marshal config to YAML: %w", err)
				}
				fmt.Print(string(data))
			default:
				return fmt.Errorf("unsupported format: %s (use json or yaml)", format)
			}

			return nil
		},
	}

	showCmd.Flags().StringVarP(&format, "format", "f", "yaml", "output format (yaml, json)")
	showCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")

	return showCmd
}

// newConfigValidateCommand creates the config validate subcommand
func newConfigValidateCommand() *cobra.Command {
	var configPath string

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate a LogSum configuration file for syntax and semantic errors.
		
Checks the configuration file for:
- Valid YAML syntax
- Required fields
- Valid values for enums
- Proper data types`,
		Example: `  # Validate current config
  logsum config validate

  # Validate specific config file  
  logsum config validate --config /path/to/config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			loader := config.NewLoader()
			cfg, err := loader.LoadConfig(configPath)
			if err != nil {
				fmt.Printf("❌ Configuration validation failed:\n")
				fmt.Printf("   %v\n", err)
				return err
			}

			// If we get here, validation passed
			fmt.Println("✅ Configuration is valid")

			// Show some basic info about the config
			fmt.Printf("📊 Configuration summary:\n")
			fmt.Printf("   Version: %s\n", cfg.Version)
			fmt.Printf("   AI Provider: %s\n", cfg.AI.Provider)
			fmt.Printf("   Output Format: %s\n", cfg.Output.DefaultFormat)
			fmt.Printf("   Pattern Directories: %d configured\n", len(cfg.Patterns.Directories))

			return nil
		},
	}

	validateCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")

	return validateCmd
}

// newConfigPathCommand creates the config path subcommand
func newConfigPathCommand() *cobra.Command {
	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file search paths",
		Long: `Display the list of paths LogSum searches for configuration files.
		
Shows the search order and indicates which files exist.`,
		Example: `  # Show config search paths
  logsum config path`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("📁 Configuration file search paths (in priority order):")
			fmt.Println()

			paths := config.GetConfigPaths()
			for i, path := range paths {
				priority := []string{"Highest", "Medium", "Lowest"}
				exists := ""
				if fileExists(path) {
					exists = " ✅ (exists)"
				} else {
					exists = " ❌ (not found)"
				}

				fmt.Printf("  %d. %s%s\n", i+1, path, exists)
				if i < len(priority) {
					fmt.Printf("     Priority: %s\n", priority[i])
				}
				fmt.Println()
			}

			// Show current config file being used
			if currentConfig, found := config.FindConfigFile(); found {
				fmt.Printf("🎯 Current config file: %s\n", currentConfig)
			} else {
				fmt.Println("📝 No config file found, using defaults")
			}

			fmt.Println()
			fmt.Println("💡 Environment variables with LOGSUM_ prefix will override file settings")
		},
	}

	return pathCmd
}

// Helper function to check if file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
