package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
)

// TestRowLevelSecurityPattern exercises the seller-isolation pattern of doc 06 §3 on a fixture table:
// policies read platform.current_audience()/current_store_id(); with no context they fail closed.
func TestRowLevelSecurityPattern(t *testing.T) {
	d := testdb.New(t, true)
	ctx := context.Background()
	mig := d.Pool("migrator")
	storeA, storeB := ids.New(), ids.New()
	stmts := []string{
		`CREATE SCHEMA rls_fixture`,
		`CREATE TABLE rls_fixture.items (id uuid PRIMARY KEY, store_id uuid NOT NULL, name text NOT NULL, published boolean NOT NULL DEFAULT false)`,
		`ALTER TABLE rls_fixture.items ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE rls_fixture.items FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY public_read ON rls_fixture.items FOR SELECT TO app_public
		   USING (platform.current_audience() = 'public' AND published)`,
		`CREATE POLICY seller_rw ON rls_fixture.items FOR ALL TO app_public
		   USING (platform.current_audience() = 'seller' AND store_id = platform.current_store_id())
		   WITH CHECK (store_id = platform.current_store_id())`,
		`GRANT USAGE ON SCHEMA rls_fixture TO app_public, app_admin`,
		`GRANT SELECT, INSERT, UPDATE ON rls_fixture.items TO app_public, app_admin`,
	}
	for _, s := range stmts {
		if _, err := mig.Exec(ctx, s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	pub := d.Pool("app_public")
	insert := func(scope db.Scope, store, name string, published bool) error {
		return db.InTx(ctx, pub, scope, db.TxOptions{}, func(tx db.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rls_fixture.items (id, store_id, name, published) VALUES ($1,$2,$3,$4)`,
				ids.New(), store, name, published)
			return err
		})
	}
	count := func(scope db.Scope) int {
		var n int
		if err := db.InTx(ctx, pub, scope, db.TxOptions{ReadOnly: true}, func(tx db.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM rls_fixture.items`).Scan(&n)
		}); err != nil {
			t.Fatal(err)
		}
		return n
	}
	sellerA := db.Scope{Audience: db.AudienceSeller, StoreID: storeA}
	sellerB := db.Scope{Audience: db.AudienceSeller, StoreID: storeB}
	if err := insert(sellerA, storeA, "a-draft", false); err != nil {
		t.Fatal(err)
	}
	if err := insert(sellerA, storeA, "a-live", true); err != nil {
		t.Fatal(err)
	}
	if err := insert(sellerB, storeB, "b-live", true); err != nil {
		t.Fatal(err)
	}
	// A seller cannot write rows for another store.
	if err := insert(sellerA, storeB, "forged", true); err == nil {
		t.Fatal("seller A inserted a row for store B")
	}
	if got := count(sellerA); got != 2 {
		t.Errorf("seller A sees %d rows, want 2 (own store only)", got)
	}
	if got := count(sellerB); got != 1 {
		t.Errorf("seller B sees %d rows, want 1", got)
	}
	if got := count(db.Scope{Audience: db.AudiencePublic}); got != 2 {
		t.Errorf("public sees %d rows, want 2 (published only)", got)
	}
	// Unknown audience: fail closed, no error (NULLIF prevents ''::uuid errors on pooled connections).
	if got := count(db.Scope{Audience: "nobody"}); got != 0 {
		t.Errorf("unknown audience sees %d rows", got)
	}
	// The context is transaction-local: a plain query on a pooled connection afterwards sees nothing.
	var n int
	if err := pub.QueryRow(ctx, `SELECT count(*) FROM rls_fixture.items`).Scan(&n); err != nil || n != 0 {
		t.Errorf("no-context query saw %d rows (err %v)", n, err)
	}
	// app_admin bypasses RLS (and audits at the application layer).
	if err := d.Pool("app_admin").QueryRow(ctx, `SELECT count(*) FROM rls_fixture.items`).Scan(&n); err != nil || n != 3 {
		t.Errorf("admin sees %d rows (err %v)", n, err)
	}
}

func TestInTxRequiresScopeAndRollsBack(t *testing.T) {
	d := testdb.New(t, true)
	ctx := context.Background()
	if err := db.InTx(ctx, d.Pool("app_public"), db.Scope{}, db.TxOptions{}, func(db.Tx) error { return nil }); !errors.Is(err, db.ErrScopeRequired) {
		t.Fatalf("want ErrScopeRequired, got %v", err)
	}
	var audience string
	err := db.InTx(ctx, d.Pool("app_public"), db.Scope{Audience: db.AudiencePublic, ActorID: "u1"}, db.TxOptions{}, func(tx db.Tx) error {
		return tx.QueryRow(ctx, `SELECT platform.current_audience()`).Scan(&audience)
	})
	if err != nil || audience != "public" {
		t.Fatalf("audience=%q err=%v", audience, err)
	}
	func() {
		defer func() { _ = recover() }()
		_ = db.InTx(ctx, d.Pool("app_public"), db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
			panic("boom")
		})
	}()
	// The pool is still usable after a panic inside a transaction.
	if err := d.Pool("app_public").Ping(ctx); err != nil {
		t.Fatal(err)
	}
}
