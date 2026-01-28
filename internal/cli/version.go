package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gcp-emulator version %s\n", cmd.Root().Version)
		fmt.Println("\nComponents:")
		fmt.Println("  IAM Emulator:      v0.8.0")
		fmt.Println("  Secret Manager:    v1.3.0")
		fmt.Println("  KMS:               v0.3.0")
		fmt.Println("  gcp-emulator-auth: v0.3.0")
	},
}
