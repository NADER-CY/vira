// Package testdb creates real, isolated PostgreSQL databases for integration tests.
//
// Set VIRA_TEST_PG_URL to a superuser DSN (e.g. postgres://postgres:postgres@localhost:5432/postgres).
// Without it, integration tests are skipped — unless VIRA_REQUIRE_DB=1 (CI), in which case they fail,
// so a missing database can never be reported as a pass.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NADER-CY/vira/backend/internal/platform/migrate"
	"github.com/NADER-CY/vira/backend/migrations"
)

// Role passwords used only by test databases.
const testPasswordPrefix = "vira_test_only_"

// DB is a migrated test database.
type DB struct {
	Name  string
	urls  map[string]string
	pools map[string]*pgxpool.Pool
	mu    sync.Mutex
	t     testing.TB
}

var bootstrapOnce sync.Once
var bootstrapErr error

func superURL(t testing.TB) string {
	u := os.Getenv("VIRA_TEST_PG_URL")
	if u == "" {
		if os.Getenv("VIRA_REQUIRE_DB") == "1" {
			t.Fatal("VIRA_TEST_PG_URL is not set but VIRA_REQUIRE_DB=1")
		}
		t.Skip("integration test: set VIRA_TEST_PG_URL to run")
	}
	return u
}

func withDB(base, dbname, user, pass string) string {
	u, err := url.Parse(base)
	if err != nil {
		panic(err)
	}
	u.Path = "/" + dbname
	if user != "" {
		u.User = url.UserPassword(user, pass)
	}
	return u.String()
}

// New creates a fresh database owned by migrator, applies all migrations and registers cleanup.
// Pass migrate=false to get an empty database (migration tests).
func New(t testing.TB, applyMigrations bool) *DB {
	t.Helper()
	base := superURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("testdb: connect superuser: %v", err)
	}
	defer admin.Close(ctx)

	bootstrapOnce.Do(func() {
		// Test packages run as parallel processes against one cluster; concurrent CREATE ROLE can race,
		// so the idempotent script is retried.
		retry := func(sql string) error {
			var err error
			for attempt := 0; attempt < 10; attempt++ {
				if _, err = admin.Exec(ctx, sql); err == nil {
					return nil
				}
				time.Sleep(time.Duration(100*(attempt+1)) * time.Millisecond)
			}
			return err
		}
		if bootstrapErr = retry(migrations.BootstrapRoles); bootstrapErr != nil {
			return
		}
		for _, r := range []string{"migrator", "app_public", "app_admin", "app_worker"} {
			if bootstrapErr = retry(fmt.Sprintf(`ALTER ROLE %s PASSWORD '%s%s'`, r, testPasswordPrefix, r)); bootstrapErr != nil {
				return
			}
		}
	})
	if bootstrapErr != nil {
		t.Fatalf("testdb: bootstrap roles: %v", bootstrapErr)
	}

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	name := "vira_test_" + hex.EncodeToString(b)
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s OWNER migrator`, name)); err != nil {
		t.Fatalf("testdb: create database: %v", err)
	}
	for _, r := range []string{"app_public", "app_admin", "app_worker"} {
		if _, err := admin.Exec(ctx, fmt.Sprintf(`GRANT CONNECT ON DATABASE %s TO %s`, name, r)); err != nil {
			t.Fatalf("testdb: grant connect: %v", err)
		}
	}
	d := &DB{Name: name, urls: map[string]string{}, pools: map[string]*pgxpool.Pool{}, t: t}
	for _, r := range []string{"migrator", "app_public", "app_admin", "app_worker"} {
		d.urls[r] = withDB(base, name, r, testPasswordPrefix+r)
	}
	d.urls["superuser"] = withDB(base, name, "", "")

	t.Cleanup(func() {
		d.mu.Lock()
		for _, p := range d.pools {
			p.Close()
		}
		d.mu.Unlock()
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		c, err := pgx.Connect(cctx, base)
		if err != nil {
			return
		}
		defer c.Close(cctx)
		_, _ = c.Exec(cctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name))
	})

	if applyMigrations {
		if _, err := d.Migrate(ctx, 0); err != nil {
			t.Fatalf("testdb: migrate: %v", err)
		}
	}
	return d
}

// URL returns the DSN for a role: migrator, app_public, app_admin, app_worker or superuser.
func (d *DB) URL(role string) string {
	u, ok := d.urls[role]
	if !ok {
		d.t.Fatalf("testdb: unknown role %q", role)
	}
	return u
}

// Pool returns a (cached) pool connected as role.
func (d *DB) Pool(role string) *pgxpool.Pool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.pools[role]; ok {
		return p
	}
	p, err := pgxpool.New(context.Background(), d.URL(role))
	if err != nil {
		d.t.Fatalf("testdb: pool %s: %v", role, err)
	}
	d.pools[role] = p
	return p
}

// Runner returns a migration runner connected as migrator; the caller closes the connection.
func (d *DB) Runner(ctx context.Context) (*migrate.Runner, *pgx.Conn, error) {
	ms, err := migrate.Load(migrations.FS)
	if err != nil {
		return nil, nil, err
	}
	conn, err := pgx.Connect(ctx, d.URL("migrator"))
	if err != nil {
		return nil, nil, err
	}
	return &migrate.Runner{Conn: conn, Migrations: ms, AllowDown: true}, conn, nil
}

// Migrate applies migrations up to target (0 = all).
func (d *DB) Migrate(ctx context.Context, target int) (int, error) {
	r, conn, err := d.Runner(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close(ctx)
	return r.Up(ctx, target)
}
