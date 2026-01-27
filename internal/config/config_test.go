package config

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with off mode",
			config: Config{
				IAMMode:     "off",
				Trace:       false,
				PullOnStart: false,
				PolicyFile:  "policy.yaml",
				Ports: PortConfig{
					IAM:           8080,
					SecretManager: 9090,
					KMS:           9091,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with permissive mode",
			config: Config{
				IAMMode:     "permissive",
				Trace:       true,
				PullOnStart: true,
				PolicyFile:  "policy.yaml",
				Ports: PortConfig{
					IAM:           8080,
					SecretManager: 9090,
					KMS:           9091,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with strict mode",
			config: Config{
				IAMMode:     "strict",
				Trace:       false,
				PullOnStart: false,
				PolicyFile:  "policy.yaml",
				Ports: PortConfig{
					IAM:           8080,
					SecretManager: 9090,
					KMS:           9091,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid IAM mode",
			config: Config{
				IAMMode:     "invalid",
				Trace:       false,
				PullOnStart: false,
				PolicyFile:  "policy.yaml",
				Ports: PortConfig{
					IAM:           8080,
					SecretManager: 9090,
					KMS:           9091,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - zero",
			config: Config{
				IAMMode:     "off",
				Trace:       false,
				PullOnStart: false,
				PolicyFile:  "policy.yaml",
				Ports: PortConfig{
					IAM:           0,
					SecretManager: 9090,
					KMS:           9091,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: Config{
				IAMMode:     "off",
				Trace:       false,
				PullOnStart: false,
				PolicyFile:  "policy.yaml",
				Ports: PortConfig{
					IAM:           8080,
					SecretManager: 70000,
					KMS:           9091,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	// Test that defaults are set correctly
	cfg := &Config{
		IAMMode:     "off",
		Trace:       false,
		PullOnStart: false,
		PolicyFile:  "policy.yaml",
		Ports: PortConfig{
			IAM:           8080,
			SecretManager: 9090,
			KMS:           9091,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}
