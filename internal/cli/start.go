package cli

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/docker"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the emulator stack",
	Long: `Start the GCP emulator stack using docker-compose.

This starts IAM, Secret Manager, and KMS emulators with the
configured IAM mode and policy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration (Viper resolves behind the scenes)
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		color.Cyan("Starting GCP Emulator Control Plane...")
		color.Cyan("IAM Mode: %s", cfg.IAMMode)

		// Pull images if requested
		if cfg.PullOnStart {
			color.Cyan("→ Pulling latest images...")
			if err := docker.Pull(); err != nil {
				color.Yellow("⚠ Failed to pull images: %v", err)
			}
		}

		// Start the stack
		if err := docker.Start(cfg); err != nil {
			color.Red("✗ Failed to start stack: %v", err)
			return err
		}

		color.Green("✓ Stack started successfully")
		color.Cyan("\nServices:")
		color.Cyan("  IAM:            http://localhost:%d", cfg.Ports.IAM)
		color.Cyan("  Secret Manager: grpc://localhost:%d, http://localhost:%d", cfg.Ports.SecretManager, cfg.Ports.SecretManager+1)
		color.Cyan("  KMS:            grpc://localhost:%d, http://localhost:%d", cfg.Ports.KMS, cfg.Ports.KMS+1)
		color.Cyan("\nRun 'gcp-emulator status' to check health")

		return nil
	},
}

func init() {
	// Define flags
	startCmd.Flags().String("mode", "", "IAM mode (off|permissive|strict)")
	startCmd.Flags().Bool("pull", false, "Pull latest images before starting")
	startCmd.Flags().BoolP("detach", "d", true, "Run in background")

	// Bind flags to viper
	viper.BindPFlag("iam-mode", startCmd.Flags().Lookup("mode"))
	viper.BindPFlag("pull-on-start", startCmd.Flags().Lookup("pull"))
}
