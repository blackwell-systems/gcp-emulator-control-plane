#!/bin/bash
# Quick smoke test - verifies CLI builds and basic commands work

set -e

echo "=== Smoke Test ==="
echo

# Build CLI
echo "Building CLI..."
go build -o bin/gcp-emulator-smoke ./cmd/gcp-emulator
echo "✓ CLI built successfully"

# Test version command
echo
echo "Testing version command..."
./bin/gcp-emulator-smoke version
echo "✓ Version command works"

# Test help command
echo
echo "Testing help command..."
./bin/gcp-emulator-smoke --help > /dev/null
echo "✓ Help command works"

# Test policy validate (without starting stack)
echo
echo "Testing policy validate..."
./bin/gcp-emulator-smoke policy validate
echo "✓ Policy validation works"

# Test config get (without starting stack)
echo
echo "Testing config get..."
./bin/gcp-emulator-smoke config get > /dev/null
echo "✓ Config get works"

# Cleanup
rm -f bin/gcp-emulator-smoke

echo
echo "=== Smoke test passed! ==="
echo
echo "Note: Full e2e tests require Docker images from GHCR."
echo "Run 'make test-e2e' to test with actual emulators."
