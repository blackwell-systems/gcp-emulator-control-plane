# Architecture

This document describes the system design, component interactions, and technical architecture of the GCP Emulator Control Plane.

---

## Table of Contents

1. [High-Level Architecture](#high-level-architecture)
2. [Component Responsibilities](#component-responsibilities)
3. [Request Flow](#request-flow)
4. [Identity Propagation](#identity-propagation)
5. [Authorization Model](#authorization-model)
6. [Failure Modes](#failure-modes)
7. [Network Topology](#network-topology)
8. [Data Flow](#data-flow)
9. [Design Decisions](#design-decisions)
10. [Extension Points](#extension-points)

---

## High-Level Architecture

The GCP Emulator Control Plane uses a **control plane + data plane** architecture modeled after real GCP:

```
┌─────────────────────────────────────────────────────────┐
│                    Control Plane                        │
│  ┌──────────────────────────────────────────────────┐   │
│  │            IAM Emulator (policy.yaml)            │   │
│  │  - Policy evaluation                             │   │
│  │  - Role expansion                                │   │
│  │  - Condition evaluation (CEL)                    │   │
│  │  - Group membership resolution                   │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                         ▲
                         │ CheckPermission(principal, resource, permission)
                         │
┌────────────────────────┴────────────────────────────────┐
│                     Data Plane                          │
│  ┌─────────────────────┐    ┌─────────────────────┐    │
│  │ Secret Manager      │    │  KMS Emulator       │    │
│  │ Emulator            │    │                     │    │
│  │ - CRUD operations   │    │  - Key management   │    │
│  │ - Permission checks │    │  - Encrypt/Decrypt  │    │
│  │ - Resource storage  │    │  - Permission checks│    │
│  └─────────────────────┘    └─────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                         ▲
                         │ X-Emulator-Principal header
                         │
                   ┌─────┴─────┐
                   │   Client  │
                   └───────────┘
```

### Key Characteristics

- **Separation of concerns**: Control plane handles authorization, data plane handles operations
- **Centralized policy**: Single `policy.yaml` drives all authorization decisions
- **Stateless data plane**: Emulators don't store policy, they delegate to IAM emulator
- **Consistent identity channel**: Principal injection works the same across all services

---

## Component Responsibilities

### IAM Emulator (Control Plane)

**Purpose:** Enforce authorization policy across all data plane services

**Responsibilities:**
- Load and parse `policy.yaml`
- Evaluate `TestIamPermissions` requests
- Expand roles into permission sets
- Resolve group memberships
- Evaluate CEL condition expressions
- Return allow/deny decisions

**Does NOT:**
- Store or manage data plane resources
- Execute business logic
- Handle data plane CRUD operations
- Maintain resource state

**Interfaces:**
- gRPC API: `google.iam.v1.IAMPolicy` service
- HTTP endpoint: `/health` for readiness checks
- Config: `policy.yaml` file mount

### Secret Manager Emulator (Data Plane)

**Purpose:** Emulate GCP Secret Manager CRUD operations with IAM enforcement

**Responsibilities:**
- Store secrets and versions in-memory
- Validate request parameters
- Check permissions via IAM emulator (when IAM_MODE enabled)
- Execute secret operations (create, read, update, delete)
- Return appropriate error codes

**Does NOT:**
- Store or evaluate IAM policy
- Make authorization decisions (delegates to IAM emulator)
- Persist data across restarts

**Interfaces:**
- gRPC API: `google.cloud.secretmanager.v1.SecretManagerService`
- HTTP API: REST gateway at `/v1/projects/{project}/secrets`
- IAM integration: Calls IAM emulator's `TestIamPermissions`

### KMS Emulator (Data Plane)

**Purpose:** Emulate GCP KMS cryptographic operations with IAM enforcement

**Responsibilities:**
- Manage key rings and crypto keys in-memory
- Perform encrypt/decrypt operations
- Check permissions via IAM emulator (when IAM_MODE enabled)
- Execute KMS operations (create keys, encrypt, decrypt)
- Return appropriate error codes

**Does NOT:**
- Store or evaluate IAM policy
- Use real cryptographic algorithms (uses simple XOR for testing)
- Persist keys across restarts

**Interfaces:**
- gRPC API: `google.cloud.kms.v1.KeyManagementService`
- HTTP API: REST gateway at `/v1/projects/{project}/locations/{location}/keyRings`
- IAM integration: Calls IAM emulator's `TestIamPermissions`

---

## Request Flow

### Standard Operation Flow

```
1. Client                     2. Data Plane               3. Control Plane
   │                             │                           │
   ├─ POST /secrets              │                           │
   │  X-Emulator-Principal:      │                           │
   │  user:alice@example.com     │                           │
   │                             │                           │
   │────────────────────────────>│                           │
   │                             │                           │
   │                             ├─ Extract principal        │
   │                             ├─ Validate request         │
   │                             ├─ Normalize resource       │
   │                             │                           │
   │                             ├─ TestIamPermissions       │
   │                             │  principal=alice          │
   │                             │  resource=projects/test   │
   │                             │  permission=secrets.create│
   │                             │                           │
   │                             │──────────────────────────>│
   │                             │                           │
   │                             │                           ├─ Load policy
   │                             │                           ├─ Expand roles
   │                             │                           ├─ Check bindings
   │                             │                           ├─ Evaluate conditions
   │                             │                           │
   │                             │<──────────────────────────┤
   │                             │  permissions=[            │
   │                             │    "secrets.create"       │
   │                             │  ]                        │
   │                             │                           │
   │                             ├─ Execute operation        │
   │                             ├─ Store resource           │
   │                             │                           │
   │<────────────────────────────┤                           │
   │  200 OK                     │                           │
   │  {secret}                   │                           │
```

### Permission Denied Flow

```
1. Client                     2. Data Plane               3. Control Plane
   │                             │                           │
   ├─ GET /secrets/prod-key      │                           │
   │  X-Emulator-Principal:      │                           │
   │  user:charlie@example.com   │                           │
   │                             │                           │
   │────────────────────────────>│                           │
   │                             │                           │
   │                             ├─ Extract principal        │
   │                             ├─ Validate request         │
   │                             │                           │
   │                             ├─ TestIamPermissions       │
   │                             │  principal=charlie        │
   │                             │  resource=projects/.../prod-key
   │                             │  permission=secrets.get   │
   │                             │                           │
   │                             │──────────────────────────>│
   │                             │                           │
   │                             │                           ├─ Check bindings
   │                             │                           ├─ No match found
   │                             │                           │
   │                             │<──────────────────────────┤
   │                             │  permissions=[]           │
   │                             │                           │
   │                             ├─ STOP (denied)            │
   │                             │                           │
   │<────────────────────────────┤                           │
   │  403 Forbidden              │                           │
   │  Permission denied          │                           │
```

---

## Identity Propagation

### Inbound: Client → Data Plane

Principal identity flows from client to data plane emulator via **standard headers**:

**gRPC:**
```
Metadata: x-emulator-principal: user:alice@example.com
```

**HTTP:**
```
Header: X-Emulator-Principal: user:alice@example.com
```

### Extraction Pattern

Data plane emulators extract the principal using `gcp-emulator-auth` library:

```go
import emulatorauth "github.com/blackwell-systems/gcp-emulator-auth"

principal := emulatorauth.ExtractPrincipalFromContext(ctx)
// Returns: "user:alice@example.com"
```

### Outbound: Data Plane → Control Plane

When calling IAM emulator, the data plane **propagates** the principal via metadata:

```go
// Inject principal into outgoing context
ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)

// Call IAM emulator
resp, err := iamClient.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
    Resource:    "projects/test-project/secrets/db-password",
    Permissions: []string{"secretmanager.secrets.get"},
})
```

### Why Metadata, Not Request Body?

**Design rationale:**
1. **Matches real GCP behavior**: GCP API requests don't include identity in the body
2. **Separation of concerns**: Identity is control plane concern, request is data plane
3. **Transparent forwarding**: Data plane can forward identity without parsing it
4. **Test realism**: Tests using real GCP SDK clients work without modification

---

## Authorization Model

### Policy Structure

```yaml
roles:                        # Role definitions
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get

groups:                       # Group membership
  developers:
    members:
      - user:alice@example.com

projects:                     # Resource hierarchy
  test-project:
    bindings:                 # IAM bindings
      - role: roles/custom.developer
        members:
          - group:developers
        condition:            # Optional CEL expression
          expression: 'resource.name.startsWith("projects/test-project/secrets/dev-")'
```

### Permission Check Algorithm

```
1. Extract principal from request
   → "user:alice@example.com"

2. Normalize resource path
   → "projects/test-project/secrets/db-password"

3. Call IAM emulator TestIamPermissions
   → CheckPermission(principal, resource, permission)

4. IAM emulator evaluates:
   a. Find all bindings for resource (project-level)
   b. Expand groups → alice is in "developers"
   c. Check if any binding grants required permission
   d. Evaluate conditions (if present)
   e. Return list of granted permissions

5. Data plane checks result:
   - If permission in result → ALLOW
   - If permission not in result → DENY (403)
```

### Condition Evaluation

Conditions use **Common Expression Language (CEL)**:

```yaml
condition:
  expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
  title: "Restrict to production secrets"
```

**Evaluation context:**
- `resource.name`: Full resource path being accessed
- `request.time`: Timestamp of request
- Custom variables (future)

**CEL operators:**
- String: `startsWith()`, `endsWith()`, `contains()`, `matches()`
- Logical: `&&`, `||`, `!`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`

---

## Failure Modes

### IAM Mode Behavior Matrix

| Scenario | IAM_MODE=off | IAM_MODE=permissive | IAM_MODE=strict |
|----------|--------------|---------------------|-----------------|
| IAM emulator healthy | No check (allow) | Check permission | Check permission |
| IAM emulator down | No check (allow) | Allow (fail-open) | Deny (fail-closed) |
| IAM returns error | No check (allow) | Allow (fail-open) | Deny (fail-closed) |
| No principal header | No check (allow) | Deny | Deny |
| Permission denied | No check (allow) | Deny | Deny |

### Error Propagation

```
Data Plane Error → Client Error Code

Permission denied     → 403 Forbidden (PermissionDenied)
IAM unavailable       → 500 Internal (Internal) [strict mode only]
Invalid request       → 400 Bad Request (InvalidArgument)
Resource not found    → 404 Not Found (NotFound)
```

### Recovery Strategies

**IAM emulator restart:**
- Data plane maintains connection pool
- Automatic reconnection on next request
- No data loss (policy in config file)

**Data plane restart:**
- In-memory data lost (emulator design)
- IAM state preserved (stateless data plane)
- Clients retry with standard backoff

**Network partition:**
- Permissive mode: Operations continue (fail-open)
- Strict mode: Operations blocked (fail-closed)
- Health checks detect partition

---

## Network Topology

### Docker Compose Deployment

```
┌─────────────────────────────────────────────────┐
│             Docker Network (bridge)             │
│                                                 │
│  ┌──────────────┐                               │
│  │ IAM Emulator │                               │
│  │ iam:8080     │◄──────────────┐               │
│  └──────────────┘               │               │
│         ▲                       │               │
│         │ CheckPermission       │               │
│         │                       │               │
│  ┌──────┴───────────┐    ┌──────┴──────────┐   │
│  │ Secret Manager   │    │  KMS Emulator   │   │
│  │ secret-mgr:9090  │    │  kms:9090       │   │
│  │ secret-mgr:8080  │    │  kms:8080       │   │
│  └──────────────────┘    └─────────────────┘   │
│         ▲                       ▲               │
│         │                       │               │
└─────────┼───────────────────────┼───────────────┘
          │                       │
          │   gRPC/HTTP           │
          │                       │
     ┌────┴───────────────────────┴────┐
     │        Host Machine             │
     │  localhost:9090 (Secret Mgr)    │
     │  localhost:8081 (Secret Mgr)    │
     │  localhost:9091 (KMS)           │
     │  localhost:8082 (KMS)           │
     │  localhost:8080 (IAM)           │
     └─────────────────────────────────┘
```

### Service Discovery

- **Within Docker network**: Services use container names (`iam:8080`, `secret-mgr:9090`)
- **From host**: Services use `localhost` with mapped ports
- **Health checks**: Docker uses `curl http://localhost:8080/health`

### Port Allocation Strategy

```
IAM Emulator:
  8080  - gRPC (standard)

Secret Manager:
  9090  - gRPC (data plane standard)
  8081  - HTTP (avoid conflict with IAM 8080)

KMS:
  9091  - gRPC (avoid conflict with Secret Manager)
  8082  - HTTP (avoid conflict with others)
```

---

## Data Flow

### Secret Creation with Encryption

**Scenario:** Create secret with KMS-encrypted value

```
Client
  │
  ├─ 1. Encrypt plaintext with KMS
  │    POST /v1/projects/test/locations/global/keyRings/app/cryptoKeys/data:encrypt
  │    X-Emulator-Principal: user:alice@example.com
  │    Body: {"plaintext": "c2VjcmV0"}
  │
  ▼
KMS Emulator
  │
  ├─ 2. Check permission: cloudkms.cryptoKeys.encrypt
  │    → TestIamPermissions(alice, projects/test/.../data, encrypt)
  │
  ▼
IAM Emulator
  │
  ├─ 3. Evaluate policy
  │    ✓ alice in developers group
  │    ✓ developers have encrypt permission
  │    → Return: ["cloudkms.cryptoKeys.encrypt"]
  │
  ▼
KMS Emulator
  │
  ├─ 4. Execute encryption
  │    → Return: {"ciphertext": "ZW5jcnlwdGVk"}
  │
  ▼
Client
  │
  ├─ 5. Store ciphertext in Secret Manager
  │    POST /v1/projects/test/secrets/db-password:addVersion
  │    X-Emulator-Principal: user:alice@example.com
  │    Body: {"payload": {"data": "ZW5jcnlwdGVk"}}
  │
  ▼
Secret Manager Emulator
  │
  ├─ 6. Check permission: secretmanager.versions.add
  │    → TestIamPermissions(alice, projects/test/secrets/db-password, versions.add)
  │
  ▼
IAM Emulator
  │
  ├─ 7. Evaluate policy
  │    ✓ alice has secretmanager.versions.add
  │    → Return: ["secretmanager.versions.add"]
  │
  ▼
Secret Manager Emulator
  │
  ├─ 8. Store version
  │    → Return: {version: "1"}
  │
  ▼
Client
```

**Permission checks:** 2 total (1 KMS encrypt, 1 Secret Manager add version)

---

## Design Decisions

### 1. Stateless Data Plane

**Decision:** Data plane emulators delegate all authorization to IAM emulator

**Rationale:**
- Single source of truth for policy
- No policy synchronization needed
- Easy to update policy without restarting data plane
- Matches real GCP architecture

**Trade-off:** Extra network hop for permission checks (acceptable for emulator)

### 2. Opt-In IAM Integration

**Decision:** IAM_MODE defaults to `off` (legacy behavior)

**Rationale:**
- Non-breaking for existing users
- Gradual migration path
- Clear opt-in signals intent to use IAM
- Flexibility for different environments

**Trade-off:** Users must explicitly enable IAM (acceptable, documented)

### 3. Metadata-Based Identity

**Decision:** Principal propagated via gRPC metadata, not request body

**Rationale:**
- Matches real GCP (identity in auth layer, not request)
- Enables SDK compatibility (no request modification)
- Separation of concerns (control vs data plane)
- Transparent forwarding

**Trade-off:** Extra header to remember (mitigated by library)

### 4. Fail-Open vs Fail-Closed Modes

**Decision:** Two modes (`permissive` and `strict`) for different use cases

**Rationale:**
- Development needs fail-open (don't block on IAM issues)
- CI needs fail-closed (catch permission bugs)
- Explicit choice forces consideration

**Trade-off:** More complexity (acceptable, well-documented)

### 5. In-Memory Storage

**Decision:** Emulators use in-memory storage, no persistence

**Rationale:**
- Simpler implementation
- Fast startup/teardown
- Hermetic testing (clean state per run)
- Emulator scope (not production replacement)

**Trade-off:** Data lost on restart (acceptable for testing)

---

## Extension Points

### Adding New Emulators

To add a new emulator to the control plane:

1. **Implement integration contract** (see [INTEGRATION_CONTRACT.md](INTEGRATION_CONTRACT.md))
2. **Add to docker-compose.yml**:
   ```yaml
   new-service:
     image: ghcr.io/blackwell-systems/gcp-new-service-emulator:latest
     ports:
       - "9092:9090"
     environment:
       - IAM_MODE=permissive
       - IAM_HOST=iam:8080
     depends_on:
       iam:
         condition: service_healthy
   ```
3. **Add permissions to policy packs** (`packs/new-service.yaml`)
4. **Update documentation** (README, tutorial)

### Custom Policy Sources

**Current:** `policy.yaml` file mount

**Future extension points:**
- Git repository sync
- Secret Manager policy storage
- Dynamic policy reload
- Policy inheritance/layering

### Advanced CEL Conditions

**Current:** Basic CEL expressions on `resource.name`

**Future capabilities:**
- Time-based conditions (`request.time`)
- Tag-based conditions (`resource.tags`)
- Custom attributes
- External data sources

### Observability Integration

**Current:** Docker logs

**Future extension points:**
- OpenTelemetry traces
- Prometheus metrics
- Audit log export
- Permission check visualization

---

## Performance Characteristics

### Latency Budget

```
Operation without IAM:  ~1ms   (in-memory)
Permission check:       ~5ms   (network + policy eval)
Total with IAM:         ~6ms   (acceptable for testing)
```

### Scalability

**Designed for:**
- Single developer workstation
- CI/CD test runners
- Integration test suites

**NOT designed for:**
- Production workloads
- High throughput (1000+ req/s)
- Large datasets (>10k resources)

### Resource Usage

```
IAM Emulator:         ~50MB RAM
Secret Manager:       ~30MB RAM
KMS:                  ~30MB RAM
Total:                ~110MB RAM (acceptable for Docker)
```

---

## Security Model

### Threat Model

**In scope:**
- Authorization logic correctness
- Permission denial enforcement
- Condition evaluation accuracy

**Out of scope:**
- Authentication (no real auth in emulator)
- Encryption strength (KMS uses weak crypto)
- Data persistence security (in-memory only)
- Network security (assumes trusted network)

### Design for Testing, Not Production

The control plane is explicitly **NOT production-ready**:
- No TLS/encryption
- No authentication
- No audit logging
- No rate limiting
- No data persistence

**Use case:** Testing IAM behavior in development/CI environments

---

## Related Documentation

- [Integration Contract](INTEGRATION_CONTRACT.md) - Technical contract for emulators
- [End-to-End Tutorial](END_TO_END_TUTORIAL.md) - Complete usage walkthrough
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
