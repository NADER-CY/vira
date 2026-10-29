// Package messaging is the messaging module (P5): Private conversations and delivery (doc 07 §3).
//
// Database schema: messaging (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/messaging/contract; this module depends on internal/platform only.
// Implementation starts in P5; nothing here is active in Phase 1.
package messaging
