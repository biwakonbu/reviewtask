package main

import (
	"fmt"
	"os"

	"gh-review-task/cmd"
)

// Build-time variables (set via -ldflags)
var (
	version    = "dev"
	commitHash = "unknown"
	buildDate  = "unknown"
)

func main() {
	// Set version information for the CLI
	cmd.SetVersionInfo(version, commitHash, buildDate)
	
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
