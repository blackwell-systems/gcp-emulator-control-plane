package policy

import (
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
