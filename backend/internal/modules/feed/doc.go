// Package feed is the feed module (P2): Home feed and discovery (derived, rebuildable).
//
// Database schema: feed (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/feed/contract; this module depends on internal/platform only.
// Implementation starts in P2; nothing here is active in Phase 1.
package feed
