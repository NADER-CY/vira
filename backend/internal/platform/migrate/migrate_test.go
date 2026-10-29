package migrate_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/NADER-CY/vira/backend/internal/platform/migrate"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
	"github.com/NADER-CY/vira/backend/migrations"
)

func TestLoadEmbeddedMigrations(t *testing.T) {
	ms, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 3 {
		t.Fatalf("expected at least 3 migrations, got %d", len(ms))
	}
	for _, m := range ms {
		if m.Down == "" {
			t.Errorf("%04d_%s: missing down migration (required for dev/test reversibility)", m.Version, m.Name)
		}
		if len(m.Checksum) != 64 {
			t.Errorf("%04d: bad checksum", m.Version)
		}
	}
}

func TestLoadRejectsBadNamesAndGaps(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"bad name":   {"001_x.up.sql": {Data: []byte("SELECT 1")}},
		"gap":        {"0001_a.up.sql": {Data: []byte("SELECT 1")}, "0003_c.up.sql": {Data: []byte("SELECT 1")}},
		"down only":  {"0001_a.down.sql": {Data: []byte("SELECT 1")}},
		"name clash": {"0001_a.up.sql": {Data: []byte("SELECT 1")}, "0001_b.down.sql": {Data: []byte("SELECT 1")}},
	}
	for name, fsys := range cases {
		if _, err := migrate.Load(fsys); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

// TestUpDownUp proves every migration applies on an empty database, reverts cleanly, and re-applies.
func TestUpDownUp(t *testing.T) {
	db := testdb.New(t, false)
	ctx := context.Background()
	r, conn, err := db.Runner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	n, err := r.Up(ctx, 0)
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	if n != len(r.Migrations) {
		t.Fatalf("applied %d, want %d", n, len(r.Migrations))
	}
	if again, err := r.Up(ctx, 0); err != nil || again != 0 {
		t.Fatalf("second up must be a no-op: n=%d err=%v", again, err)
	}
	back, err := r.Down(ctx, len(r.Migrations))
	if err != nil {
		t.Fatalf("down: %v", err)
	}
	if back != len(r.Migrations) {
		t.Fatalf("reverted %d, want %d", back, len(r.Migrations))
	}
	var schemas int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM pg_namespace WHERE nspname IN ('platform','authz','ledger')`).Scan(&schemas); err != nil {
		t.Fatal(err)
	}
	if schemas != 0 {
		t.Fatalf("down left %d schemas behind", schemas)
	}
	if n, err := r.Up(ctx, 0); err != nil || n != len(r.Migrations) {
		t.Fatalf("re-up: n=%d err=%v", n, err)
	}
}

func TestChecksumMismatchIsRefused(t *testing.T) {
	db := testdb.New(t, true)
	ctx := context.Background()
	r, conn, err := db.Runner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	r.Migrations[0].Checksum = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := r.Up(ctx, 0); !errors.Is(err, migrate.ErrChecksumMismatch) {
		t.Fatalf("want checksum mismatch, got %v", err)
	}
}

func TestDownRefusedWhenNotAllowed(t *testing.T) {
	r := &migrate.Runner{AllowDown: false}
	if _, err := r.Down(context.Background(), 1); err == nil {
		t.Fatal("down must be refused when AllowDown=false")
	}
}

// TestRuntimeRolesDoNotOwnObjects enforces doc 06 §1: runtime roles never own tables (so RLS cannot be
// bypassed by ownership) and only app_admin bypasses RLS.
func TestRuntimeRolesDoNotOwnObjects(t *testing.T) {
	db := testdb.New(t, true)
	ctx := context.Background()
	var owned int
	err := db.Pool("superuser").QueryRow(ctx, `
		SELECT count(*) FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner
		WHERE r.rolname IN ('app_public','app_admin','app_worker')`).Scan(&owned)
	if err != nil {
		t.Fatal(err)
	}
	if owned != 0 {
		t.Fatalf("runtime roles own %d relations", owned)
	}
	rows, err := db.Pool("superuser").Query(ctx, `SELECT rolname, rolbypassrls FROM pg_roles WHERE rolname LIKE 'app_%' OR rolname = 'migrator'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var bypass bool
		if err := rows.Scan(&name, &bypass); err != nil {
			t.Fatal(err)
		}
		if bypass != (name == "app_admin") {
			t.Errorf("role %s: bypassrls=%v", name, bypass)
		}
	}
}
