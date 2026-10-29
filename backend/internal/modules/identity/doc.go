// Package identity is the identity module (P2): Users, phone OTP sign-in, devices, sessions and token issuance.
//
// Database schema: identity (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/identity/contract; this module depends on internal/platform only.
// Implementation starts in P2; nothing here is active in Phase 1.
package identity
