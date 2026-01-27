package docker

import (
	"fmt"
	"net/http"
	"time"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
)

// ServiceStatus represents the status of a service
type ServiceStatus int

const (
	ServiceUnknown ServiceStatus = iota
	ServiceUp
	ServiceDown
	ServiceStarting
)

// StackStatus represents the status of all services
type StackStatus struct {
	IAM           ServiceStatus
	SecretManager ServiceStatus
	KMS           ServiceStatus
}

// Status returns health status of all services
func Status(cfg *config.Config) (*StackStatus, error) {
	status := &StackStatus{}

	// Check IAM health
	status.IAM = checkHealth(fmt.Sprintf("http://localhost:%d/health", cfg.Ports.IAM))

	// Check Secret Manager health (HTTP port is gRPC port + 1)
	status.SecretManager = checkHealth(fmt.Sprintf("http://localhost:%d/health", cfg.Ports.SecretManager+1))

	// Check KMS health (HTTP port is gRPC port + 1)
	status.KMS = checkHealth(fmt.Sprintf("http://localhost:%d/health", cfg.Ports.KMS+1))

	return status, nil
}

func checkHealth(url string) ServiceStatus {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return ServiceDown
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return ServiceUp
	}

	return ServiceDown
}
