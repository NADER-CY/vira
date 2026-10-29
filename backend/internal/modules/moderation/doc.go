// Package moderation is the moderation module (P2): Reports and moderation decisions.
//
// Database schema: moderation (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/moderation/contract; this module depends on internal/platform only.
// Implementation starts in P2; nothing here is active in Phase 1.
package moderation
