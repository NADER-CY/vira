package jobs_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/jobs"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
)

func enqueue(t *testing.T, pool *db.Pool, kind string, o jobs.Options) (string, error) {
	t.Helper()
	var id string
	err := db.InTx(context.Background(), pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		id, err = jobs.Enqueue(context.Background(), tx, kind, map[string]string{"k": "v"}, o)
		return err
	})
	return id, err
}

func state(t *testing.T, d *testdb.DB, id string) (string, int) {
	var s string
	var a int
	if err := d.Pool("superuser").QueryRow(context.Background(), `SELECT state, attempt FROM platform.jobs WHERE id = $1`, id).Scan(&s, &a); err != nil {
		t.Fatal(err)
	}
	return s, a
}

func TestJobSucceeds(t *testing.T) {
	d := testdb.New(t, true)
	id, err := enqueue(t, d.Pool("app_public"), "test.ok", jobs.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var calls int32
	w := &jobs.Worker{Pool: d.Pool("app_worker"), Handlers: map[string]jobs.Handler{
		"test.ok": func(_ context.Context, j jobs.Job) error {
			atomic.AddInt32(&calls, 1)
			if string(j.Payload) != `{"k": "v"}` && string(j.Payload) != `{"k":"v"}` {
				return errors.New("bad payload " + string(j.Payload))
			}
			return nil
		}}}
	if n, err := w.RunOnce(context.Background(), 5); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if s, _ := state(t, d, id); s != "succeeded" || calls != 1 {
		t.Fatalf("state=%s calls=%d", s, calls)
	}
}

func TestRetryThenDead(t *testing.T) {
	d := testdb.New(t, true)
	id, _ := enqueue(t, d.Pool("app_public"), "test.fail", jobs.Options{MaxAttempts: 2})
	w := &jobs.Worker{Pool: d.Pool("app_worker"), Handlers: map[string]jobs.Handler{
		"test.fail": func(context.Context, jobs.Job) error { return errors.New("boom") }}}
	w.RunOnce(context.Background(), 1) //nolint:errcheck
	if s, a := state(t, d, id); s != "available" || a != 1 {
		t.Fatalf("after first failure: state=%s attempt=%d", s, a)
	}
	// Skip the back-off.
	d.Pool("superuser").Exec(context.Background(), `UPDATE platform.jobs SET run_at = now()`) //nolint:errcheck
	w.RunOnce(context.Background(), 1)                                                        //nolint:errcheck
	if s, a := state(t, d, id); s != "dead" || a != 2 {
		t.Fatalf("after max attempts: state=%s attempt=%d", s, a)
	}
}

func TestPermanentErrorAndPanicDoNotLoop(t *testing.T) {
	d := testdb.New(t, true)
	perm, _ := enqueue(t, d.Pool("app_public"), "test.perm", jobs.Options{})
	pan, _ := enqueue(t, d.Pool("app_public"), "test.panic", jobs.Options{MaxAttempts: 1})
	w := &jobs.Worker{Pool: d.Pool("app_worker"), Handlers: map[string]jobs.Handler{
		"test.perm":  func(context.Context, jobs.Job) error { return jobs.Permanent(errors.New("invalid input")) },
		"test.panic": func(context.Context, jobs.Job) error { panic("bug") },
	}}
	w.RunOnce(context.Background(), 5) //nolint:errcheck
	if s, _ := state(t, d, perm); s != "dead" {
		t.Fatalf("permanent: state=%s", s)
	}
	if s, _ := state(t, d, pan); s != "dead" {
		t.Fatalf("panic: state=%s", s)
	}
}

func TestExpiredLeaseIsReclaimed(t *testing.T) {
	d := testdb.New(t, true)
	id, _ := enqueue(t, d.Pool("app_public"), "test.lease", jobs.Options{})
	// Simulate a worker that claimed the job and died.
	d.Pool("superuser").Exec(context.Background(), //nolint:errcheck
		`UPDATE platform.jobs SET state='running', attempt=1, locked_by='dead-worker', locked_until=now() - interval '1 second'`)
	w := &jobs.Worker{Pool: d.Pool("app_worker"), ID: "w2", Handlers: map[string]jobs.Handler{
		"test.lease": func(context.Context, jobs.Job) error { return nil }}}
	if n, err := w.RunOnce(context.Background(), 1); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if s, a := state(t, d, id); s != "succeeded" || a != 2 {
		t.Fatalf("state=%s attempt=%d", s, a)
	}
}

func TestUniqueKeyAndScheduledJobs(t *testing.T) {
	d := testdb.New(t, true)
	if _, err := enqueue(t, d.Pool("app_public"), "test.unique", jobs.Options{UniqueKey: "day-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := enqueue(t, d.Pool("app_public"), "test.unique", jobs.Options{UniqueKey: "day-1"}); !errors.Is(err, jobs.ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	future, _ := enqueue(t, d.Pool("app_public"), "test.later", jobs.Options{RunAt: time.Now().Add(time.Hour)})
	w := &jobs.Worker{Pool: d.Pool("app_worker"), Handlers: map[string]jobs.Handler{
		"test.later": func(context.Context, jobs.Job) error { return nil }}}
	if n, _ := w.RunOnce(context.Background(), 5); n != 0 {
		t.Fatal("future job ran early")
	}
	if s, _ := state(t, d, future); s != "available" {
		t.Fatalf("state=%s", s)
	}
}

func TestSchedulePeriodicIsIdempotent(t *testing.T) {
	d := testdb.New(t, true)
	now := time.Now()
	ps := []jobs.Periodic{{Kind: "platform.prune_outbox", Every: time.Hour}}
	for i := 0; i < 3; i++ {
		if err := jobs.SchedulePeriodic(context.Background(), d.Pool("app_worker"), now, ps); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	d.Pool("superuser").QueryRow(context.Background(), `SELECT count(*) FROM platform.jobs WHERE kind='platform.prune_outbox'`).Scan(&n) //nolint:errcheck
	if n != 1 {
		t.Fatalf("scheduled %d times, want 1", n)
	}
}
