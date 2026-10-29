// Package payments is the payments module (P6): Payment intents and PSP adapters (sandbox until FG-1b).
//
// Database schema: payments (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/payments/contract; this module depends on internal/platform only.
// Implementation starts in P6; nothing here is active in Phase 1.
package payments
