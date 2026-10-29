// Package checkout is the checkout module (P6): Cart and checkout (PSP sandbox until FG-1b).
//
// Database schema: checkout (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/checkout/contract; this module depends on internal/platform only.
// Implementation starts in P6; nothing here is active in Phase 1.
package checkout
