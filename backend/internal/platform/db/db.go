// Package db provides the PostgreSQL pool and the transaction helper that every module uses.
//
// Rules (doc 06 §1, §3):
//   - Each runtime connects with its own role (app_public, app_admin, app_worker); runtime roles never own tables.
//   - PgBouncer runs in transaction mode, so all session state is SET LOCAL inside an explicit transaction.
//   - Seller-scoped and public requests set app.audience / app.store_id with SET LOCAL before any query,
//     which the row-level-security policies read through platform.current_audience()/current_store_id().
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the shared connection pool type.
type Pool = pgxpool.Pool

// Tx is the transaction handle passed to transactional functions.
type Tx = pgx.Tx

// Options configures a pool.
type Options struct {
	URL       string
	MaxConns  int32
	PgBouncer bool   // transaction pooling: no server-side prepared statement cache
	AppName   string // application_name, visible in pg_stat_activity
}

// Open creates and pings a pool.
func Open(ctx context.Context, o Options) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(o.URL)
	if err != nil {
		return nil, fmt.Errorf("db: parse url: %w", err)
	}
	if o.MaxConns > 0 {
		cfg.MaxConns = o.MaxConns
	}
	cfg.MaxConnIdleTime = 5 * time.Minute
	if o.AppName != "" {
		cfg.ConnConfig.RuntimeParams["application_name"] = o.AppName
	}
	if o.PgBouncer {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// Audience values understood by the RLS policies.
const (
	AudiencePublic = "public"
	AudienceSeller = "seller"
	AudienceSystem = "system"
	AudienceAdmin  = "admin"
)

// Scope is the request context written with SET LOCAL at the start of every transaction.
type Scope struct {
	Audience string // required: public | seller | system | admin
	StoreID  string // seller audience only (UUID string, already authorized by the caller)
	ActorID  string // optional, for auditing triggers and diagnostics
}

// TxOptions controls isolation.
type TxOptions struct {
	Isolation pgx.TxIsoLevel // default read committed
	ReadOnly  bool
}

// ErrScopeRequired is returned when a transaction is started without an audience.
var ErrScopeRequired = errors.New("db: transaction scope (audience) is required")

// InTx runs fn in a transaction with the given scope. The transaction is committed when fn returns nil and
// rolled back otherwise (including panics). Serialization failures are not retried here; callers that use
// SERIALIZABLE (ledger transaction scripts, Phase 6) wrap InTx in their own retry loop.
func InTx(ctx context.Context, pool *Pool, scope Scope, opts TxOptions, fn func(Tx) error) (err error) {
	if scope.Audience == "" {
		return ErrScopeRequired
	}
	mode := pgx.ReadWrite
	if opts.ReadOnly {
		mode = pgx.ReadOnly
	}
	iso := opts.Isolation
	if iso == "" {
		iso = pgx.ReadCommitted
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: iso, AccessMode: mode})
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = SetScope(ctx, tx, scope); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

// SetScope writes the RLS context with transaction-local settings (equivalent to SET LOCAL).
func SetScope(ctx context.Context, tx Tx, s Scope) error {
	_, err := tx.Exec(ctx,
		`SELECT set_config('app.audience', $1, true), set_config('app.store_id', $2, true), set_config('app.actor_id', $3, true)`,
		s.Audience, s.StoreID, s.ActorID)
	if err != nil {
		return fmt.Errorf("db: set scope: %w", err)
	}
	return nil
}

// IsUniqueViolation reports whether err is a unique-constraint violation.
func IsUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

// IsCheckViolation reports whether err is a check-constraint violation.
func IsCheckViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23514"
}

// ErrNoRows aliases pgx.ErrNoRows.
var ErrNoRows = pgx.ErrNoRows
