# Policy Reference

Complete reference for policy file syntax, semantics, and best practices.

---

## Table of Contents

1. [Policy File Format](#policy-file-format)
2. [Policy Structure](#policy-structure)
3. [Roles](#roles)
4. [Groups](#groups)
5. [Projects and Bindings](#projects-and-bindings)
6. [Conditions](#conditions)
7. [Permission Format](#permission-format)
8. [Principal Format](#principal-format)
9. [Policy Validation](#policy-validation)
10. [Policy Packs](#policy-packs)
11. [Examples](#examples)
12. [Best Practices](#best-practices)

---

## Policy File Format

The control plane supports both YAML and JSON policy formats, detected automatically by file extension.

### YAML Format (`.yaml`, `.yml`)

**Recommended for:** Hand-written policies, local development, readability

```yaml
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get

groups:
  developers:
    members:
      - user:alice@example.com

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
```

**Advantages:**
- Human-readable and editable
- Supports comments
- Less verbose
- Better for version control diffs

### JSON Format (`.json`)

**Recommended for:** Production policy testing, GCP policy exports, automation

```json
{
  "roles": {
    "roles/custom.developer": {
      "permissions": [
        "secretmanager.secrets.create",
        "secretmanager.secrets.get"
      ]
    }
  },
  "groups": {
    "developers": {
      "members": [
        "user:alice@example.com"
      ]
    }
  },
  "projects": {
    "test-project": {
      "bindings": [
        {
          "role": "roles/custom.developer",
          "members": [
            "group:developers"
          ]
        }
      ]
    }
  }
}
```

**Advantages:**
- Matches GCP's native IAM policy format
- Can import actual production policies via `gcloud`
- Machine-readable
- Works with JSON tooling (jq, etc.)

### Exporting Production Policies

Test with actual GCP policies:

```bash
# Export production IAM policy
gcloud projects get-iam-policy my-prod-project --format=json > prod-policy.json

# Use with emulator
gcp-emulator start --policy-file=prod-policy.json
```

---

## Policy Structure

A policy file has three top-level sections:

```yaml
roles:      # Custom role definitions (permission sets)
groups:     # Group membership (principal collections)
projects:   # Resource hierarchy with IAM bindings
```

All three sections are optional but at least one must be present.

---

## Roles

Roles are named collections of permissions.

### Built-in Roles

GCP built-in roles are supported (resolved by IAM emulator):

```yaml
projects:
  test-project:
    bindings:
      - role: roles/owner
        members:
          - user:admin@example.com
```

Common built-in roles:
- `roles/owner` - Full access
- `roles/editor` - Read/write access
- `roles/viewer` - Read-only access
- Service-specific roles: `roles/secretmanager.secretAccessor`, `roles/cloudkms.cryptoKeyEncrypter`

### Custom Roles

Define your own roles with specific permissions:

```yaml
roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.encrypt
```

**Naming convention:**
- Must start with `roles/`
- Custom roles typically use `roles/custom.*` prefix
- Use descriptive names: `roles/custom.developer`, `roles/custom.ciRunner`

### Permission Sets

Group related permissions into roles:

```yaml
roles:
  # Developer role - full secret management
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.secrets.list
      - secretmanager.secrets.update
      - secretmanager.secrets.delete
      - secretmanager.versions.add
      - secretmanager.versions.access
      - secretmanager.versions.list

  # CI role - limited secret access
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

  # Admin role - everything
  roles/custom.admin:
    permissions:
      # Secret Manager
      - secretmanager.secrets.*    # Not yet supported - must list explicitly
      # KMS
      - cloudkms.keyRings.create
      - cloudkms.cryptoKeys.create
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt
```

**Note:** Wildcard permissions (`*`) are not currently supported. List permissions explicitly.

---

## Groups

Groups are reusable collections of principals.

### Basic Groups

```yaml
groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com
      - user:charlie@example.com
```

### Nested Groups

Groups can contain other groups (one level):

```yaml
groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

  operations:
    members:
      - user:ops@example.com

  admins:
    members:
      - user:admin@example.com
      - group:developers    # Admins include all developers
      - group:operations    # Admins include all operations
```

**Nesting rules:**
- One level of nesting supported
- Two levels not supported: `group:admins` → `group:developers` → `group:juniors` ❌
- Circular references not checked (avoid them)

### Service Account Groups

```yaml
groups:
  ci-accounts:
    members:
      - serviceAccount:ci@test-project.iam.gserviceaccount.com
      - serviceAccount:github-actions@test-project.iam.gserviceaccount.com
```

---

## Projects and Bindings

Projects define the resource hierarchy and IAM bindings.

### Basic Project Binding

```yaml
projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
```

### Multiple Bindings

```yaml
projects:
  test-project:
    bindings:
      # Admins have full access
      - role: roles/custom.admin
        members:
          - user:admin@example.com
          - group:admins

      # Developers have read/write access
      - role: roles/custom.developer
        members:
          - group:developers

      # CI has limited access
      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
```

### Multiple Projects

```yaml
projects:
  # Production project - restricted access
  prod-project:
    bindings:
      - role: roles/custom.admin
        members:
          - group:admins

  # Staging project - developer access
  staging-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
          - serviceAccount:ci@test-project.iam.gserviceaccount.com

  # Development project - open access
  dev-project:
    bindings:
      - role: roles/owner
        members:
          - group:developers
```

### Bindings with Conditions

Restrict access based on resource attributes:

```yaml
projects:
  test-project:
    bindings:
      # CI can only access production secrets
      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "CI limited to production secrets"

      # Developers can access non-production secrets
      - role: roles/custom.developer
        members:
          - group:developers
        condition:
          expression: '!resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "Developers excluded from production secrets"
```

---

## Conditions

Conditions use Common Expression Language (CEL) to restrict access based on context.

### Condition Structure

```yaml
condition:
  expression: 'CEL expression'     # Required
  title: "Human-readable title"    # Optional but recommended
  description: "Detailed explanation"  # Optional
```

### Available Variables

**`resource.name`** - Full resource path being accessed

Examples:
- `projects/test-project/secrets/db-password`
- `projects/test-project/secrets/db-password/versions/1`
- `projects/test-project/locations/global/keyRings/app/cryptoKeys/data`

**`request.time`** - Timestamp of request (future)

### CEL String Operators

**`startsWith(prefix)`** - Check if resource name starts with prefix

```yaml
expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
```

**`endsWith(suffix)`** - Check if resource name ends with suffix

```yaml
expression: 'resource.name.endsWith("/versions/latest")'
```

**`contains(substring)`** - Check if resource name contains substring

```yaml
expression: 'resource.name.contains("/secrets/dev-")'
```

**`matches(regex)`** - Match resource name against regex

```yaml
expression: 'resource.name.matches("projects/[^/]+/secrets/(dev|test)-.*")'
```

### CEL Logical Operators

**`&&`** - AND operator

```yaml
expression: 'resource.name.startsWith("projects/test-project/") && resource.name.contains("/secrets/prod-")'
```

**`||`** - OR operator

```yaml
expression: 'resource.name.contains("/dev-") || resource.name.contains("/test-")'
```

**`!`** - NOT operator

```yaml
expression: '!resource.name.contains("/prod-")'
```

### CEL Comparison Operators

**`==`, `!=`** - Equality

```yaml
expression: 'resource.name == "projects/test-project/secrets/allowed-secret"'
```

**`<`, `>`, `<=`, `>=`** - Comparison (for numbers, timestamps)

```yaml
# Future: time-based access
expression: 'request.time < timestamp("2025-12-31T23:59:59Z")'
```

### Condition Examples

**Restrict by resource name prefix:**

```yaml
condition:
  expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
  title: "Production secrets only"
```

**Restrict by environment:**

```yaml
condition:
  expression: 'resource.name.contains("/dev-") || resource.name.contains("/test-")'
  title: "Dev and test environments only"
```

**Exclude specific resources:**

```yaml
condition:
  expression: '!resource.name.endsWith("/secrets/admin-key")'
  title: "All secrets except admin-key"
```

**Complex multi-condition:**

```yaml
condition:
  expression: >
    resource.name.startsWith("projects/test-project/") &&
    (resource.name.contains("/secrets/app-") || resource.name.contains("/secrets/service-")) &&
    !resource.name.contains("/secrets/app-admin")
  title: "App and service secrets, excluding admin"
  description: "CI can access application and service secrets but not administrative secrets"
```

**Regex-based pattern matching:**

```yaml
condition:
  expression: 'resource.name.matches("projects/test-project/secrets/(dev|staging)-.*")'
  title: "Dev and staging secrets matching pattern"
```

---

## Permission Format

Permissions follow GCP's standard format: `service.resource.verb`

### Secret Manager Permissions

**Secret-level:**
- `secretmanager.secrets.create` - Create new secrets
- `secretmanager.secrets.get` - Get secret metadata
- `secretmanager.secrets.list` - List secrets in project
- `secretmanager.secrets.update` - Update secret metadata
- `secretmanager.secrets.delete` - Delete secrets

**Version-level:**
- `secretmanager.versions.add` - Add new secret versions
- `secretmanager.versions.access` - Access secret version payload
- `secretmanager.versions.list` - List secret versions
- `secretmanager.versions.enable` - Enable disabled versions
- `secretmanager.versions.disable` - Disable versions
- `secretmanager.versions.destroy` - Permanently destroy versions

### KMS Permissions

**KeyRing-level:**
- `cloudkms.keyRings.create` - Create key rings
- `cloudkms.keyRings.get` - Get key ring metadata
- `cloudkms.keyRings.list` - List key rings

**CryptoKey-level:**
- `cloudkms.cryptoKeys.create` - Create crypto keys
- `cloudkms.cryptoKeys.get` - Get crypto key metadata
- `cloudkms.cryptoKeys.list` - List crypto keys
- `cloudkms.cryptoKeys.update` - Update crypto key metadata
- `cloudkms.cryptoKeys.encrypt` - Encrypt data
- `cloudkms.cryptoKeys.decrypt` - Decrypt data

**CryptoKeyVersion-level:**
- `cloudkms.cryptoKeyVersions.create` - Create key versions
- `cloudkms.cryptoKeyVersions.get` - Get key version metadata
- `cloudkms.cryptoKeyVersions.list` - List key versions
- `cloudkms.cryptoKeyVersions.update` - Update key version state
- `cloudkms.cryptoKeyVersions.destroy` - Destroy key versions

### Permission Validation

Valid permission format: `service.resource.verb`

**Valid:**
- ✓ `secretmanager.secrets.get`
- ✓ `cloudkms.cryptoKeys.encrypt`
- ✓ `cloudkms.cryptoKeyVersions.useToDecrypt`

**Invalid:**
- ✗ `secretmanager.get` (too short - missing resource)
- ✗ `secretmanager` (only service)
- ✗ `secretmanager.*` (wildcards not supported)
- ✗ `secrets.get` (missing service prefix)

---

## Principal Format

Principals identify who is requesting access.

### User Principals

**Format:** `user:email@domain.com`

```yaml
members:
  - user:alice@example.com
  - user:bob@example.com
```

### Service Account Principals

**Format:** `serviceAccount:name@project.iam.gserviceaccount.com`

```yaml
members:
  - serviceAccount:ci@test-project.iam.gserviceaccount.com
  - serviceAccount:app@prod-project.iam.gserviceaccount.com
```

### Group Principals

**Format:** `group:groupname`

```yaml
members:
  - group:developers
  - group:admins
```

**Note:** Group names reference groups defined in the `groups:` section, not GCP Workspace groups.

### Principal Injection

Principals are injected via headers:

**HTTP:**
```bash
curl -H "X-Emulator-Principal: user:alice@example.com" ...
```

**gRPC:**
```go
ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", "user:alice@example.com")
```

---

## Policy Validation

Validate policy files before using them:

```bash
# Validate default policy.yaml
gcp-emulator policy validate

# Validate specific file
gcp-emulator policy validate path/to/policy.json

# Validate and show detailed output
gcp-emulator policy validate policy.yaml --verbose
```

### Validation Checks

The validator checks:

1. **Role names** - Must start with `roles/`
2. **Permission format** - Must be `service.resource.verb`
3. **Role references** - Custom roles must be defined in `roles:` section
4. **Group references** - Groups must be defined in `groups:` section
5. **Principal format** - Must match `user:*`, `serviceAccount:*`, or `group:*`
6. **Condition syntax** - CEL expressions must be valid
7. **YAML/JSON syntax** - File must be parseable

### Validation Output

**Valid policy:**
```
✓ Policy is valid

2 roles defined
3 groups defined
1 projects configured
```

**Invalid policy:**
```
✗ Validation failed

Errors:
  Role roles/custom.developer references undefined permission: secretmanager.bad.permission
  Binding in project test-project references undefined role: roles/custom.nonexistent
  Principal format invalid: alice@example.com (should be user:alice@example.com)
```

---

## Policy Packs

The `packs/` directory contains ready-to-use role definitions for common services.

### Available Packs

**`packs/secretmanager.yaml`** - Secret Manager roles

```yaml
roles:
  roles/custom.secretManager.admin:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.delete
      - secretmanager.versions.add
      - secretmanager.versions.access

  roles/custom.secretManager.accessor:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access
```

**`packs/kms.yaml`** - KMS roles

```yaml
roles:
  roles/custom.kms.encrypter:
    permissions:
      - cloudkms.cryptoKeys.encrypt

  roles/custom.kms.decrypter:
    permissions:
      - cloudkms.cryptoKeys.decrypt

  roles/custom.kms.admin:
    permissions:
      - cloudkms.keyRings.create
      - cloudkms.cryptoKeys.create
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt
```

**`packs/ci.yaml`** - CI/CD patterns

```yaml
roles:
  roles/custom.ci.secrets:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

  roles/custom.ci.deploy:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.decrypt
```

### Using Policy Packs

**Copy/paste approach:**

```bash
# Copy role definitions from pack
cat packs/secretmanager.yaml >> policy.yaml

# Edit to customize
vim policy.yaml
```

**Manual merge:**

Open `packs/secretmanager.yaml` and copy desired roles into your `policy.yaml` under the `roles:` section.

---

## Examples

### Example 1: Basic Development Setup

```yaml
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.versions.add
      - secretmanager.versions.access

groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

projects:
  dev-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
```

### Example 2: Multi-Environment with Conditions

```yaml
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.versions.add
      - secretmanager.versions.access

  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

groups:
  developers:
    members:
      - user:alice@example.com

projects:
  test-project:
    bindings:
      # Developers can access non-prod secrets
      - role: roles/custom.developer
        members:
          - group:developers
        condition:
          expression: '!resource.name.contains("/prod-")'
          title: "Non-production secrets only"

      # CI can access prod secrets
      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.contains("/prod-")'
          title: "Production secrets only"
```

### Example 3: KMS Encryption Pipeline

```yaml
roles:
  roles/custom.kms.encrypter:
    permissions:
      - cloudkms.cryptoKeys.encrypt

  roles/custom.kms.decrypter:
    permissions:
      - cloudkms.cryptoKeys.decrypt

  roles/custom.secretWriter:
    permissions:
      - secretmanager.versions.add
      - cloudkms.cryptoKeys.encrypt  # Can encrypt before storing

  roles/custom.secretReader:
    permissions:
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.decrypt  # Can decrypt after retrieving

groups:
  writers:
    members:
      - serviceAccount:writer@test-project.iam.gserviceaccount.com

  readers:
    members:
      - serviceAccount:reader@test-project.iam.gserviceaccount.com

projects:
  test-project:
    bindings:
      - role: roles/custom.secretWriter
        members:
          - group:writers

      - role: roles/custom.secretReader
        members:
          - group:readers
```

### Example 4: Admin with Full Access

```yaml
roles:
  roles/custom.admin:
    permissions:
      # Secret Manager - all permissions
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.secrets.list
      - secretmanager.secrets.update
      - secretmanager.secrets.delete
      - secretmanager.versions.add
      - secretmanager.versions.access
      - secretmanager.versions.list
      - secretmanager.versions.enable
      - secretmanager.versions.disable
      - secretmanager.versions.destroy
      # KMS - all permissions
      - cloudkms.keyRings.create
      - cloudkms.keyRings.get
      - cloudkms.keyRings.list
      - cloudkms.cryptoKeys.create
      - cloudkms.cryptoKeys.get
      - cloudkms.cryptoKeys.list
      - cloudkms.cryptoKeys.update
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt
      - cloudkms.cryptoKeyVersions.create
      - cloudkms.cryptoKeyVersions.destroy

groups:
  admins:
    members:
      - user:admin@example.com

projects:
  prod-project:
    bindings:
      - role: roles/custom.admin
        members:
          - group:admins
```

---

## Best Practices

### 1. Use Groups for Maintainability

**Bad:**
```yaml
projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - user:alice@example.com
          - user:bob@example.com
          - user:charlie@example.com
```

**Good:**
```yaml
groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com
      - user:charlie@example.com

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
```

### 2. Principle of Least Privilege

Grant only the permissions needed for each role:

```yaml
# Bad: Overly broad
roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.create  # CI doesn't need create
      - secretmanager.secrets.delete  # CI doesn't need delete
      - secretmanager.versions.access # CI only needs this

# Good: Minimal permissions
roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get     # Read metadata
      - secretmanager.versions.access # Access payload
```

### 3. Use Conditions for Resource-Based Access

```yaml
# Bad: Separate roles per environment
roles:
  roles/custom.devSecrets:
    permissions: [...]
  roles/custom.prodSecrets:
    permissions: [...]

# Good: One role with conditions
roles:
  roles/custom.developer:
    permissions: [...]

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
        condition:
          expression: '!resource.name.contains("/prod-")'
```

### 4. Document Conditions with Titles

```yaml
# Bad: No title
condition:
  expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'

# Good: Clear title
condition:
  expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
  title: "CI limited to production secrets"
  description: "CI pipeline can only access secrets with 'prod-' prefix for deployment"
```

### 5. Organize Roles by Responsibility

```yaml
roles:
  # Read-only roles
  roles/custom.viewer:
    permissions: [...]

  # Read-write roles
  roles/custom.developer:
    permissions: [...]

  # Administrative roles
  roles/custom.admin:
    permissions: [...]
```

### 6. Version Control Your Policy

- Keep `policy.yaml` in version control
- Review policy changes in pull requests
- Use meaningful commit messages
- Tag policy versions for production

### 7. Test Policy Changes

```bash
# Validate before committing
gcp-emulator policy validate policy.yaml

# Test with strict mode in CI
gcp-emulator start --mode=strict
```

### 8. Use JSON for Production Parity

For testing with production policies:

```bash
# Export production policy
gcloud projects get-iam-policy prod-project --format=json > prod-policy.json

# Test locally
gcp-emulator start --policy-file=prod-policy.json
go test ./...
```

### 9. Keep Policies DRY with Packs

Don't duplicate role definitions. Use policy packs:

```bash
# Include ready-made roles
cat packs/secretmanager.yaml >> policy.yaml
cat packs/kms.yaml >> policy.yaml
```

### 10. Monitor Permission Denials

Check IAM logs for unexpected denials:

```bash
gcp-emulator logs iam | grep DENY
```

---

## Related Documentation

- [Architecture](ARCHITECTURE.md) - Control plane design and authorization model
- [Integration Contract](INTEGRATION_CONTRACT.md) - Emulator integration requirements
- [CI Integration](CI_INTEGRATION.md) - CI/CD usage patterns
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
