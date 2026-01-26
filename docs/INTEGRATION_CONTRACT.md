# Integration Contract

This document defines the stable contract that all emulators in the GCP Emulator Control Plane must implement.

## Overview

The integration contract ensures consistent authorization behavior across all emulators. Any emulator that implements this contract can join the control plane mesh.

---

## 1. Resource Naming

All emulators must use canonical GCP resource naming conventions.

### Secret Manager

```
projects/{project}/secrets/{secret}
projects/{project}/secrets/{secret}/versions/{version}
```

### KMS

```
projects/{project}/locations/{location}/keyRings/{keyring}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}/cryptoKeyVersions/{version}
```

### General Pattern

```
projects/{project}[/locations/{location}]/{collection}/{resource}[/{subcollection}/{subresource}]
```

**Requirements:**
- Resource names must be URL-safe
- Use lowercase for all path segments except variable values
- Support both short names and fully-qualified names

---

## 2. Permission Mapping

Each operation must map to a real GCP IAM permission.

### Format

```
{service}.{resource}.{verb}
```

### Examples

| Operation | Permission | Resource Target |
|-----------|-----------|----------------|
| CreateSecret | `secretmanager.secrets.create` | Parent project |
| AccessSecretVersion | `secretmanager.versions.access` | Secret version |
| CreateKeyRing | `cloudkms.keyRings.create` | Parent location |
| Encrypt | `cloudkms.cryptoKeys.encrypt` | Crypto key |

### Permission Check Timing

**Before operation execution:**
- Extract principal from request
- Normalize resource path
- Call IAM emulator with `TestIamPermissions`
- Return 403 Permission Denied if denied

**Resource target rules:**
- Create operations check against parent
- Read/Update/Delete operations check against target resource
- List operations check against parent

---

## 3. Principal Injection (Inbound)

Emulators must accept principal identity via standardized headers/metadata.

### gRPC

**Metadata key:** `x-emulator-principal`

```go
import "google.golang.org/grpc/metadata"

md, ok := metadata.FromIncomingContext(ctx)
if ok {
    principals := md.Get("x-emulator-principal")
    if len(principals) > 0 {
        principal = principals[0]
    }
}
```

### HTTP

**Header:** `X-Emulator-Principal`

```bash
curl -H "X-Emulator-Principal: user:alice@example.com" ...
```

### Supported Principal Formats

```
user:{email}
serviceAccount:{email}
group:{name}
allUsers
allAuthenticatedUsers
```

---

## 4. Principal Propagation (Outbound)

When calling the IAM emulator, propagate the principal via metadata (not request body).

### gRPC to IAM Emulator

```go
import (
    "google.golang.org/grpc/metadata"
    iampb "cloud.google.com/go/iam/apiv1/iampb"
)

// Extract principal from incoming request
principal := extractPrincipalFromContext(ctx)

// Inject into outgoing context for IAM call
ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)

// Call IAM emulator
resp, err := iamClient.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
    Resource:    resource,
    Permissions: []string{permission},
})
```

**Why metadata, not request body?**
- Keeps request format identical to real GCP
- Separates control plane (identity) from data plane (request)
- Enables transparent principal forwarding

---

## 5. IAM Mode Configuration

All emulators must support three IAM modes via environment variables.

### Environment Variables

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `IAM_MODE` | `off`, `permissive`, `strict` | `off` | Authorization mode |
| `IAM_HOST` | `host:port` | `localhost:8080` | IAM emulator address |

### Mode Behavior

**`off` (default):**
- No permission checks
- All requests succeed
- Legacy emulator behavior

**`permissive` (fail-open):**
- Check permissions
- If IAM unavailable → allow
- If IAM denies → deny
- Good for development

**`strict` (fail-closed):**
- Check permissions
- If IAM unavailable → deny
- If IAM denies → deny
- Good for CI/CD

### Implementation Pattern

```go
type Server struct {
    iamClient *iam.Client
    iamMode   AuthMode
}

func (s *Server) checkPermission(ctx context.Context, resource, permission string) error {
    if s.iamMode == AuthModeOff {
        return nil  // No checks
    }

    principal := extractPrincipalFromContext(ctx)
    
    allowed, err := s.iamClient.CheckPermission(ctx, principal, resource, permission)
    if err != nil {
        if isConnectivityError(err) {
            if s.iamMode == AuthModePermissive {
                return nil  // Fail-open
            }
            return status.Errorf(codes.Internal, "IAM unavailable")
        }
        return err
    }

    if !allowed {
        return status.Error(codes.PermissionDenied, "Permission denied")
    }

    return nil
}
```

---

## 6. Error Responses

### Permission Denied

**gRPC:** `codes.PermissionDenied`
**HTTP:** `403 Forbidden`

```json
{
  "error": {
    "code": 403,
    "message": "Permission denied",
    "status": "PERMISSION_DENIED"
  }
}
```

### IAM Unavailable (strict mode)

**gRPC:** `codes.Internal`
**HTTP:** `500 Internal Server Error`

```json
{
  "error": {
    "code": 500,
    "message": "IAM check failed: connection refused",
    "status": "INTERNAL"
  }
}
```

---

## 7. Shared Authentication Library

Use [`gcp-emulator-auth`](https://github.com/blackwell-systems/gcp-emulator-auth) for standardized IAM integration.

### Go Module

```bash
go get github.com/blackwell-systems/gcp-emulator-auth
```

### Usage

```go
import emulatorauth "github.com/blackwell-systems/gcp-emulator-auth"

// Load config from environment
config := emulatorauth.LoadFromEnv()

// Create IAM client
iamClient, err := emulatorauth.NewClient(config.Host, config.Mode)

// Check permission
allowed, err := iamClient.CheckPermission(ctx, principal, resource, permission)
```

**Benefits:**
- Consistent behavior across emulators
- Maintained error classification
- Standard principal extraction/injection
- No copy/paste drift

---

## 8. Testing Requirements

Emulators must include:

### Unit Tests
- Test with IAM_MODE=off (default behavior)
- Test resource normalization
- Test permission mapping correctness

### Integration Tests
- Test with IAM emulator running
- Test permissive vs strict mode behavior
- Test principal propagation
- Test permission denied scenarios

### Example Test Structure

```go
func TestIAMIntegration(t *testing.T) {
    tests := []struct {
        name         string
        iamMode      string
        principal    string
        expectError  bool
        expectedCode codes.Code
    }{
        {
            name:      "permissive mode - allow without principal",
            iamMode:   "permissive",
            principal: "",
            expectError: false,
        },
        {
            name:         "strict mode - deny without principal",
            iamMode:      "strict",
            principal:    "",
            expectError:  true,
            expectedCode: codes.PermissionDenied,
        },
    }
    // ...
}
```

---

## 9. Documentation Requirements

Emulators must document:

### README.md
- IAM integration feature
- Configuration variables
- Principal injection examples (gRPC + HTTP)
- Permission mapping table

### Integration Tests
- Example test file showing IAM integration
- Instructions for running with IAM emulator

### Permission Reference
- Complete operation → permission mapping
- Resource target rules (parent vs self)

---

## 10. Backward Compatibility

**Non-breaking requirement:**
- IAM_MODE=off must remain default
- Existing users see no behavior change
- Opt-in activation via environment variables

**Version compatibility:**
- IAM emulator API: v1 (stable)
- Principal injection: backward compatible
- Permission format: follows GCP naming

---

## Checklist for New Emulators

- [ ] Canonical resource naming
- [ ] Operation → permission mapping documented
- [ ] Principal extraction (gRPC + HTTP)
- [ ] Principal propagation to IAM emulator
- [ ] Three IAM modes supported (off/permissive/strict)
- [ ] Uses `gcp-emulator-auth` library
- [ ] Unit tests with IAM_MODE=off
- [ ] Integration tests with IAM emulator
- [ ] README with IAM section
- [ ] Permission reference table
- [ ] Docker Compose integration example
- [ ] Backward compatible (opt-in)

---

## Reference Implementations

- [gcp-secret-manager-emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator)
- [gcp-kms-emulator](https://github.com/blackwell-systems/gcp-kms-emulator)
