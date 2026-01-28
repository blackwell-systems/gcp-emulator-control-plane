package docker

import (
	"fmt"
	"net/http"
	"time"

	"github.com/blackwell-systems/gcp-iam-control-plane/internal/config"
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

	// Check IAM health (health server on gRPC port + 1000)
	status.IAM = checkHealth(fmt.Sprintf("http://localhost:%d/health", cfg.Ports.IAM+1000))

	// Check Secret Manager health (HTTP port is 8081, mapped from container 8080)
	status.SecretManager = checkHealth("http://localhost:8081/health")

	// Check KMS health (HTTP port is 8082, mapped from container 8080)
	status.KMS = checkHealth("http://localhost:8082/health")

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
