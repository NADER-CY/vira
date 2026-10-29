// Package orders is the orders module (P6): Orders, fulfilment, returns.
//
// Database schema: orders (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/orders/contract; this module depends on internal/platform only.
// Implementation starts in P6; nothing here is active in Phase 1.
package orders
