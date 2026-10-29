// Package migrate applies the embedded SQL migrations.
//
// Guarantees:
//   - a cluster-wide advisory lock serializes concurrent runners;
//   - each migration runs in its own transaction (unless marked "-- vira:no-transaction");
//   - applied migrations are recorded with a SHA-256 checksum; editing an applied migration is an error;
//   - down migrations exist for development/test only and are refused when VIRA_ENV is staging/production.
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Migration is one versioned migration.
type Migration struct {
	Version  int
	Name     string
	Up       string
	Down     string
	Checksum string
	NoTx     bool
}

// Status describes one migration's state.
type Status struct {
	Version   int
	Name      string
	Applied   bool
	AppliedAt time.Time
}

var fileRe = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.(up|down)\.sql$`)

const lockID = 7_314_202_601 // arbitrary constant for pg_advisory_lock

// Load reads migrations from fsys (root directory).
func Load(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	byVersion := map[int]*Migration{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migrate: bad file name %q (want NNNN_name.up.sql / .down.sql)", e.Name())
		}
		v, _ := strconv.Atoi(m[1])
		b, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		mg := byVersion[v]
		if mg == nil {
			mg = &Migration{Version: v, Name: m[2]}
			byVersion[v] = mg
		} else if mg.Name != m[2] {
			return nil, fmt.Errorf("migrate: version %04d has two names (%s, %s)", v, mg.Name, m[2])
		}
		if m[3] == "up" {
			mg.Up = string(b)
			sum := sha256.Sum256(b)
			mg.Checksum = hex.EncodeToString(sum[:])
			mg.NoTx = strings.HasPrefix(strings.TrimSpace(mg.Up), "-- vira:no-transaction")
		} else {
			mg.Down = string(b)
		}
	}
	out := make([]Migration, 0, len(byVersion))
	for _, m := range byVersion {
		if m.Up == "" {
			return nil, fmt.Errorf("migrate: %04d_%s has no up migration", m.Version, m.Name)
		}
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	for i, m := range out {
		if m.Version != i+1 {
			return nil, fmt.Errorf("migrate: versions must be contiguous from 0001; found %04d at position %d", m.Version, i+1)
		}
	}
	return out, nil
}

// Runner applies migrations over a single dedicated connection (the migrator role).
type Runner struct {
	Conn       *pgx.Conn
	Migrations []Migration
	AllowDown  bool
	Log        func(format string, args ...any)
}

func (r *Runner) logf(format string, args ...any) {
	if r.Log != nil {
		r.Log(format, args...)
	}
}

const ledgerTable = `CREATE TABLE IF NOT EXISTS public.vira_schema_migrations (
  version    integer     PRIMARY KEY,
  name       text        NOT NULL,
  checksum   text        NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now()
)`

func (r *Runner) withLock(ctx context.Context, fn func() error) error {
	if _, err := r.Conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	defer r.Conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockID) //nolint:errcheck
	if _, err := r.Conn.Exec(ctx, ledgerTable); err != nil {
		return fmt.Errorf("migrate: ledger table: %w", err)
	}
	return fn()
}

func (r *Runner) applied(ctx context.Context) (map[int]Status, map[int]string, error) {
	rows, err := r.Conn.Query(ctx, `SELECT version, name, checksum, applied_at FROM public.vira_schema_migrations`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	st, sums := map[int]Status{}, map[int]string{}
	for rows.Next() {
		var s Status
		var sum string
		if err := rows.Scan(&s.Version, &s.Name, &sum, &s.AppliedAt); err != nil {
			return nil, nil, err
		}
		s.Applied = true
		st[s.Version], sums[s.Version] = s, sum
	}
	return st, sums, rows.Err()
}

// ErrChecksumMismatch means an applied migration file was edited.
var ErrChecksumMismatch = errors.New("migrate: checksum mismatch (applied migrations must never be edited)")

func (r *Runner) verify(sums map[int]string) error {
	known := map[int]bool{}
	for _, m := range r.Migrations {
		known[m.Version] = true
		if s, ok := sums[m.Version]; ok && s != m.Checksum {
			return fmt.Errorf("%w: %04d_%s", ErrChecksumMismatch, m.Version, m.Name)
		}
	}
	for v := range sums {
		if !known[v] {
			return fmt.Errorf("migrate: database has version %04d which this build does not know (newer schema?)", v)
		}
	}
	return nil
}

// Up applies all pending migrations, or up to target when target > 0. Returns the number applied.
func (r *Runner) Up(ctx context.Context, target int) (int, error) {
	n := 0
	err := r.withLock(ctx, func() error {
		_, sums, err := r.applied(ctx)
		if err != nil {
			return err
		}
		if err := r.verify(sums); err != nil {
			return err
		}
		for _, m := range r.Migrations {
			if target > 0 && m.Version > target {
				break
			}
			if _, done := sums[m.Version]; done {
				continue
			}
			start := time.Now()
			if err := r.apply(ctx, m); err != nil {
				return fmt.Errorf("migrate: %04d_%s: %w", m.Version, m.Name, err)
			}
			r.logf("applied %04d_%s in %s", m.Version, m.Name, time.Since(start).Round(time.Millisecond))
			n++
		}
		return nil
	})
	return n, err
}

func (r *Runner) apply(ctx context.Context, m Migration) error {
	record := `INSERT INTO public.vira_schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`
	if m.NoTx {
		if _, err := r.Conn.Exec(ctx, m.Up); err != nil {
			return err
		}
		_, err := r.Conn.Exec(ctx, record, m.Version, m.Name, m.Checksum)
		return err
	}
	return pgx.BeginFunc(ctx, r.Conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, m.Up); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, record, m.Version, m.Name, m.Checksum)
		return err
	})
}

// Down reverts the last `steps` applied migrations (development/test only).
func (r *Runner) Down(ctx context.Context, steps int) (int, error) {
	if !r.AllowDown {
		return 0, errors.New("migrate: down migrations are disabled in this environment (forward-only)")
	}
	n := 0
	err := r.withLock(ctx, func() error {
		st, sums, err := r.applied(ctx)
		if err != nil {
			return err
		}
		if err := r.verify(sums); err != nil {
			return err
		}
		for i := len(r.Migrations) - 1; i >= 0 && n < steps; i-- {
			m := r.Migrations[i]
			if !st[m.Version].Applied {
				continue
			}
			if m.Down == "" {
				return fmt.Errorf("migrate: %04d_%s has no down migration", m.Version, m.Name)
			}
			err := pgx.BeginFunc(ctx, r.Conn, func(tx pgx.Tx) error {
				if _, err := tx.Exec(ctx, m.Down); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, `DELETE FROM public.vira_schema_migrations WHERE version = $1`, m.Version)
				return err
			})
			if err != nil {
				return fmt.Errorf("migrate: down %04d_%s: %w", m.Version, m.Name, err)
			}
			r.logf("reverted %04d_%s", m.Version, m.Name)
			n++
		}
		return nil
	})
	return n, err
}

// Status lists every known migration with its state.
func (r *Runner) Status(ctx context.Context) ([]Status, error) {
	var out []Status
	err := r.withLock(ctx, func() error {
		st, sums, err := r.applied(ctx)
		if err != nil {
			return err
		}
		if err := r.verify(sums); err != nil {
			return err
		}
		for _, m := range r.Migrations {
			s := st[m.Version]
			s.Version, s.Name = m.Version, m.Name
			out = append(out, s)
		}
		return nil
	})
	return out, err
}
