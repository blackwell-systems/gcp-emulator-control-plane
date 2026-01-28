package cli

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/blackwell-systems/gcp-iam-control-plane/internal/config"
	"github.com/blackwell-systems/gcp-iam-control-plane/internal/docker"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all services",
	Long:  `Display health status of IAM, Secret Manager, and KMS emulators.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		status, err := docker.Status(cfg)
		if err != nil {
			color.Red("✗ Failed to get status: %v", err)
			return err
		}

		// Print status
		color.Cyan("Service          Status    Ports")
		color.Cyan("────────────────────────────────────────")

		printServiceStatus("IAM Emulator", status.IAM, cfg.Ports.IAM)
		printServiceStatus("Secret Manager", status.SecretManager, cfg.Ports.SecretManager)
		printServiceStatus("KMS", status.KMS, cfg.Ports.KMS)

		return nil
	},
}

func printServiceStatus(name string, status docker.ServiceStatus, port int) {
	var statusText string
	switch status {
	case docker.ServiceUp:
		statusText = color.GreenString("✓ UP")
	case docker.ServiceDown:
		statusText = color.RedString("✗ DOWN")
	case docker.ServiceStarting:
		statusText = color.YellowString("⚠ STARTING")
	default:
		statusText = color.RedString("✗ UNKNOWN")
	}

	color.New().Printf("%-16s %s       %d\n", name, statusText, port)
}
