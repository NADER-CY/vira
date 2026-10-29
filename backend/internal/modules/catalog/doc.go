// Package catalog is the catalog module (P4): Products, variants and inventory.
//
// Database schema: catalog (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/catalog/contract; this module depends on internal/platform only.
// Implementation starts in P4; nothing here is active in Phase 1.
package catalog
