// Package policy provides policy file parsing and validation for the GCP emulator ecosystem.
//
// It validates IAM policy structure including roles, groups, projects, bindings,
// and CEL conditions. The validator ensures permission format correctness and
// catches common configuration errors before runtime.
//
// Supports both YAML (.yaml, .yml) and JSON (.json) policy files for maximum flexibility.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Policy represents the policy file structure
type Policy struct {
	Roles    map[string]Role       `yaml:"roles" json:"roles"`
	Groups   map[string]Group      `yaml:"groups" json:"groups"`
	Projects map[string]Project    `yaml:"projects" json:"projects"`
}

// Role represents a custom role with permissions
type Role struct {
	Permissions []string `yaml:"permissions" json:"permissions"`
}

// Group represents a group with members
type Group struct {
	Members []string `yaml:"members" json:"members"`
}

// Project represents a project with IAM bindings
type Project struct {
	Bindings []Binding `yaml:"bindings" json:"bindings"`
}

// Binding represents an IAM binding
type Binding struct {
	Role      string     `yaml:"role" json:"role"`
	Members   []string   `yaml:"members" json:"members"`
	Condition *Condition `yaml:"condition,omitempty" json:"condition,omitempty"`
}

// Condition represents a CEL condition
type Condition struct {
	Expression  string `yaml:"expression" json:"expression"`
	Title       string `yaml:"title,omitempty" json:"title,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Load loads and parses a policy file (supports .yaml, .yml, and .json)
func Load(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	
	// Detect format by file extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &policy); err != nil {
			return nil, fmt.Errorf("failed to parse policy JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &policy); err != nil {
			return nil, fmt.Errorf("failed to parse policy YAML: %w", err)
		}
	default:
		// Try YAML as fallback for backwards compatibility
		if err := yaml.Unmarshal(data, &policy); err != nil {
			return nil, fmt.Errorf("failed to parse policy (unknown extension %s, tried YAML): %w", ext, err)
		}
	}

	return &policy, nil
}

// Save saves policy to file (format determined by file extension)
func Save(policy *Policy, path string) error {
	var data []byte
	var err error
	
	// Detect format by file extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(policy, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal policy JSON: %w", err)
		}
	case ".yaml", ".yml":
		data, err = yaml.Marshal(policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy YAML: %w", err)
		}
	default:
		// Default to YAML for backwards compatibility
		data, err = yaml.Marshal(policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy YAML: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	return nil
}
