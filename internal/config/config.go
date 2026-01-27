// Package config provides configuration management for the gcp-emulator CLI.
//
// It implements the disciplined Viper pattern where Viper stays contained
// in this package and the rest of the codebase receives explicit Config structs.
// Configuration sources are resolved in this order: flags > env > config file > defaults.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config is the explicit configuration struct
// This is what the rest of the codebase sees
type Config struct {
	IAMMode     string
	Trace       bool
	PullOnStart bool
	PolicyFile  string
	Ports       PortConfig
}

// PortConfig defines port mappings for all services
type PortConfig struct {
	IAM           int
	SecretManager int
	KMS           int
}

// Init initializes viper with defaults and config file paths
func Init() error {
	// Set config file name and type
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config file search paths
	viper.AddConfigPath("$HOME/.gcp-emulator")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("iam-mode", "permissive")
	viper.SetDefault("trace", false)
	viper.SetDefault("pull-on-start", false)
	viper.SetDefault("policy-file", "./policy.yaml")
	viper.SetDefault("port-iam", 8080)
	viper.SetDefault("port-secret-manager", 9090)
	viper.SetDefault("port-kms", 9091)

	// Bind environment variables with prefix
	viper.SetEnvPrefix("GCP_EMULATOR")
	viper.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return nil
}

// Load reads from all sources and returns explicit Config
func Load() (*Config, error) {
	cfg := &Config{
		IAMMode:     viper.GetString("iam-mode"),
		Trace:       viper.GetBool("trace"),
		PullOnStart: viper.GetBool("pull-on-start"),
		PolicyFile:  viper.GetString("policy-file"),
		Ports: PortConfig{
			IAM:           viper.GetInt("port-iam"),
			SecretManager: viper.GetInt("port-secret-manager"),
			KMS:           viper.GetInt("port-kms"),
		},
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ensures config is sane
func (c *Config) Validate() error {
	if c.IAMMode != "off" && c.IAMMode != "permissive" && c.IAMMode != "strict" {
		return fmt.Errorf("invalid iam-mode: %s (must be off, permissive, or strict)", c.IAMMode)
	}

	if c.Ports.IAM < 1 || c.Ports.IAM > 65535 {
		return fmt.Errorf("invalid IAM port: %d", c.Ports.IAM)
	}

	if c.Ports.SecretManager < 1 || c.Ports.SecretManager > 65535 {
		return fmt.Errorf("invalid Secret Manager port: %d", c.Ports.SecretManager)
	}

	if c.Ports.KMS < 1 || c.Ports.KMS > 65535 {
		return fmt.Errorf("invalid KMS port: %d", c.Ports.KMS)
	}

	return nil
}

// Save writes current config to file
func Save(cfg *Config) error {
	viper.Set("iam-mode", cfg.IAMMode)
	viper.Set("trace", cfg.Trace)
	viper.Set("pull-on-start", cfg.PullOnStart)
	viper.Set("policy-file", cfg.PolicyFile)
	viper.Set("port-iam", cfg.Ports.IAM)
	viper.Set("port-secret-manager", cfg.Ports.SecretManager)
	viper.Set("port-kms", cfg.Ports.KMS)

	return viper.WriteConfig()
}

// Display shows current config (for gcp-emulator config get)
func Display() (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = "(not found)"
	}

	return fmt.Sprintf(`Configuration:
  iam-mode:           %s
  trace:              %t
  pull-on-start:      %t
  policy-file:        %s
  
Ports:
  IAM:                %d
  Secret Manager:     %d
  KMS:                %d
  
Sources:
  Config file:        %s
  Environment:        GCP_EMULATOR_*
  Flags:              (per command)
`,
		cfg.IAMMode,
		cfg.Trace,
		cfg.PullOnStart,
		cfg.PolicyFile,
		cfg.Ports.IAM,
		cfg.Ports.SecretManager,
		cfg.Ports.KMS,
		configFile,
	), nil
}
