// Package support is the support module (P7): Support tickets.
//
// Database schema: support (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/support/contract; this module depends on internal/platform only.
// Implementation starts in P7; nothing here is active in Phase 1.
package support
