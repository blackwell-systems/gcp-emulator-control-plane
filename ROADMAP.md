# Roadmap

Shared concerns and future improvements for the GCP Emulator ecosystem.

---

## Ecosystem-Wide Improvements

### Error Handling Consistency

**Status:** Under consideration

**Problem:** HTTP gateways (Secret Manager, KMS) currently use inconsistent error responses:
- Raw `http.Error()` with inline JSON strings
- No trace IDs for request correlation
- No structured details for debugging
- No retry signals for clients
- Format: `{"error":"message"}` varies by endpoint

**Proposed Solution:** Integrate [err-envelope](https://github.com/blackwell-systems/err-envelope) into HTTP gateway components.

**Benefits:**
- Consistent error format across all emulators
- Trace ID propagation from HTTP gateway → gRPC → IAM emulator
- Structured details for better debugging
- Retry signals (retryable: true/false)
- slog integration for structured logging
- Client-friendly error contracts

**Scope:**
- gcp-secret-manager-emulator HTTP gateway
- gcp-kms-emulator HTTP gateway
- (Not CLI - that's terminal errors, not HTTP responses)

**Migration Strategy:**
- Start with Secret Manager Gateway as proof of concept
- Add feature flag: `GATEWAY_ERROR_FORMAT=legacy|envelope` for backwards compatibility
- Test with existing clients
- Document error format in emulator READMEs
- Roll out to KMS if successful

**Open Questions:**
- Breaking change concern - does format change impact existing clients?
- Should we version the endpoint (`/v2/`) for new format?
- Trace ID propagation strategy - HTTP header vs gRPC metadata?

---

## CLI Improvements

### Policy Management

- Add policy templates for common patterns (multi-tenant, least privilege, CI/CD)
- Policy diff command (`gcp-emulator policy diff`)
- Policy lint/security check command

### Testing Utilities

- Permission test command (`gcp-emulator test permission`)
- Simulate IAM check against policy without starting stack
- Validate principal format

### Developer Experience

- Shell completion improvements (dynamic suggestions based on running services)
- Better error messages with actionable remediation steps
- Interactive mode for exploring policy

---

## Integration Contract

### New Emulator Support

Future emulators to add:
- Cloud Storage (bucket operations)
- Pub/Sub (topic/subscription management)
- Firestore (document database)

Requirements for new emulators:
- Follow [Integration Contract](docs/INTEGRATION_CONTRACT.md)
- Support IAM mode configuration (off/permissive/strict)
- Implement canonical resource naming
- Map operations to GCP permission names
- Support principal propagation (HTTP + gRPC)

---

## Documentation

- Video walkthrough of setting up multi-tenant IAM
- Blog post series on emulator architecture
- OpenAPI/Swagger specs for HTTP gateways
- Postman collections for testing

---

## Notes

This roadmap tracks shared ecosystem concerns. Individual emulator repos may have their own roadmaps for service-specific features.

Items move from "Under consideration" → "Planned" → "In progress" → "Done" as decisions are made.
