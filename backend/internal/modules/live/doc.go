// Package live is the live module (P8): Live shopping and calls.
//
// Database schema: live (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/live/contract; this module depends on internal/platform only.
// Implementation starts in P8; nothing here is active in Phase 1.
package live
