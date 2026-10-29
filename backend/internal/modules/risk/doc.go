// Package risk is the risk module (P4): Fraud and risk signals.
//
// Database schema: risk (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/risk/contract; this module depends on internal/platform only.
// Implementation starts in P4; nothing here is active in Phase 1.
package risk
