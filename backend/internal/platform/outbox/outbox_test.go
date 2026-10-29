package outbox_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/inbox"
	"github.com/NADER-CY/vira/backend/internal/platform/outbox"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
)

type recorder struct {
	mu     sync.Mutex
	got    []outbox.Envelope
	fail   error
	panicN int // panic after recording this many messages (simulates a crash after publish, before commit)
}

func (r *recorder) Publish(_ context.Context, subject string, env outbox.Envelope, _ []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	if subject != env.Type {
		return errors.New("subject must equal type")
	}
	r.got = append(r.got, env)
	if r.panicN > 0 && len(r.got) >= r.panicN {
		panic("simulated crash after publish")
	}
	return nil
}

func (r *recorder) ids() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.got))
	for i, e := range r.got {
		out[i] = e.ID
	}
	return out
}

func writeEvent(t *testing.T, pool *db.Pool, commit bool) string {
	t.Helper()
	var id string
	sentinel := errors.New("rollback")
	err := db.InTx(context.Background(), pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		id, err = outbox.Write(context.Background(), tx, outbox.Event{
			Module: "platform", AggregateType: "probe", AggregateID: "p-1", AggregateVersion: 1,
			Type: "vira.platform.probe_created.v1", Data: map[string]any{"n": 1},
		})
		if err != nil {
			return err
		}
		if !commit {
			return sentinel
		}
		return nil
	})
	if commit && err != nil {
		t.Fatal(err)
	}
	if !commit && !errors.Is(err, sentinel) {
		t.Fatalf("expected rollback, got %v", err)
	}
	return id
}

func pendingCount(t *testing.T, d *testdb.DB) int {
	var n int
	if err := d.Pool("superuser").QueryRow(context.Background(),
		`SELECT count(*) FROM platform.outbox WHERE dispatched_at IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRolledBackTransactionPublishesNothing(t *testing.T) {
	d := testdb.New(t, true)
	writeEvent(t, d.Pool("app_public"), false)
	if n := pendingCount(t, d); n != 0 {
		t.Fatalf("rolled-back transaction left %d events", n)
	}
}

func TestCommittedEventIsPublishedOnceRelayRuns(t *testing.T) {
	d := testdb.New(t, true)
	id := writeEvent(t, d.Pool("app_public"), true) // committed; relay not running yet ("crash before publish")
	rec := &recorder{}
	relay := &outbox.Relay{Pool: d.Pool("app_worker"), Publisher: rec}
	n, err := relay.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("relay: n=%d err=%v", n, err)
	}
	got := rec.ids()
	if len(got) != 1 || got[0] != id {
		t.Fatalf("published %v, want [%s]", got, id)
	}
	env := rec.got[0]
	if env.SpecVersion != "1.0" || env.Source != "vira/platform" || env.Subject != "probe/p-1" || string(env.Data) != `{"n": 1}` && string(env.Data) != `{"n":1}` {
		t.Fatalf("bad envelope: %+v data=%s", env, env.Data)
	}
	if n, _ := relay.RunOnce(context.Background()); n != 0 {
		t.Fatalf("dispatched event published again (%d)", n)
	}
}

// TestCrashAfterPublishBeforeCommit: the relay publishes, then the process dies before marking the row.
// The event must be published again (at-least-once) and the inbox must make the effect happen once.
func TestCrashAfterPublishBeforeCommit(t *testing.T) {
	d := testdb.New(t, true)
	id := writeEvent(t, d.Pool("app_public"), true)

	crashing := &recorder{panicN: 1}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected simulated crash")
			}
		}()
		_, _ = (&outbox.Relay{Pool: d.Pool("app_worker"), Publisher: crashing}).RunOnce(context.Background())
	}()
	if n := pendingCount(t, d); n != 1 {
		t.Fatalf("after crash the event must still be pending, got %d", n)
	}

	healthy := &recorder{}
	if n, err := (&outbox.Relay{Pool: d.Pool("app_worker"), Publisher: healthy}).RunOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("recovery relay: n=%d err=%v", n, err)
	}

	// The consumer sees the event twice (once per delivery) but applies it once.
	effects := 0
	for _, delivered := range append(crashing.ids(), healthy.ids()...) {
		if delivered != id {
			t.Fatalf("unexpected event %s", delivered)
		}
		ran, err := inbox.ProcessOnce(context.Background(), d.Pool("app_worker"), "test-consumer", delivered, func(db.Tx) error {
			effects++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		_ = ran
	}
	if effects != 1 {
		t.Fatalf("effect applied %d times, want exactly 1", effects)
	}
}

func TestPublishFailureBacksOffAndRetries(t *testing.T) {
	d := testdb.New(t, true)
	writeEvent(t, d.Pool("app_public"), true)
	failing := &recorder{fail: errors.New("bus down")}
	relay := &outbox.Relay{Pool: d.Pool("app_worker"), Publisher: failing}
	if n, err := relay.RunOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	var attempts int
	var lastErr string
	var delayed bool
	if err := d.Pool("superuser").QueryRow(context.Background(),
		`SELECT attempts, last_error, available_at > now() FROM platform.outbox`).Scan(&attempts, &lastErr, &delayed); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || lastErr != "bus down" || !delayed {
		t.Fatalf("attempts=%d last_error=%q delayed=%v", attempts, lastErr, delayed)
	}
	// Not yet available: nothing is retried immediately.
	ok := &recorder{}
	if n, _ := (&outbox.Relay{Pool: d.Pool("app_worker"), Publisher: ok}).RunOnce(context.Background()); n != 0 {
		t.Fatal("event retried before its back-off elapsed")
	}
	if _, err := d.Pool("superuser").Exec(context.Background(), `UPDATE platform.outbox SET available_at = now()`); err != nil {
		t.Fatal(err)
	}
	if n, _ := (&outbox.Relay{Pool: d.Pool("app_worker"), Publisher: ok}).RunOnce(context.Background()); n != 1 {
		t.Fatal("event not retried after back-off")
	}
}

func TestConcurrentRelaysPublishEachEventOnce(t *testing.T) {
	d := testdb.New(t, true)
	const events = 60
	for i := 0; i < events; i++ {
		writeEvent(t, d.Pool("app_public"), true)
	}
	rec := &recorder{}
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := &outbox.Relay{Pool: d.Pool("app_worker"), Publisher: rec, BatchSize: 7}
			for {
				n, err := r.RunOnce(context.Background())
				if err != nil {
					t.Error(err)
					return
				}
				if n == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()
	seen := map[string]int{}
	for _, id := range rec.ids() {
		seen[id]++
	}
	if len(seen) != events {
		t.Fatalf("published %d distinct events, want %d", len(seen), events)
	}
	for id, c := range seen {
		if c != 1 {
			t.Fatalf("event %s published %d times", id, c)
		}
	}
	lag, err := outbox.Lag(context.Background(), d.Pool("app_worker"))
	if err != nil || lag != 0 {
		t.Fatalf("lag=%v err=%v", lag, err)
	}
}

func TestInvalidEventTypeRejected(t *testing.T) {
	d := testdb.New(t, true)
	err := db.InTx(context.Background(), d.Pool("app_public"), db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		_, err := outbox.Write(context.Background(), tx, outbox.Event{Module: "x", AggregateType: "y", AggregateID: "z", Type: "Order.Created"})
		return err
	})
	if err == nil {
		t.Fatal("invalid type accepted")
	}
}

func TestPruneDispatched(t *testing.T) {
	d := testdb.New(t, true)
	writeEvent(t, d.Pool("app_public"), true)
	if _, err := d.Pool("superuser").Exec(context.Background(),
		`UPDATE platform.outbox SET dispatched_at = now() - interval '8 days'`); err != nil {
		t.Fatal(err)
	}
	var n int64
	err := db.InTx(context.Background(), d.Pool("app_worker"), db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		n, err = outbox.PruneDispatched(context.Background(), tx, 7*24*time.Hour, 1000)
		return err
	})
	if err != nil || n != 1 {
		t.Fatalf("pruned %d err=%v", n, err)
	}
}
