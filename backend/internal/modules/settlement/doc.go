// Package settlement is the settlement module (P7): Seller settlement and payouts (model chosen at FG-1a; live only after FG-1b).
//
// Database schema: settlement (owned by migrator; created empty in migration 0003).
// Boundary rules (enforced by internal/platform/archlint): other modules may import only
// internal/modules/settlement/contract; this module depends on internal/platform only.
// Implementation starts in P7; nothing here is active in Phase 1.
package settlement
