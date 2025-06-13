package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/common"
)

func newPatternsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "Manage analysis patterns",
		Long: `Manage and validate analysis patterns used for log analysis.

Patterns are YAML files that define log patterns to detect errors, anomalies,
performance issues, and security events.`,
	}

	cmd.AddCommand(newPatternsListCommand())
	cmd.AddCommand(newPatternsValidateCommand())

	return cmd
}

func newPatternsListCommand() *cobra.Command {
	var directory string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available patterns",
		Long: `List all patterns found in the patterns directory.

Shows pattern ID, name, type, and description for each pattern file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPatternsList(directory)
		},
	}

	cmd.Flags().StringVarP(&directory, "directory", "d", "configs/patterns", "patterns directory")

	return cmd
}

func newPatternsValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [file...]",
		Short: "Validate pattern files",
		Long: `Validate one or more pattern YAML files.

Checks YAML syntax and verifies required fields are present.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPatternsValidate(args)
		},
	}

	return cmd
}

func runPatternsList(directory string) error {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Scanning patterns directory: %s\n", directory)
	}

	patterns, err := loadPatternsFromDirectoryCmd(directory)
	if err != nil {
		return fmt.Errorf("failed to load patterns: %w", err)
	}

	if len(patterns) == 0 {
		fmt.Printf("No patterns found in %s\n", directory)
		return nil
	}

	fmt.Printf("Found %d patterns in %s:\n\n", len(patterns), directory)

	// Group by type for better organization
	typeGroups := make(map[common.PatternType][]*common.Pattern)
	for _, pattern := range patterns {
		typeGroups[pattern.Type] = append(typeGroups[pattern.Type], pattern)
	}

	for patternType, typePatterns := range typeGroups {
		fmt.Printf("%s Patterns:\n", strings.ToUpper(string(patternType)[:1])+string(patternType)[1:])
		for _, pattern := range typePatterns {
			fmt.Printf("  %-20s %s\n", pattern.ID, pattern.Name)
			if pattern.Description != "" {
				fmt.Printf("  %-20s %s\n", "", pattern.Description)
			}
		}
		fmt.Println()
	}

	return nil
}

func runPatternsValidate(files []string) error {
	allValid := true

	for _, file := range files {
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Validating: %s\n", file)
		}

		valid, err := validatePatternFile(file)
		switch {
		case err != nil:
			fmt.Printf("%s %s: %v\n", GetEmoji("error"), file, err)
			allValid = false
		case valid:
			fmt.Printf("%s %s: Valid\n", GetEmoji("success"), file)
		default:
			fmt.Printf("%s %s: Invalid (see errors above)\n", GetEmoji("error"), file)
			allValid = false
		}
	}

	if !allValid {
		return fmt.Errorf("some pattern files are invalid")
	}

	return nil
}

func loadPatternsFromDirectoryCmd(directory string) ([]*common.Pattern, error) {
	return common.LoadDefaultPatterns()
}

func loadPatternsFromFileCmd(filename string) ([]*common.Pattern, error) {
	return common.LoadPatternsFromFile(filename)
}

func validatePatternFile(filename string) (bool, error) {
	patterns, err := loadPatternsFromFileCmd(filename)
	if err != nil {
		return false, err
	}

	valid := true
	for i, pattern := range patterns {
		if err := validatePattern(pattern, i); err != nil {
			fmt.Printf("  Pattern %d: %v\n", i, err)
			valid = false
		}
	}

	return valid, nil
}

func validatePattern(pattern *common.Pattern, index int) error {
	if pattern.ID == "" {
		return fmt.Errorf("missing required field: id")
	}

	if pattern.Name == "" {
		return fmt.Errorf("missing required field: name")
	}

	if pattern.Type == "" {
		return fmt.Errorf("missing required field: type")
	}

	// Validate pattern type
	validTypes := map[common.PatternType]bool{
		common.PatternTypeError:       true,
		common.PatternTypeAnomaly:     true,
		common.PatternTypePerformance: true,
		common.PatternTypeSecurity:    true,
	}

	if !validTypes[pattern.Type] {
		return fmt.Errorf("invalid pattern type: %s. Valid types: error, anomaly, performance, security", pattern.Type)
	}

	// Must have either regex or keywords
	if pattern.Regex == "" && len(pattern.Keywords) == 0 {
		return fmt.Errorf("pattern must have either regex or keywords")
	}

	return nil
}
