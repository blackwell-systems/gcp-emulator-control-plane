// Package gcp-iam-control-plane provides orchestration for the GCP Emulator ecosystem.
//
// This is the control plane repository that unifies the IAM, Secret Manager, and KMS
// emulators with centralized authorization policy.
//
// # Overview
//
// The control plane provides:
//   - gcp-emulator CLI for stack management
//   - docker-compose orchestration
//   - Centralized policy.yaml for IAM
//   - Integration contract documentation
//
// # Installation
//
//	go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
//
// # Quick Start
//
//	gcp-emulator start
//	gcp-emulator status
//	gcp-emulator policy validate
//
// # Architecture
//
// The ecosystem consists of 5 repositories:
//   - gcp-iam-control-plane: Orchestration layer (this repo)
//   - gcp-iam-emulator: Authorization engine
//   - gcp-secret-manager-emulator: Secret Manager data plane
//   - gcp-kms-emulator: KMS data plane
//   - gcp-emulator-auth: Shared auth library
//
// # Documentation
//
// For complete documentation, see:
//   - README.md: Quickstart and usage
//   - docs/ARCHITECTURE.md: System design and repository ecosystem
//   - docs/CLI_DESIGN.md: CLI design principles
//   - docs/END_TO_END_TUTORIAL.md: Complete walkthrough
//
// # License
//
// Apache 2.0 - See LICENSE file for details.
package main
