# Migration Guide

This guide helps you migrate from standalone GCP emulators (without IAM) to the GCP Emulator Control Plane with IAM enforcement.

---

## Table of Contents

1. [Overview](#overview)
2. [Migration Strategies](#migration-strategies)
3. [Step-by-Step Migration](#step-by-step-migration)
4. [Compatibility Matrix](#compatibility-matrix)
5. [Common Migration Patterns](#common-migration-patterns)
6. [Rollback Plan](#rollback-plan)
7. [Validation Checklist](#validation-checklist)

---

## Overview

### What's Changing

**Before (Standalone Emulators):**
- No permission checks
- All operations succeed
- No principal identity required
- Single-service deployment

**After (Control Plane with IAM):**
- Optional permission checks (opt-in via `IAM_MODE`)
- Operations gated by IAM policy
- Principal identity required (when IAM enabled)
- Multi-service orchestration

### Backward Compatibility Promise

+ **IAM_MODE=off is the default** - Existing code works without changes
+ **Opt-in activation** - You choose when to enable IAM
+ **Non-breaking** - All existing emulator features continue to work

### Migration Philosophy

**Gradual, not big-bang:**
1. Start with control plane in IAM_MODE=off (identical to before)
2. Add policy.yaml with permissive rules
3. Switch to IAM_MODE=permissive (fail-open)
4. Fix permission issues
5. Switch to IAM_MODE=strict (fail-closed) in CI

---

## Migration Strategies

### Strategy 1: Keep IAM Disabled (No Migration)

**When to use:**
- Quick local development
- Prototyping
- Not testing authorization logic

**Steps:**
1. Use control plane docker-compose
2. Keep default `IAM_MODE=off` in environment
3. No policy file needed
4. No code changes required

**Result:** Identical behavior to standalone emulators

---

### Strategy 2: Permissive Mode Migration (Recommended)

**When to use:**
- Testing authorization in development
- Gradual rollout
- Want to catch permission issues without blocking work

**Timeline:** 1-2 days

**Steps:**

#### Day 1 Morning: Deploy Control Plane
```bash
# Clone control plane
git clone https://github.com/blackwell-systems/gcp-emulator-control-plane.git
cd gcp-emulator-control-plane

# Start with IAM disabled
docker compose up -d

# Verify existing tests pass
cd /your/app
go test ./...
```

#### Day 1 Afternoon: Create Initial Policy
```yaml
# policy.yaml - Start permissive
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.*  # Wildcard grants all permissions
      - cloudkms.*

projects:
  your-project:
    bindings:
      - role: roles/custom.developer
        members:
          - allAuthenticatedUsers  # Allow anyone
```

#### Day 2 Morning: Enable Permissive Mode
```yaml
# docker-compose.yml
secret-manager:
  environment:
    - IAM_MODE=permissive  # Changed from 'off'
    - IAM_HOST=iam:8080

kms:
  environment:
    - IAM_MODE=permissive
    - IAM_HOST=iam:8080
```

```bash
docker compose restart secret-manager kms
```

#### Day 2 Afternoon: Add Principal Injection
```go
// In your test code
import "google.golang.org/grpc/metadata"

ctx = metadata.AppendToOutgoingContext(ctx, 
    "x-emulator-principal", 
    "user:test@example.com")
```

```bash
# In curl scripts
curl -H "X-Emulator-Principal: user:test@example.com" ...
```

Run tests again. They should still pass (permissive mode fails open).

---

### Strategy 3: Strict Mode Migration (For CI)

**When to use:**
- CI/CD pipelines
- Pre-production validation
- Want to catch permission bugs early

**Prerequisites:**
- Completed Strategy 2 (permissive mode working)
- All tests passing with principal injection
- Policy refined to real requirements

**Steps:**

#### 1. Refine Policy to Real Requirements
```yaml
# policy.yaml - Realistic permissions
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.versions.add
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt

  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers

      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
```

#### 2. Test in Permissive Mode
```bash
# Update policy
docker compose restart iam

# Run tests - should still pass
go test ./...
```

#### 3. Switch to Strict Mode
```yaml
# docker-compose.yml
secret-manager:
  environment:
    - IAM_MODE=strict  # Fail-closed

kms:
  environment:
    - IAM_MODE=strict
```

#### 4. Fix Permission Denials
```bash
docker compose restart secret-manager kms
go test ./...  # May fail now

# Check IAM logs for denials
docker compose logs iam | grep DENY

# Fix:
# - Add missing permissions to roles
# - Update principal in test code
# - Fix resource names
```

#### 5. Iterate Until Green
Repeat step 4 until all tests pass with `IAM_MODE=strict`.

---

## Step-by-Step Migration

### Phase 1: Assessment (30 minutes)

**Goal:** Understand current usage

1. **Inventory emulator usage**
   ```bash
   # Find all places emulators are used
   grep -r "localhost:9090" .
   grep -r "secretmanager" .
   grep -r "cloudkms" .
   ```

2. **Identify test scenarios**
   - Which tests use Secret Manager?
   - Which tests use KMS?
   - What principals do tests represent?

3. **Map operations to permissions**
   ```
   CreateSecret       → secretmanager.secrets.create
   AccessSecretVersion → secretmanager.versions.access
   Encrypt            → cloudkms.cryptoKeys.encrypt
   ```

### Phase 2: Deploy Control Plane (15 minutes)

1. **Clone repo**
   ```bash
   git clone https://github.com/blackwell-systems/gcp-emulator-control-plane.git
   cd gcp-emulator-control-plane
   ```

2. **Start stack (IAM disabled)**
   ```bash
   docker compose up -d
   docker compose ps  # Verify all healthy
   ```

3. **Update test configuration**
   ```bash
   # If tests hardcode ports, update them
   localhost:9090 → localhost:9090  # Secret Manager (unchanged)
   localhost:9091 → localhost:9091  # KMS (if using both)
   ```

4. **Verify tests pass**
   ```bash
   go test ./...
   ```

### Phase 3: Create Initial Policy (30 minutes)

1. **Start with permissive policy**
   ```yaml
   roles:
     roles/custom.allAccess:
       permissions:
         - secretmanager.*
         - cloudkms.*

   projects:
     test-project:
       bindings:
         - role: roles/custom.allAccess
           members:
             - allAuthenticatedUsers
   ```

2. **Copy to control plane**
   ```bash
   cp policy-permissive.yaml gcp-emulator-control-plane/policy.yaml
   ```

3. **Restart IAM emulator**
   ```bash
   cd gcp-emulator-control-plane
   docker compose restart iam
   ```

### Phase 4: Enable IAM (15 minutes)

1. **Switch to permissive mode**
   ```yaml
   # docker-compose.yml
   secret-manager:
     environment:
       - IAM_MODE=permissive
   ```

2. **Restart data plane**
   ```bash
   docker compose restart secret-manager kms
   ```

3. **Run tests**
   ```bash
   cd /your/app
   go test ./...
   ```

Tests should pass (permissive policy + permissive mode = all allowed).

### Phase 5: Add Principal Injection (1-2 hours)

**For Go SDK tests:**
```go
import "google.golang.org/grpc/metadata"

// Add to test setup
ctx = metadata.AppendToOutgoingContext(ctx, 
    "x-emulator-principal", 
    "user:test@example.com")

// Use ctx for all operations
client.CreateSecret(ctx, req)
```

**For REST/curl tests:**
```bash
curl -H "X-Emulator-Principal: user:test@example.com" \
  http://localhost:8081/v1/projects/test/secrets
```

**For test helpers:**
```go
// test/helpers.go
func NewTestContext(t *testing.T, principal string) context.Context {
    ctx := context.Background()
    ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)
    return ctx
}

// test/example_test.go
ctx := NewTestContext(t, "user:alice@example.com")
```

### Phase 6: Refine Policy (1-2 hours)

1. **Replace wildcards with specific permissions**
   ```yaml
   roles:
     roles/custom.developer:
       permissions:
         - secretmanager.secrets.create
         - secretmanager.secrets.get
         - secretmanager.versions.add
         - secretmanager.versions.access
         # No wildcard
   ```

2. **Add realistic principals**
   ```yaml
   groups:
     developers:
       members:
         - user:alice@example.com
         - user:bob@example.com

   projects:
     test-project:
       bindings:
         - role: roles/custom.developer
           members:
             - group:developers
   ```

3. **Test incrementally**
   ```bash
   # After each policy change
   docker compose restart iam
   go test ./...
   ```

### Phase 7: Strict Mode (30 minutes)

1. **Switch to strict mode**
   ```yaml
   # docker-compose.yml
   secret-manager:
     environment:
       - IAM_MODE=strict
   ```

2. **Fix failures**
   ```bash
   docker compose restart secret-manager kms
   go test ./... -v  # See which tests fail

   # Check logs
   docker compose logs iam | grep DENY
   ```

3. **Common fixes:**
   - Missing principal in test
   - Missing permission in role
   - Wrong resource name format
   - Principal not in any group

---

## Compatibility Matrix

### Emulator Versions

| Emulator | Minimum Version | IAM Support |
|----------|----------------|-------------|
| Secret Manager | v1.2.0+ | Full |
| KMS | v0.2.0+ | Full |
| Secret Manager | v1.1.0 | None (pre-IAM) |
| KMS | v0.1.0 | None (pre-IAM) |

### Client SDK Compatibility

| SDK | Compatible | Notes |
|-----|-----------|-------|
| Go SDK (cloud.google.com/go) | Yes | Requires metadata injection |
| Python SDK (google-cloud-*) | Yes | Requires metadata injection |
| REST/curl | Yes | Use X-Emulator-Principal header |
| Terraform | Partial | No header injection support |

### IAM Mode Compatibility

| Environment | Recommended Mode | Rationale |
|-------------|------------------|-----------|
| Local dev | `off` or `permissive` | Don't block development |
| Integration tests | `permissive` | Catch issues but don't block |
| CI/CD | `strict` | Enforce correct permissions |
| Production | N/A | Not for production use |

---

## Common Migration Patterns

### Pattern 1: Gradual Test Migration

**Scenario:** Large test suite, can't migrate all at once

**Solution:** Use IAM_MODE=off, migrate tests incrementally

```go
// Migrated test (with principal)
func TestCreateSecret_WithIAM(t *testing.T) {
    ctx := NewTestContextWithPrincipal(t, "user:alice@example.com")
    // Test logic
}

// Legacy test (no principal)
func TestCreateSecret_Legacy(t *testing.T) {
    ctx := context.Background()
    // Works because IAM_MODE=off
}
```

Switch to IAM_MODE=permissive once all tests have principals.

### Pattern 2: Multi-Environment Config

**Scenario:** Different IAM modes for different environments

**Solution:** Environment-specific docker-compose files

```yaml
# docker-compose.dev.yml
services:
  secret-manager:
    environment:
      - IAM_MODE=off

# docker-compose.ci.yml
services:
  secret-manager:
    environment:
      - IAM_MODE=strict
```

```bash
# Development
docker compose -f docker-compose.yml -f docker-compose.dev.yml up

# CI
docker compose -f docker-compose.yml -f docker-compose.ci.yml up
```

### Pattern 3: Service-Specific Migration

**Scenario:** Migrate Secret Manager first, KMS later

**Solution:** Different IAM modes per service

```yaml
secret-manager:
  environment:
    - IAM_MODE=permissive  # Migrated

kms:
  environment:
    - IAM_MODE=off  # Not migrated yet
```

### Pattern 4: Principal Per Test Fixture

**Scenario:** Different tests need different principals

**Solution:** Parameterized test context

```go
type TestFixture struct {
    Client    *secretmanager.Client
    Principal string
}

func NewFixture(t *testing.T, principal string) *TestFixture {
    ctx := context.Background()
    ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)
    
    conn, _ := grpc.Dial("localhost:9090", ...)
    client, _ := secretmanager.NewClient(ctx, option.WithGRPCConn(conn))
    
    return &TestFixture{
        Client:    client,
        Principal: principal,
    }
}

func TestAsAlice(t *testing.T) {
    f := NewFixture(t, "user:alice@example.com")
    // Test as Alice
}

func TestAsBob(t *testing.T) {
    f := NewFixture(t, "user:bob@example.com")
    // Test as Bob
}
```

---

## Rollback Plan

If migration causes issues, you can roll back safely:

### Quick Rollback (30 seconds)

**Switch back to standalone emulators:**
```bash
# Stop control plane
cd gcp-emulator-control-plane
docker compose down

# Start standalone Secret Manager
docker run -p 9090:9090 ghcr.io/blackwell-systems/gcp-secret-manager-emulator:v1.1.0

# Run tests
go test ./...
```

### Partial Rollback (1 minute)

**Keep control plane, disable IAM:**
```yaml
# docker-compose.yml
secret-manager:
  environment:
    - IAM_MODE=off  # Back to legacy
```

```bash
docker compose restart secret-manager
```

### Rollback Checklist

- [ ] Stop control plane containers
- [ ] Start standalone emulators on original ports
- [ ] Verify tests pass
- [ ] Remove principal injection code (optional)
- [ ] Document rollback reason
- [ ] Plan retry timeline

---

## Validation Checklist

### Pre-Migration Validation

- [ ] All tests passing with standalone emulators
- [ ] Emulator connection details documented
- [ ] Test suite execution time measured (baseline)
- [ ] Permission requirements documented

### Post-Deployment Validation (IAM_MODE=off)

- [ ] Control plane started successfully
- [ ] All services healthy (`docker compose ps`)
- [ ] All tests pass (identical to before)
- [ ] Test execution time similar to baseline

### Post-IAM-Enable Validation (IAM_MODE=permissive)

- [ ] IAM emulator logs show policy loaded
- [ ] Principal injection added to tests
- [ ] All tests pass with principal headers
- [ ] Permission denials logged but not enforced

### Post-Strict-Mode Validation (IAM_MODE=strict)

- [ ] All tests pass (no permission denials)
- [ ] Permission denials cause test failures
- [ ] Unauthorized principals correctly denied
- [ ] IAM logs show all permission checks

### CI Integration Validation

- [ ] CI pipeline uses control plane
- [ ] IAM_MODE=strict in CI environment
- [ ] Tests fail on permission issues
- [ ] Logs accessible for debugging

---

## Troubleshooting Migration Issues

### Issue: Tests Fail After Enabling IAM

**Symptoms:** Tests pass with IAM_MODE=off, fail with IAM_MODE=permissive

**Causes:**
1. Missing principal in test
2. Wrong permission in policy
3. Wrong resource name format

**Debug:**
```bash
# Check IAM logs
docker compose logs iam | tail -50

# Look for:
# - "No principal" warnings
# - "Permission denied" with details
# - Resource name mismatches
```

**Fix:**
```go
// Add principal
ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", "user:test@example.com")

// Or update policy
permissions:
  - secretmanager.versions.access  # Add missing permission
```

### Issue: "IAM check failed: connection refused"

**Symptoms:** Tests fail in strict mode with connection error

**Causes:**
1. IAM emulator not started
2. IAM_HOST wrong address
3. Networking issue

**Fix:**
```bash
# Verify IAM running
docker compose ps iam

# Check health
curl http://localhost:8080/health

# Check IAM_HOST
docker compose logs secret-manager | grep IAM_HOST

# Restart if needed
docker compose restart iam secret-manager
```

### Issue: "Permission denied" even with wildcard permission

**Symptoms:** Policy has `secretmanager.*` but still denied

**Causes:**
1. Wildcard not supported (use specific permissions)
2. Principal not in binding
3. Resource name mismatch

**Fix:**
```yaml
# Replace wildcard
permissions:
  - secretmanager.secrets.create
  - secretmanager.secrets.get
  # ... list all needed

# Verify principal in binding
bindings:
  - role: roles/custom.developer
    members:
      - user:test@example.com  # Must match exactly
```

---

## Migration Timeline Estimates

### Small Project (1-10 tests)
- Assessment: 30 minutes
- Deploy control plane: 15 minutes
- Create policy: 30 minutes
- Add principal injection: 30 minutes
- Switch to strict mode: 15 minutes
- **Total: 2 hours**

### Medium Project (10-100 tests)
- Assessment: 1 hour
- Deploy control plane: 15 minutes
- Create policy: 1 hour
- Add principal injection: 2-4 hours
- Refine policy: 1-2 hours
- Switch to strict mode: 1 hour
- **Total: 1-2 days**

### Large Project (100+ tests)
- Assessment: 2-4 hours
- Deploy control plane: 30 minutes
- Create policy: 2-4 hours
- Add principal injection: 1-2 days
- Refine policy: 4-8 hours
- Switch to strict mode: 2-4 hours
- **Total: 1-2 weeks**

---

## Success Criteria

Migration is complete when:

+ Control plane deployed and stable
+ All tests pass with IAM_MODE=strict
+ Policy reflects real permission requirements
+ Principal injection in all test code
+ CI uses strict mode
+ Documentation updated
+ Team trained on new workflow

---

## Getting Help

If you encounter issues during migration:

1. Check [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review [End-to-End Tutorial](END_TO_END_TUTORIAL.md)
3. Check IAM logs: `docker compose logs iam`
4. Open GitHub issue with:
   - Migration strategy used
   - IAM mode
   - Error logs
   - Policy file (redacted)
