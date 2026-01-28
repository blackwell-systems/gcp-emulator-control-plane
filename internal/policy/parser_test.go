package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePermissionFormat(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		wantErr    bool
	}{
		{
			name:       "valid permission",
			permission: "secretmanager.secrets.get",
			wantErr:    false,
		},
		{
			name:       "valid kms permission",
			permission: "cloudkms.cryptoKeys.encrypt",
			wantErr:    false,
		},
		{
			name:       "invalid - too short",
			permission: "secretmanager.get",
			wantErr:    true,
		},
		{
			name:       "invalid - only service",
			permission: "secretmanager",
			wantErr:    true,
		},
		{
			name:       "invalid - wildcard not allowed",
			permission: "secretmanager.*",
			wantErr:    true,
		},
		{
			name:       "valid long permission",
			permission: "cloudkms.cryptoKeyVersions.useToDecrypt",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePermission(tt.permission)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePermission(%q) error = %v, wantErr %v", tt.permission, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name     string
		roleName string
		wantErr  bool
	}{
		{
			name:     "valid custom role",
			roleName: "roles/custom.developer",
			wantErr:  false,
		},
		{
			name:     "valid built-in role",
			roleName: "roles/owner",
			wantErr:  false,
		},
		{
			name:     "invalid - missing prefix",
			roleName: "custom.developer",
			wantErr:  true,
		},
		{
			name:     "invalid - empty",
			roleName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasPrefix := len(tt.roleName) > 0 && tt.roleName[:6] == "roles/"
			if hasPrefix == tt.wantErr {
				t.Errorf("role %q validation incorrect, hasPrefix=%v, wantErr=%v", tt.roleName, hasPrefix, tt.wantErr)
			}
		})
	}
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	if !result.Valid {
		t.Error("New ValidationResult should be valid initially")
	}

	result.addError("test error")

	if result.Valid {
		t.Error("ValidationResult should be invalid after adding error")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0] != "test error" {
		t.Errorf("Expected 'test error', got %q", result.Errors[0])
	}
}

func TestLoadYAML(t *testing.T) {
	// Load YAML test fixture
	policy, err := Load("../../testdata/policy.yaml")
	if err != nil {
		t.Fatalf("Failed to load YAML policy: %v", err)
	}

	// Verify structure
	if policy == nil {
		t.Fatal("Policy is nil")
	}

	// Check roles
	if len(policy.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(policy.Roles))
	}

	ciRunner, ok := policy.Roles["roles/custom.ciRunner"]
	if !ok {
		t.Error("Missing roles/custom.ciRunner")
	}
	if len(ciRunner.Permissions) != 3 {
		t.Errorf("Expected 3 permissions for ciRunner, got %d", len(ciRunner.Permissions))
	}

	// Check groups
	if len(policy.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(policy.Groups))
	}

	developers, ok := policy.Groups["developers"]
	if !ok {
		t.Error("Missing developers group")
	}
	if len(developers.Members) != 2 {
		t.Errorf("Expected 2 members in developers, got %d", len(developers.Members))
	}

	// Check projects
	if len(policy.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(policy.Projects))
	}

	testProject, ok := policy.Projects["test-project"]
	if !ok {
		t.Error("Missing test-project")
	}
	if len(testProject.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(testProject.Bindings))
	}

	// Check condition
	if testProject.Bindings[1].Condition == nil {
		t.Error("Expected condition on second binding")
	} else {
		if testProject.Bindings[1].Condition.Title != "CI limited to production secrets" {
			t.Errorf("Unexpected condition title: %s", testProject.Bindings[1].Condition.Title)
		}
	}
}

func TestLoadJSON(t *testing.T) {
	// Load JSON test fixture
	policy, err := Load("../../testdata/policy.json")
	if err != nil {
		t.Fatalf("Failed to load JSON policy: %v", err)
	}

	// Verify structure (should be identical to YAML)
	if policy == nil {
		t.Fatal("Policy is nil")
	}

	// Check roles
	if len(policy.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(policy.Roles))
	}

	ciRunner, ok := policy.Roles["roles/custom.ciRunner"]
	if !ok {
		t.Error("Missing roles/custom.ciRunner")
	}
	if len(ciRunner.Permissions) != 3 {
		t.Errorf("Expected 3 permissions for ciRunner, got %d", len(ciRunner.Permissions))
	}

	// Check groups
	if len(policy.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(policy.Groups))
	}

	developers, ok := policy.Groups["developers"]
	if !ok {
		t.Error("Missing developers group")
	}
	if len(developers.Members) != 2 {
		t.Errorf("Expected 2 members in developers, got %d", len(developers.Members))
	}

	// Check projects
	if len(policy.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(policy.Projects))
	}

	testProject, ok := policy.Projects["test-project"]
	if !ok {
		t.Error("Missing test-project")
	}
	if len(testProject.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(testProject.Bindings))
	}

	// Check condition
	if testProject.Bindings[1].Condition == nil {
		t.Error("Expected condition on second binding")
	} else {
		if testProject.Bindings[1].Condition.Title != "CI limited to production secrets" {
			t.Errorf("Unexpected condition title: %s", testProject.Bindings[1].Condition.Title)
		}
	}
}

func TestSaveYAML(t *testing.T) {
	// Create test policy
	policy := &Policy{
		Roles: map[string]Role{
			"roles/custom.test": {
				Permissions: []string{"secretmanager.secrets.get"},
			},
		},
		Groups: map[string]Group{
			"testers": {
				Members: []string{"user:test@example.com"},
			},
		},
		Projects: map[string]Project{
			"test-project": {
				Bindings: []Binding{
					{
						Role:    "roles/custom.test",
						Members: []string{"group:testers"},
					},
				},
			},
		},
	}

	// Save to temp file
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.yaml")

	err := Save(policy, outPath)
	if err != nil {
		t.Fatalf("Failed to save YAML: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Load it back
	loaded, err := Load(outPath)
	if err != nil {
		t.Fatalf("Failed to load saved YAML: %v", err)
	}

	// Verify content
	if len(loaded.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(loaded.Roles))
	}
}

func TestSaveJSON(t *testing.T) {
	// Create test policy
	policy := &Policy{
		Roles: map[string]Role{
			"roles/custom.test": {
				Permissions: []string{"secretmanager.secrets.get"},
			},
		},
		Groups: map[string]Group{
			"testers": {
				Members: []string{"user:test@example.com"},
			},
		},
		Projects: map[string]Project{
			"test-project": {
				Bindings: []Binding{
					{
						Role:    "roles/custom.test",
						Members: []string{"group:testers"},
					},
				},
			},
		},
	}

	// Save to temp file
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.json")

	err := Save(policy, outPath)
	if err != nil {
		t.Fatalf("Failed to save JSON: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Load it back
	loaded, err := Load(outPath)
	if err != nil {
		t.Fatalf("Failed to load saved JSON: %v", err)
	}

	// Verify content
	if len(loaded.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(loaded.Roles))
	}
}

func TestLoadUnknownExtension(t *testing.T) {
	// Create temp file with .txt extension
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "policy.txt")

	// Write YAML content (should fallback to YAML parsing)
	content := []byte(`roles:
  roles/custom.test:
    permissions:
      - secretmanager.secrets.get
groups: {}
projects: {}
`)
	err := os.WriteFile(txtPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should succeed with YAML fallback
	policy, err := Load(txtPath)
	if err != nil {
		t.Fatalf("Failed to load with unknown extension: %v", err)
	}

	if len(policy.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(policy.Roles))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	content := []byte(`{invalid json}`)
	err := os.WriteFile(jsonPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should fail
	_, err = Load(jsonPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	content := []byte(`roles:
  - this is not
  - a map
`)
	err := os.WriteFile(yamlPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should fail
	_, err = Load(yamlPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
