// Package notify is the notify module (P5): Notifications for users and seller staff (doc 04 §10.5).
//
// Database schema: notify (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/notify/contract; this module depends on internal/platform only.
// Implementation starts in P5; nothing here is active in Phase 1.
package notify
