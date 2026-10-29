// Package social is the social module (P2): Social graph, posts, comments and reactions.
//
// Database schema: social (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/social/contract; this module depends on internal/platform only.
// Implementation starts in P2; nothing here is active in Phase 1.
package social
