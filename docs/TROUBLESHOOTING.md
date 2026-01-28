# Troubleshooting Guide

Common issues and solutions for the GCP Emulator Control Plane.

---

## Table of Contents

1. [Startup Issues](#startup-issues)
2. [Permission Denied Errors](#permission-denied-errors)
3. [Connection Issues](#connection-issues)
4. [Policy Problems](#policy-problems)
5. [Principal Injection Issues](#principal-injection-issues)
6. [Performance Problems](#performance-problems)
7. [Debugging Tools](#debugging-tools)
8. [Common Error Messages](#common-error-messages)

---

## Startup Issues

### Issue: "Service unhealthy" in docker compose ps

**Symptoms:**
```bash
$ docker compose ps
NAME        STATUS
iam         unhealthy
```

**Causes:**
1. IAM emulator failed to start
2. Health check failing
3. Policy file invalid

**Diagnosis:**
```bash
# Check logs
docker compose logs iam

# Check if process running
docker compose exec iam ps aux

# Test health endpoint manually
curl http://localhost:8080/health
```

**Solutions:**

**Invalid policy file:**
```bash
# Check policy syntax
docker compose logs iam | grep -i error
docker compose logs iam | grep -i "failed to load"

# Common syntax errors:
# - Incorrect YAML indentation
# - Missing colons
# - Invalid role/permission names
```

**Fix:**
```yaml
# Ensure proper YAML structure
roles:    # Must have colon
  roles/custom.developer:    # Must have colon
    permissions:    # Must have colon
      - secretmanager.secrets.create    # Dash + space
```

**Port already in use:**
```bash
# Check what's using port 8080
lsof -i :8080
netstat -an | grep 8080

# Kill conflicting process or change port
# docker-compose.yml:
ports:
  - "8081:8080"  # Use different host port
```

---

### Issue: "Bind: address already in use"

**Symptoms:**
```
Error starting userland proxy: listen tcp4 0.0.0.0:9090: bind: address already in use
```

**Cause:** Another process using the port

**Solution:**
```bash
# Find process using port
lsof -i :9090

# Kill it
kill -9 <PID>

# Or change docker-compose.yml port mapping
ports:
  - "9092:9090"  # Use different host port
```

---

### Issue: "Failed to connect to IAM emulator"

**Symptoms:**
- Data plane services start but immediately fail
- Logs show "connection refused" to IAM emulator

**Cause:** IAM emulator not ready when data plane starts

**Solution:**
```yaml
# docker-compose.yml - Ensure proper depends_on
secret-manager:
  depends_on:
    iam:
      condition: service_healthy  # Wait for health check
```

```bash
# Restart with proper dependency
docker compose down
docker compose up -d
```

---

## Permission Denied Errors

### Issue: "Permission denied" with IAM_MODE=off

**Symptoms:**
- IAM_MODE=off but still getting 403 errors
- Tests that should succeed are failing

**Cause:** IAM mode not actually disabled

**Diagnosis:**
```bash
# Check environment variable
docker compose exec secret-manager env | grep IAM_MODE

# Check logs for IAM client initialization
docker compose logs secret-manager | grep IAM
```

**Solution:**
```yaml
# docker-compose.yml - Explicitly set to off
environment:
  - IAM_MODE=off

# Restart
docker compose restart secret-manager
```

---

### Issue: "Permission denied" despite having correct role

**Symptoms:**
- User has role in policy
- Role has permission
- Still getting 403

**Diagnosis:**
```bash
# Enable trace logging
# docker-compose.yml
iam:
  command: ["--config", "/policy.yaml", "--trace"]

# Restart and check logs
docker compose restart iam
docker compose logs -f iam

# Look for:
# - Principal extraction
# - Permission check details
# - Which bindings were evaluated
```

**Common Causes:**

**1. Resource name mismatch:**
```bash
# IAM log might show:
[TRACE] Checking permission for resource: projects/test-project/secrets/db-password
[TRACE] Policy has binding for: projects/prod-project/secrets/db-password
# ^ Mismatch!
```

**Fix:** Ensure request uses correct project name

**2. Principal format mismatch:**
```yaml
# Policy has:
members:
  - user:alice@example.com

# But request sends:
X-Emulator-Principal: alice@example.com  # Missing "user:" prefix
```

**Fix:**
```bash
# Always use format: type:identifier
curl -H "X-Emulator-Principal: user:alice@example.com" ...
```

**3. Group membership not working:**
```yaml
groups:
  developers:
    members:
      - user:alice@example.com

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers  # Binding to group
```

```bash
# Request must use user principal, not group
# CORRECT:
X-Emulator-Principal: user:alice@example.com

# WRONG:
X-Emulator-Principal: group:developers
```

**4. Condition not satisfied:**
```yaml
bindings:
  - role: roles/custom.developer
    members:
      - user:alice@example.com
    condition:
      expression: 'resource.name.startsWith("projects/test/secrets/prod-")'
```

```bash
# If accessing: projects/test/secrets/dev-password
# Condition evaluates to false → denied

# Fix: Access correct resource or update condition
```

---

### Issue: Tests pass locally, fail in CI

**Symptoms:**
- IAM_MODE=permissive locally (passes)
- IAM_MODE=strict in CI (fails)

**Cause:** Strict mode catches permission issues that permissive mode allows

**Solution:**
```bash
# Run locally with strict mode
export IAM_MODE=strict
docker compose up -d
go test ./...

# Fix permission issues before pushing
```

---

## Connection Issues

### Issue: "connection refused" to emulator

**Symptoms:**
```
dial tcp 127.0.0.1:9090: connect: connection refused
```

**Diagnosis:**
```bash
# Check if service running
docker compose ps

# Check if port exposed
docker compose port secret-manager 9090

# Test connectivity
curl http://localhost:8081/health  # HTTP endpoint
```

**Solutions:**

**Service not running:**
```bash
docker compose up -d secret-manager
```

**Wrong port:**
```go
// Check connection string
conn, err := grpc.Dial(
    "localhost:9090",  // Verify correct port
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

**Docker network issue:**
```bash
# Recreate network
docker compose down
docker network prune
docker compose up -d
```

---

### Issue: "IAM check failed: connection refused"

**Symptoms:**
- Data plane emulator can't reach IAM emulator
- Strict mode: all requests denied
- Permissive mode: all requests allowed

**Diagnosis:**
```bash
# Check IAM_HOST setting
docker compose exec secret-manager env | grep IAM_HOST

# Test connectivity from data plane to IAM
docker compose exec secret-manager curl http://iam:8080/health
```

**Solutions:**

**Wrong IAM_HOST:**
```yaml
# docker-compose.yml
environment:
  - IAM_HOST=iam:8080  # Use container name, not localhost
```

**IAM not in same network:**
```yaml
services:
  iam:
    networks:
      - emulator-net
  secret-manager:
    networks:
      - emulator-net

networks:
  emulator-net:
```

---

## Policy Problems

### Issue: "Failed to load policy: parse error"

**Symptoms:**
- IAM emulator fails to start
- Error message about YAML syntax

**Common Syntax Errors:**

**1. Indentation:**
```yaml
# WRONG:
roles:
roles/custom.developer:  # Missing indent
  permissions:

# CORRECT:
roles:
  roles/custom.developer:  # Indented 2 spaces
    permissions:
```

**2. Missing colons:**
```yaml
# WRONG:
roles
  roles/custom.developer

# CORRECT:
roles:
  roles/custom.developer:
```

**3. List syntax:**
```yaml
# WRONG:
permissions:
secretmanager.secrets.create  # Missing dash

# CORRECT:
permissions:
  - secretmanager.secrets.create  # Dash + space
```

**Validation:**
```bash
# Use YAML linter
yamllint policy.yaml

# Or online: https://www.yamllint.com/
```

---

### Issue: Wildcard permissions not working

**Symptoms:**
- Policy has `secretmanager.*`
- Still getting permission denied

**Cause:** Wildcards not supported in current IAM emulator

**Solution:**
```yaml
# Replace wildcards with explicit permissions
roles:
  roles/custom.developer:
    permissions:
      # WRONG:
      # - secretmanager.*
      
      # CORRECT:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.secrets.update
      - secretmanager.secrets.delete
      - secretmanager.secrets.list
      - secretmanager.versions.add
      - secretmanager.versions.get
      - secretmanager.versions.access
      - secretmanager.versions.list
      - secretmanager.versions.enable
      - secretmanager.versions.disable
      - secretmanager.versions.destroy
```

---

### Issue: Policy changes not taking effect

**Symptoms:**
- Updated policy.yaml
- Still using old policy behavior

**Cause:** Policy loaded at startup only

**Solution:**
```bash
# Restart IAM emulator to reload policy
docker compose restart iam

# Verify new policy loaded
docker compose logs iam | grep "Loaded policy"
```

---

## Principal Injection Issues

### Issue: "No principal in context"

**Symptoms:**
- Strict mode: All requests denied
- Logs show "no principal" or "empty principal"

**Cause:** Principal header not sent or not extracted

**Diagnosis:**

**For gRPC:**
```go
// Check if metadata is set
md, ok := metadata.FromOutgoingContext(ctx)
if ok {
    fmt.Println("Metadata:", md)  // Should show x-emulator-principal
}
```

**For HTTP:**
```bash
# Verify header sent
curl -v -H "X-Emulator-Principal: user:alice@example.com" http://localhost:8081/...
# Look for header in request output
```

**Solutions:**

**gRPC - Missing metadata:**
```go
// WRONG:
ctx := context.Background()
client.CreateSecret(ctx, req)  // No principal

// CORRECT:
ctx := context.Background()
ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", "user:alice@example.com")
client.CreateSecret(ctx, req)
```

**HTTP - Missing header:**
```bash
# WRONG:
curl http://localhost:8081/v1/projects/test/secrets

# CORRECT:
curl -H "X-Emulator-Principal: user:alice@example.com" \
  http://localhost:8081/v1/projects/test/secrets
```

---

### Issue: Principal format rejected

**Symptoms:**
- Principal sent but still denied
- Logs show "invalid principal format"

**Cause:** Wrong principal format

**Correct Formats:**
```
user:alice@example.com
serviceAccount:sa@project.iam.gserviceaccount.com
group:developers
allUsers
allAuthenticatedUsers
```

**Wrong Formats:**
```
alice@example.com               # Missing type prefix
user:alice                      # Missing domain
users:alice@example.com         # "users" should be "user"
group:user:alice@example.com    # Multiple prefixes
```

---

## Performance Problems

### Issue: Slow request processing

**Symptoms:**
- Requests take multiple seconds
- Noticeable latency compared to standalone emulators

**Expected Performance:**
```
Without IAM: ~1-2ms per operation
With IAM:    ~5-10ms per operation (acceptable for testing)
```

**If slower than 100ms:**

**Diagnosis:**
```bash
# Check IAM emulator resource usage
docker stats iam

# Check if IAM logs show delays
docker compose logs iam | grep -i "slow\|timeout"

# Check network latency
docker compose exec secret-manager ping iam
```

**Solutions:**

**CPU throttling:**
```bash
# Check system resources
top
docker stats

# Close unnecessary applications
```

**DNS resolution issues:**
```yaml
# docker-compose.yml - Use IP instead of hostname
environment:
  - IAM_HOST=172.18.0.2:8080  # Use container IP
```

**Large policy file:**
```yaml
# Simplify policy during development
# Keep minimal roles/bindings for tests
# Use specific projects, not many wildcards
```

---

### Issue: High memory usage

**Symptoms:**
- Docker containers using excessive RAM
- System slow/swapping

**Expected Memory:**
```
IAM:            ~50MB
Secret Manager: ~30MB
KMS:            ~30MB
Total:          ~110MB
```

**If using 500MB+:**

**Diagnosis:**
```bash
docker stats --no-stream
```

**Solutions:**

**Memory leak (rare):**
```bash
# Restart containers
docker compose restart
```

**Too many resources stored:**
```bash
# Emulators use in-memory storage
# Creating 10000+ secrets will consume RAM

# Solution: Clean up or restart
docker compose restart secret-manager
```

---

## Debugging Tools

### Enable Trace Logging

**IAM Emulator:**
```yaml
# docker-compose.yml
iam:
  command: ["--config", "/policy.yaml", "--trace"]
```

```bash
docker compose restart iam
docker compose logs -f iam
```

**Output:**
```
[TRACE] CheckPermission: principal=user:alice@example.com resource=projects/test/secrets/db permission=secretmanager.secrets.get result=ALLOW
[TRACE] Binding matched: role=roles/custom.developer member=group:developers
[TRACE] Group expansion: developers -> [user:alice@example.com, user:bob@example.com]
```

---

### Interactive Shell in Container

```bash
# Access container
docker compose exec secret-manager sh

# Check environment
env | grep IAM

# Test IAM connectivity
curl http://iam:8080/health

# Check process
ps aux

# Exit
exit
```

---

### Network Inspection

```bash
# List networks
docker network ls

# Inspect control plane network
docker network inspect gcp-iam-control-plane_default

# See which containers are connected
docker network inspect gcp-iam-control-plane_default | jq '.[0].Containers'
```

---

### Watch Logs in Real-Time

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f iam

# Filter by keyword
docker compose logs -f | grep -i "denied\|error"

# Last 100 lines
docker compose logs --tail=100 iam
```

---

### Test IAM Directly

```bash
# Test policy loading
curl http://localhost:8080/health

# Check if IAM responds (requires gRPC client)
grpcurl -plaintext localhost:8080 list

# Or use Go test
cat > test_iam.go <<'EOF'
package main

import (
    "context"
    "fmt"
    iampb "cloud.google.com/go/iam/apiv1/iampb"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
)

func main() {
    conn, _ := grpc.NewClient("localhost:8080",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    defer conn.Close()

    client := iampb.NewIAMPolicyClient(conn)
    ctx := metadata.AppendToOutgoingContext(context.Background(),
        "x-emulator-principal", "user:alice@example.com")

    resp, err := client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
        Resource:    "projects/test-project",
        Permissions: []string{"secretmanager.secrets.create"},
    })
    
    fmt.Printf("Granted: %v, Error: %v\n", resp.GetPermissions(), err)
}
EOF

go run test_iam.go
```

---

## Common Error Messages

### "code = PermissionDenied desc = Permission denied"

**Meaning:** IAM policy denied the operation

**Check:**
1. Principal in policy.yaml
2. Role has required permission
3. Binding assigns role to principal
4. Resource name matches
5. Condition (if any) is satisfied

**Debug:**
```bash
docker compose logs iam --tail=50 | grep -A5 -B5 "DENY"
```

---

### "code = Internal desc = IAM check failed"

**Meaning:** Data plane couldn't reach IAM emulator (strict mode)

**Check:**
1. IAM emulator running: `docker compose ps iam`
2. IAM_HOST correct: `docker compose exec secret-manager env | grep IAM_HOST`
3. Network connectivity: `docker compose exec secret-manager curl http://iam:8080/health`

---

### "code = InvalidArgument desc = parent is required"

**Meaning:** Request validation failed (before IAM check)

**Check:**
- Request format
- Required fields present
- Resource naming correct

**Example:**
```bash
# WRONG:
curl -X POST http://localhost:8081/v1/projects//secrets

# CORRECT:
curl -X POST http://localhost:8081/v1/projects/test-project/secrets
```

---

### "Failed to load policy: unknown field"

**Meaning:** policy.yaml has invalid field name

**Common Mistakes:**
```yaml
# WRONG:
permissions:
  - secret.create  # Missing "secretmanager" prefix

# CORRECT:
permissions:
  - secretmanager.secrets.create

# WRONG:
bindings:
  - roles: roles/custom.developer  # "roles" should be "role"

# CORRECT:
bindings:
  - role: roles/custom.developer
```

---

## When to Ask for Help

If you've tried troubleshooting and still have issues:

1. **Gather information:**
   - Docker compose logs: `docker compose logs > logs.txt`
   - Policy file (redacted if needed)
   - Error message
   - Steps to reproduce

2. **Check documentation:**
   - [End-to-End Tutorial](END_TO_END_TUTORIAL.md)
   - [Migration Guide](MIGRATION.md)
   - [Integration Contract](INTEGRATION_CONTRACT.md)

3. **Search existing issues:**
   - [Control Plane Issues](https://github.com/blackwell-systems/gcp-iam-control-plane/issues)
   - [IAM Emulator Issues](https://github.com/blackwell-systems/gcp-iam-emulator/issues)
   - [Secret Manager Issues](https://github.com/blackwell-systems/gcp-secret-manager-emulator/issues)

4. **Open new issue:**
   - Use issue template
   - Include logs and policy (redacted)
   - Describe expected vs actual behavior
   - Include steps to reproduce

---

## Quick Fixes Summary

| Problem | Quick Fix |
|---------|-----------|
| Service won't start | `docker compose down && docker compose up -d` |
| Permission denied | Check IAM logs: `docker compose logs iam | grep DENY` |
| No principal | Add header: `X-Emulator-Principal: user:test@example.com` |
| IAM unavailable | Restart: `docker compose restart iam` |
| Policy not loading | Check syntax: `yamllint policy.yaml` |
| Port conflict | Change port in docker-compose.yml |
| Slow performance | Check resources: `docker stats` |
| Policy changes ignored | Restart IAM: `docker compose restart iam` |
