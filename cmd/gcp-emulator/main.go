package main

import (
	"fmt"
	"os"

	"github.com/blackwell-systems/gcp-iam-control-plane/internal/cli"
	"github.com/blackwell-systems/gcp-iam-control-plane/internal/config"
)

var version = "dev"

func main() {
	// Initialize configuration
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Execute root command
	if err := cli.Execute(version); err != nil {
		os.Exit(1)
	}
}
