# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-01-27

### Added
- **Initial Release**: Complete GCP emulator control plane orchestration
- **Multi-Service Docker Compose**: Orchestrate IAM, Secret Manager, and KMS emulators
  - IAM Emulator as control plane (port 8080)
  - Secret Manager Emulator as data plane (ports 9090 gRPC, 8081 HTTP)
  - KMS Emulator as data plane (ports 9091 gRPC, 8082 HTTP)
  - Health check dependencies
  - Service-specific IAM mode configuration
- **Policy Orchestration**: Single `policy.yaml` drives all authorization
  - Custom role definitions
  - Group membership management
  - Project-level IAM bindings
  - Conditional access with CEL expressions
  - Centralized policy source of truth
- **Comprehensive Documentation**:
  - README with quickstart and architecture overview
  - END_TO_END_TUTORIAL.md - Complete 30-minute walkthrough
  - ARCHITECTURE.md - System design, request flows, component interactions
  - MIGRATION.md - Step-by-step migration from standalone emulators
  - TROUBLESHOOTING.md - Common issues, debugging tools, solutions
  - INTEGRATION_CONTRACT.md - Stable contract for emulator authors
- **Policy Packs**: Ready-to-use role definitions
  - Secret Manager permissions
  - KMS permissions
  - CI/CD patterns
- **Examples**: Working code samples
  - Go SDK integration
  - REST/curl scripts
  - Multi-service integration patterns

### Features
- **One Policy File**: Offline, deterministic authorization universe
- **One Identity Channel**: Consistent principal injection (gRPC + HTTP)
- **Cross-Service Authorization**: Same policy engine across all emulators
- **Three IAM Modes**: off (legacy), permissive (fail-open), strict (fail-closed)
- **CI-Friendly**: Hermetic testing without cloud credentials
- **Production-Like Behavior**: Tests mirror real GCP IAM enforcement

### Emulator Versions
- IAM Emulator: v0.5.0+
- Secret Manager: v1.2.0+ (with IAM integration)
- KMS: v0.2.0+ (with IAM integration)

### Documentation Highlights
- 11-part end-to-end tutorial covering all features
- Complete architecture with diagrams and request flows
- Three migration strategies (gradual rollout recommended)
- Troubleshooting guide with debugging tools
- Integration contract for adding new emulators

### Design Philosophy
- Control plane + data plane separation
- Stateless data plane (IAM makes all decisions)
- Opt-in IAM integration (non-breaking)
- Fail-open vs fail-closed configurability
- Single source of truth for policy

### Use Cases
- Local development with IAM enforcement
- Integration tests with realistic permission checks
- CI/CD pipelines with strict mode
- Multi-service testing without GCP credentials
- Authorization logic validation

[Unreleased]: https://github.com/blackwell-systems/gcp-emulator-control-plane/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/blackwell-systems/gcp-emulator-control-plane/releases/tag/v0.1.0
