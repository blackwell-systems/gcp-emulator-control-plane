package cli

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/docker"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the emulator stack",
	Long:  `Stop all running emulator services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("Stopping GCP Emulator Control Plane...")

		if err := docker.Stop(); err != nil {
			color.Red("✗ Failed to stop stack: %v", err)
			return err
		}

		color.Green("✓ Stack stopped successfully")
		return nil
	},
}
