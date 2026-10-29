// Package media is the media module (P3): Media metadata, upload pipeline and the VideoProvider adapter (ADR-009).
//
// Database schema: media (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/media/contract; this module depends on internal/platform only.
// Implementation starts in P3; nothing here is active in Phase 1.
package media
