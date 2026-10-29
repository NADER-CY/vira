// Package kyc is the kyc module (P4): KYC levels, evidence references and decisions (highly sensitive).
//
// Database schema: kyc (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/kyc/contract; this module depends on internal/platform only.
// Implementation starts in P4; nothing here is active in Phase 1.
package kyc
