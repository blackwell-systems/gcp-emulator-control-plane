# Documentation Index

Complete documentation for the GCP Emulator Control Plane.

---

## Getting Started

**New to the control plane?** Start here:

1. **[Main README](../README.md)** - Quick introduction, installation, and quickstart
2. **[End-to-End Tutorial](END_TO_END_TUTORIAL.md)** - Complete walkthrough from installation to production policy testing
3. **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions

---

## Core Documentation

### Policy Management

**[Policy Reference](POLICY_REFERENCE.md)** - Complete policy file syntax and semantics

Learn how to:
- Define roles and permissions
- Create groups and bindings
- Write CEL conditions for resource-based access
- Use YAML vs JSON formats
- Test with production GCP policies
- Validate policy files

**Topics covered:**
- Policy structure (roles, groups, projects)
- Permission format (`service.resource.verb`)
- Principal format (`user:*`, `serviceAccount:*`, `group:*`)
- Condition evaluation (CEL expressions)
- Policy packs for common services
- Best practices and examples

### CI/CD Integration

**[CI Integration](CI_INTEGRATION.md)** - Complete CI/CD integration guide

Platform-specific examples for:
- GitHub Actions
- GitLab CI
- CircleCI
- Jenkins
- Docker Compose in CI

**Topics covered:**
- IAM modes for CI (off/permissive/strict)
- Testing strategies (progressive IAM, production policy, permission boundaries)
- Troubleshooting CI failures
- Performance optimization
- Best practices

### System Architecture

**[Architecture](ARCHITECTURE.md)** - Control plane design and system architecture

Deep dive into:
- Repository ecosystem (5 repos working together)
- Control plane vs data plane separation
- CLI architecture (Cobra + Viper pattern)
- Request flow and identity propagation
- Authorization model and permission checking
- Failure modes and recovery strategies
- Design decisions and tradeoffs

**Topics covered:**
- Component responsibilities
- Network topology and service discovery
- Data flow diagrams
- Performance characteristics
- Security model
- Custom HTTP gateway vs grpc-gateway decision

---

## Developer Documentation

### Building Emulators

**[Integration Contract](INTEGRATION_CONTRACT.md)** - Contract for building new emulators

If you're building a new GCP emulator to join the mesh:
- Canonical resource naming conventions
- Operation → permission mappings
- Principal injection (inbound)
- Principal propagation (outbound - calling IAM emulator)
- IAM mode configuration
- Health check requirements

### CLI Development

**[CLI Design](CLI_DESIGN.md)** - CLI implementation and architecture

Understand the `gcp-emulator` CLI:
- Command structure (Cobra)
- Configuration management (Viper)
- Docker Compose wrapper
- Policy validation
- Colored output for UX

**[CLI Viper Pattern](CLI_VIPER_PATTERN.md)** - Disciplined Viper usage

Why and how we contain Viper to the config package:
- Explicit Config structs
- Configuration precedence (flags > env > file > defaults)
- Testability patterns

---

## Migration and Maintenance

**[Migration Guide](MIGRATION.md)** - Migrating from standalone emulators

If you're already using GCP emulators:
- Migration from standalone Secret Manager emulator
- Migration from standalone KMS emulator
- Breaking changes and compatibility
- Rollback procedures

**[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions

Common problems:
- Services not starting
- Permission denied errors
- IAM emulator unreachable
- Policy validation failures
- Network issues
- Logs not appearing

---

## Documentation by Use Case

### I want to...

**Get started quickly**
→ [Main README](../README.md) + [End-to-End Tutorial](END_TO_END_TUTORIAL.md)

**Write policy files**
→ [Policy Reference](POLICY_REFERENCE.md)

**Set up CI/CD**
→ [CI Integration](CI_INTEGRATION.md)

**Test with production policies**
→ [Policy Reference - Exporting Production Policies](POLICY_REFERENCE.md#exporting-production-policies) + [Main README - Testing with Production Policies](../README.md#testing-with-production-policies)

**Build a new emulator**
→ [Integration Contract](INTEGRATION_CONTRACT.md) + [Architecture](ARCHITECTURE.md)

**Understand control plane architecture**
→ [Architecture](ARCHITECTURE.md)

**Debug permission denials**
→ [Troubleshooting](TROUBLESHOOTING.md#permission-denied-errors) + [CI Integration - Troubleshooting](CI_INTEGRATION.md#troubleshooting-ci)

**Contribute to the CLI**
→ [CLI Design](CLI_DESIGN.md) + [CLI Viper Pattern](CLI_VIPER_PATTERN.md)

**Migrate from standalone emulators**
→ [Migration Guide](MIGRATION.md)

---

## Documentation Principles

This documentation follows these principles:

1. **README = positioning + quick success** - Get value in 5 minutes
2. **Deep dives = separate docs** - Comprehensive but focused
3. **Examples over explanation** - Show, don't just tell
4. **Real-world use cases** - Production policy testing, CI/CD, conditional access
5. **Progressive disclosure** - Start simple, go deep when needed

---

## Contributing to Documentation

Found an issue or want to improve docs?

- **Typos/errors**: Open an issue or PR
- **Missing examples**: We'd love more real-world examples
- **Unclear sections**: Let us know what's confusing
- **New use cases**: Share your integration patterns

---

## External Resources

- **[GCP IAM Documentation](https://cloud.google.com/iam/docs)** - Official GCP IAM reference
- **[CEL Specification](https://github.com/google/cel-spec)** - Common Expression Language spec
- **[gRPC Documentation](https://grpc.io/docs/)** - gRPC protocol and APIs
- **[Docker Compose Documentation](https://docs.docker.com/compose/)** - Docker Compose reference

---

## Quick Links

| Document | Description | When to Read |
|----------|-------------|--------------|
| [README](../README.md) | Quick start and overview | First time |
| [Policy Reference](POLICY_REFERENCE.md) | Complete policy syntax | Writing policies |
| [CI Integration](CI_INTEGRATION.md) | CI/CD setup and examples | Setting up CI |
| [Architecture](ARCHITECTURE.md) | System design and internals | Understanding how it works |
| [Integration Contract](INTEGRATION_CONTRACT.md) | Emulator requirements | Building new emulators |
| [CLI Design](CLI_DESIGN.md) | CLI implementation | Contributing to CLI |
| [End-to-End Tutorial](END_TO_END_TUTORIAL.md) | Complete walkthrough | Learning by doing |
| [Troubleshooting](TROUBLESHOOTING.md) | Common issues | Something's broken |
| [Migration Guide](MIGRATION.md) | Migration from standalone | Upgrading existing setup |
