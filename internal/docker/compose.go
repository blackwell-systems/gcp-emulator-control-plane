package docker

import (
	"fmt"
	"os/exec"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
)

// Start starts the docker compose stack
func Start(cfg *config.Config) error {
	// Generate environment variables for docker-compose
	env := []string{
		fmt.Sprintf("IAM_MODE=%s", cfg.IAMMode),
		fmt.Sprintf("IAM_PORT=%d", cfg.Ports.IAM),
		fmt.Sprintf("SECRET_MANAGER_PORT=%d", cfg.Ports.SecretManager),
		fmt.Sprintf("KMS_PORT=%d", cfg.Ports.KMS),
	}

	// Run docker-compose up
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Env = append(cmd.Environ(), env...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker-compose up failed: %w\n%s", err, output)
	}

	return nil
}

// Stop stops the docker compose stack
func Stop() error {
	cmd := exec.Command("docker-compose", "down")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker-compose down failed: %w\n%s", err, output)
	}

	return nil
}

// Pull pulls the latest images
func Pull() error {
	cmd := exec.Command("docker-compose", "pull")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker-compose pull failed: %w\n%s", err, output)
	}

	return nil
}
