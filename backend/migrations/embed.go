// Package migrations embeds the SQL migrations and the cluster bootstrap script.
//
// Naming: NNNN_description.up.sql (required) and NNNN_description.down.sql (development/test only;
// production is forward-only with expand → migrate → contract, doc 06 §10). A migration whose first
// line is "-- vira:no-transaction" runs outside a transaction (for CREATE INDEX CONCURRENTLY).
package migrations

import "embed"

// FS holds every migration file.
//
//go:embed *.sql
var FS embed.FS

// BootstrapRoles creates the cluster roles (run once by a superuser, see bootstrap/roles.sql).
//
//go:embed bootstrap/roles.sql
var BootstrapRoles string
