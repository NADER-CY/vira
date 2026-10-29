// Package ledger is the ledger module (P6): Double-entry ledger in integer Rial; append-only.
//
// Database schema: ledger (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/ledger/contract; this module depends on internal/platform only.
// Implementation starts in P6; nothing here is active in Phase 1.
package ledger
