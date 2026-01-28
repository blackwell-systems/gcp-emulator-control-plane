# CI Integration

Complete guide for integrating the GCP Emulator Control Plane into CI/CD pipelines.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [GitHub Actions](#github-actions)
3. [GitLab CI](#gitlab-ci)
4. [CircleCI](#circleci)
5. [Jenkins](#jenkins)
6. [Docker Compose in CI](#docker-compose-in-ci)
7. [IAM Modes for CI](#iam-modes-for-ci)
8. [Testing Strategies](#testing-strategies)
9. [Troubleshooting CI](#troubleshooting-ci)
10. [Best Practices](#best-practices)

---

## Quick Start

**Recommended approach:** Use the `gcp-emulator` CLI in strict mode.

```yaml
- name: Install gcp-emulator
  run: go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest

- name: Start emulator stack
  run: gcp-emulator start --mode=strict

- name: Run tests
  run: go test ./...

- name: Stop emulators
  run: gcp-emulator stop
```

**Why strict mode?** Fail-closed behavior catches permission bugs before deploying to production.

---

## GitHub Actions

### Basic Integration

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install gcp-emulator CLI
        run: go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest

      - name: Start emulator stack
        run: gcp-emulator start --mode=strict

      - name: Wait for services
        run: gcp-emulator status

      - name: Run tests
        run: go test -v ./...

      - name: Check IAM logs for denials
        if: failure()
        run: gcp-emulator logs iam | grep DENY

      - name: Stop emulators
        if: always()
        run: gcp-emulator stop
```

### With Custom Policy

```yaml
- name: Copy production policy
  run: cp .ci/prod-policy.json policy.json

- name: Validate policy
  run: gcp-emulator policy validate policy.json

- name: Start with custom policy
  run: gcp-emulator start --mode=strict --policy-file=policy.json
```

### Matrix Testing (Multiple IAM Modes)

```yaml
strategy:
  matrix:
    iam-mode: [off, permissive, strict]

steps:
  - name: Start emulators with ${{ matrix.iam-mode }}
    run: gcp-emulator start --mode=${{ matrix.iam-mode }}

  - name: Run tests
    run: go test ./...
```

### Caching Docker Images

```yaml
- name: Cache Docker images
  uses: actions/cache@v3
  with:
    path: /var/lib/docker
    key: docker-${{ runner.os }}-${{ hashFiles('docker-compose.yml') }}

- name: Pull images (if not cached)
  run: gcp-emulator start --pull
```

### Upload IAM Logs as Artifacts

```yaml
- name: Upload IAM logs
  if: failure()
  uses: actions/upload-artifact@v3
  with:
    name: iam-logs
    path: |
      /tmp/iam-emulator.log
    retention-days: 7

- name: Capture logs before upload
  if: failure()
  run: gcp-emulator logs iam > /tmp/iam-emulator.log
```

---

## GitLab CI

### Basic Integration

```yaml
test:
  image: golang:1.21
  services:
    - docker:dind
  
  variables:
    DOCKER_HOST: tcp://docker:2376
    DOCKER_TLS_CERTDIR: "/certs"
    DOCKER_TLS_VERIFY: 1
    DOCKER_CERT_PATH: "$DOCKER_TLS_CERTDIR/client"
  
  before_script:
    - go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
  
  script:
    - gcp-emulator start --mode=strict
    - gcp-emulator status
    - go test -v ./...
  
  after_script:
    - gcp-emulator logs iam
    - gcp-emulator stop
```

### Multi-Stage Pipeline

```yaml
stages:
  - validate
  - test
  - deploy

validate-policy:
  stage: validate
  image: golang:1.21
  script:
    - go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
    - gcp-emulator policy validate policy.yaml

integration-test:
  stage: test
  image: golang:1.21
  services:
    - docker:dind
  script:
    - go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
    - gcp-emulator start --mode=strict
    - go test -v ./...
  artifacts:
    when: on_failure
    paths:
      - iam-logs.txt
    expire_in: 1 week
  after_script:
    - gcp-emulator logs iam > iam-logs.txt
```

### With Custom Docker Network

```yaml
test:
  services:
    - name: docker:dind
      command: ["--network=ci-network"]
  
  script:
    - docker network create ci-network || true
    - gcp-emulator start --mode=strict
    - go test -v ./...
```

---

## CircleCI

### Basic Integration

```yaml
version: 2.1

jobs:
  test:
    docker:
      - image: cimg/go:1.21
    
    steps:
      - checkout
      
      - setup_remote_docker:
          version: 20.10.24
      
      - run:
          name: Install gcp-emulator
          command: go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
      
      - run:
          name: Start emulator stack
          command: gcp-emulator start --mode=strict
      
      - run:
          name: Run tests
          command: go test -v ./...
      
      - run:
          name: Check IAM logs
          command: gcp-emulator logs iam | grep DENY || true
          when: on_fail
      
      - run:
          name: Stop emulators
          command: gcp-emulator stop
          when: always

workflows:
  version: 2
  test:
    jobs:
      - test
```

### With Parallelism

```yaml
jobs:
  test:
    docker:
      - image: cimg/go:1.21
    parallelism: 4
    
    steps:
      - checkout
      - setup_remote_docker
      
      - run:
          name: Install gcp-emulator
          command: go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
      
      - run:
          name: Start emulators
          command: gcp-emulator start --mode=strict
      
      - run:
          name: Run tests in parallel
          command: |
            TESTFILES=$(go list ./... | circleci tests split)
            go test -v $TESTFILES
      
      - run:
          name: Stop emulators
          command: gcp-emulator stop
          when: always
```

---

## Jenkins

### Declarative Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        PATH = "${env.HOME}/go/bin:${env.PATH}"
    }
    
    stages {
        stage('Install CLI') {
            steps {
                sh 'go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest'
            }
        }
        
        stage('Validate Policy') {
            steps {
                sh 'gcp-emulator policy validate policy.yaml'
            }
        }
        
        stage('Start Emulators') {
            steps {
                sh 'gcp-emulator start --mode=strict'
                sh 'gcp-emulator status'
            }
        }
        
        stage('Run Tests') {
            steps {
                sh 'go test -v ./...'
            }
        }
    }
    
    post {
        failure {
            sh 'gcp-emulator logs iam | grep DENY'
        }
        always {
            sh 'gcp-emulator stop || true'
        }
    }
}
```

### Scripted Pipeline

```groovy
node {
    try {
        stage('Checkout') {
            checkout scm
        }
        
        stage('Install CLI') {
            sh 'go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest'
        }
        
        stage('Start Emulators') {
            sh 'gcp-emulator start --mode=strict'
        }
        
        stage('Test') {
            sh 'go test -v ./...'
        }
    } catch (Exception e) {
        sh 'gcp-emulator logs iam | grep DENY'
        throw e
    } finally {
        sh 'gcp-emulator stop || true'
    }
}
```

---

## Docker Compose in CI

For environments where the CLI isn't available, use docker-compose directly:

### GitHub Actions

```yaml
- name: Start emulators
  run: docker compose up -d

- name: Wait for services
  run: |
    timeout 30 bash -c 'until curl -f http://localhost:8080/health; do sleep 1; done'

- name: Run tests
  run: go test -v ./...
  env:
    GCP_EMULATOR_HOST: localhost:9090
    GCP_IAM_HOST: localhost:8080

- name: View logs
  if: failure()
  run: docker compose logs iam

- name: Stop emulators
  if: always()
  run: docker compose down
```

### GitLab CI

```yaml
test:
  image: golang:1.21
  services:
    - name: ghcr.io/blackwell-systems/gcp-iam-emulator:latest
      alias: iam
    - name: ghcr.io/blackwell-systems/gcp-secret-manager-emulator:latest
      alias: secretmanager
  
  variables:
    IAM_MODE: strict
    IAM_HOST: iam:8080
  
  script:
    - go test -v ./...
```

---

## IAM Modes for CI

### Off Mode

**Use case:** Fast feedback, no IAM enforcement

```yaml
- name: Start emulators (no IAM)
  run: gcp-emulator start --mode=off

- name: Run tests
  run: go test ./...
```

**When to use:**
- Local development
- Quick iteration
- Performance benchmarks
- Tests that don't involve permissions

### Permissive Mode

**Use case:** Develop with IAM, fail-open on errors

```yaml
- name: Start emulators (permissive)
  run: gcp-emulator start --mode=permissive

- name: Run tests
  run: go test ./...
```

**Behavior:**
- IAM checks enabled
- If IAM emulator unavailable → allow request
- If permission denied → deny request
- Logs denials for review

**When to use:**
- Integration testing during development
- Gradual IAM adoption
- Debugging IAM issues
- Pre-merge checks (warning, not blocking)

### Strict Mode

**Use case:** Production-like behavior, fail-closed

```yaml
- name: Start emulators (strict)
  run: gcp-emulator start --mode=strict

- name: Run tests
  run: go test ./...
```

**Behavior:**
- IAM checks enabled
- If IAM emulator unavailable → deny request
- If permission denied → deny request
- Fail fast on misconfiguration

**When to use:**
- CI/CD gate before merge
- Pre-deployment validation
- Testing with production policies
- Security-critical paths

---

## Testing Strategies

### Strategy 1: Separate Test Suites

```yaml
unit-tests:
  steps:
    # No emulators needed
    - run: go test -short ./...

integration-tests:
  steps:
    - run: gcp-emulator start --mode=permissive
    - run: go test -run Integration ./...

security-tests:
  steps:
    - run: gcp-emulator start --mode=strict
    - run: go test -run Security ./...
```

### Strategy 2: Progressive IAM Testing

```yaml
# Step 1: Validate policy syntax
- run: gcp-emulator policy validate

# Step 2: Test with IAM disabled (baseline)
- run: gcp-emulator start --mode=off
- run: go test ./...
- run: gcp-emulator stop

# Step 3: Test with IAM permissive (catch obvious errors)
- run: gcp-emulator start --mode=permissive
- run: go test ./...
- run: gcp-emulator stop

# Step 4: Test with IAM strict (production parity)
- run: gcp-emulator start --mode=strict
- run: go test ./...
```

### Strategy 3: Production Policy Testing

```yaml
- name: Export production policy
  run: |
    gcloud auth activate-service-account --key-file=${{ secrets.GCP_SA_KEY }}
    gcloud projects get-iam-policy prod-project --format=json > prod-policy.json

- name: Test with production policy
  run: gcp-emulator start --policy-file=prod-policy.json --mode=strict

- name: Run integration tests
  run: go test -tags=integration ./...
```

### Strategy 4: Permission Boundary Testing

Test that your app respects IAM boundaries:

```yaml
- name: Test with minimal permissions (should fail)
  run: |
    cat > minimal-policy.yaml <<EOF
    roles:
      roles/custom.readonly:
        permissions:
          - secretmanager.secrets.get
    groups:
      test-users:
        members:
          - user:test@example.com
    projects:
      test-project:
        bindings:
          - role: roles/custom.readonly
            members:
              - group:test-users
    EOF
    
    gcp-emulator start --policy-file=minimal-policy.yaml --mode=strict
    
    # These should fail with 403
    ! go test -run TestCreateSecret ./...
    ! go test -run TestDeleteSecret ./...
    
    # These should succeed
    go test -run TestGetSecret ./...

- name: Test with full permissions (should succeed)
  run: |
    gcp-emulator stop
    gcp-emulator start --policy-file=full-policy.yaml --mode=strict
    go test ./...
```

---

## Troubleshooting CI

### Services Not Starting

**Problem:** Emulators fail to start

```yaml
- name: Debug emulator startup
  run: |
    gcp-emulator start --mode=strict || true
    docker compose ps
    docker compose logs
```

**Solutions:**
- Check Docker daemon is running
- Verify port availability
- Check docker-compose.yml syntax
- Ensure sufficient memory (>2GB recommended)

### Health Check Failures

**Problem:** Services start but fail health checks

```yaml
- name: Wait with retries
  run: |
    for i in {1..30}; do
      gcp-emulator status && break
      echo "Waiting for services (attempt $i/30)..."
      sleep 2
    done
    gcp-emulator status
```

### Permission Denied Errors

**Problem:** Tests fail with 403 Forbidden

**Debug:**
```yaml
- name: Debug IAM denials
  if: failure()
  run: |
    echo "=== IAM Logs ==="
    gcp-emulator logs iam
    
    echo "=== Policy File ==="
    cat policy.yaml
    
    echo "=== Bindings ==="
    gcp-emulator policy validate policy.yaml
```

**Common causes:**
- Principal header not set (`X-Emulator-Principal`)
- Role not defined in policy
- Permission not granted to role
- Condition expression excludes resource

### IAM Emulator Unreachable

**Problem:** Data plane can't reach IAM emulator

**Check networking:**
```yaml
- name: Test IAM connectivity
  run: |
    curl -v http://localhost:8080/health
    docker network inspect gcp-iam-control-plane_default
```

**Solution:**
```yaml
# Use explicit network
- run: docker network create gcp-network
- run: gcp-emulator start --network=gcp-network
```

### Logs Not Appearing

**Problem:** `gcp-emulator logs` returns no output

```yaml
- name: Capture logs directly
  run: |
    docker compose logs iam > iam.log
    docker compose logs secretmanager > secretmanager.log
    cat iam.log secretmanager.log
```

---

## Best Practices

### 1. Use Strict Mode in CI

Always use strict mode for pre-merge checks:

```yaml
# Good: Fail fast on permission issues
- run: gcp-emulator start --mode=strict

# Bad: Silent failures in CI
- run: gcp-emulator start --mode=off
```

### 2. Validate Policy Before Starting

```yaml
- name: Validate policy
  run: gcp-emulator policy validate policy.yaml

- name: Start emulators
  run: gcp-emulator start --mode=strict
```

### 3. Check IAM Logs on Failure

```yaml
- name: Upload IAM logs
  if: failure()
  uses: actions/upload-artifact@v3
  with:
    name: iam-logs
    path: iam-logs.txt

- name: Capture logs
  if: failure()
  run: gcp-emulator logs iam > iam-logs.txt
```

### 4. Always Stop Emulators

Use `always()` condition to clean up:

```yaml
- name: Stop emulators
  if: always()
  run: gcp-emulator stop
```

### 5. Cache Docker Images

Speed up CI by caching images:

```yaml
- name: Pull images
  run: gcp-emulator start --pull || docker compose pull
```

### 6. Set Explicit Timeouts

Don't wait forever for services:

```yaml
- name: Start with timeout
  run: timeout 60 gcp-emulator start --mode=strict
```

### 7. Test Policy Changes in CI

```yaml
on:
  pull_request:
    paths:
      - 'policy.yaml'
      - 'packs/**'

jobs:
  validate-policy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: gcp-emulator policy validate policy.yaml
```

### 8. Use Environment-Specific Policies

```yaml
- name: Load staging policy
  run: cp .ci/staging-policy.yaml policy.yaml

- name: Test with staging policy
  run: |
    gcp-emulator start --mode=strict
    go test ./...
```

### 9. Monitor CI Performance

Track emulator startup time:

```yaml
- name: Start emulators
  run: |
    START=$(date +%s)
    gcp-emulator start --mode=strict
    END=$(date +%s)
    echo "Startup time: $((END - START))s"
```

### 10. Document CI Requirements

Add to README:

```markdown
## CI Requirements

- Docker 20.10+
- Go 1.21+
- 2GB RAM minimum
- IAM mode: strict
```

---

## Related Documentation

- [Architecture](ARCHITECTURE.md) - System design and components
- [Policy Reference](POLICY_REFERENCE.md) - Policy file syntax
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues
- [CLI Design](CLI_DESIGN.md) - CLI command reference
