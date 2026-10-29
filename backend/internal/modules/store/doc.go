// Package store is the store module (P4): Stores, sellers and the shared seller identity (doc 04 §10).
//
// Database schema: store (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/store/contract; this module depends on internal/platform only.
// Implementation starts in P4; nothing here is active in Phase 1.
package store
