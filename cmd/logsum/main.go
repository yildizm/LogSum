package main

import (
	"os"

	"github.com/yildizm/LogSum/internal/cli"
)

// Build variables set by ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd := cli.NewRootCommand(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
