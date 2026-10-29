// Package jobs is the PostgreSQL-backed background job queue (ADR-007, doc 07 §6).
//
// Jobs are enqueued inside the caller's transaction (so they exist only if the business change commits),
// claimed by workers with FOR UPDATE SKIP LOCKED and a lease, retried with exponential back-off, and moved to
// state "dead" after max_attempts. A worker that dies mid-job loses its lease and the job becomes claimable again.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Options for Enqueue.
type Options struct {
	RunAt       time.Time // zero = now
	MaxAttempts int       // zero = 25
	// UniqueKey de-duplicates active jobs of the same kind (e.g. "2026-10-29" for a daily job).
	UniqueKey string
}

var kindRe = regexp.MustCompile(`^[a-z][a-z0-9_.]*$`)

// ErrDuplicate is returned when an active job with the same kind and unique key exists.
var ErrDuplicate = errors.New("jobs: duplicate active job")

// Enqueue inserts a job in the caller's transaction and returns its id.
func Enqueue(ctx context.Context, tx db.Tx, kind string, payload any, o Options) (string, error) {
	if !kindRe.MatchString(kind) {
		return "", fmt.Errorf("jobs: invalid kind %q", kind)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if o.MaxAttempts == 0 {
		o.MaxAttempts = 25
	}
	var runAt any
	if !o.RunAt.IsZero() {
		runAt = o.RunAt
	}
	id := ids.New()
	_, err = tx.Exec(ctx, `INSERT INTO platform.jobs (id, kind, payload, max_attempts, run_at, unique_key)
		VALUES ($1, $2, $3::jsonb, $4, COALESCE($5::timestamptz, now()), NULLIF($6, ''))`,
		id, kind, string(body), o.MaxAttempts, runAt, o.UniqueKey)
	if db.IsUniqueViolation(err) {
		return "", ErrDuplicate
	}
	if err != nil {
		return "", fmt.Errorf("jobs: enqueue: %w", err)
	}
	return id, nil
}

// Job is a claimed job passed to a handler.
type Job struct {
	ID      string
	Kind    string
	Payload json.RawMessage
	Attempt int // 1-based attempt number of this execution
}

// Handler executes one job. Returning an error schedules a retry (or marks the job dead).
type Handler func(ctx context.Context, j Job) error

// Permanent wraps an error that must not be retried.
func Permanent(err error) error { return permanentError{err} }

type permanentError struct{ error }

func (p permanentError) Unwrap() error { return p.error }

// Worker claims and runs jobs.
type Worker struct {
	Pool     *db.Pool
	Handlers map[string]Handler
	Lease    time.Duration // how long a claim is valid; default 5 min
	Log      *slog.Logger
	ID       string
	// MaxBackoff caps retry delays; default 1 h.
	MaxBackoff time.Duration
}

func (w *Worker) id() string {
	if w.ID == "" {
		h, _ := os.Hostname()
		w.ID = fmt.Sprintf("%s-%d", h, os.Getpid())
	}
	return w.ID
}

func (w *Worker) kinds() []string {
	out := make([]string, 0, len(w.Handlers))
	for k := range w.Handlers {
		out = append(out, k)
	}
	return out
}

// claim takes up to n available (or lease-expired) jobs of known kinds.
func (w *Worker) claim(ctx context.Context, n int) ([]Job, error) {
	lease := w.Lease
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	var out []Job
	err := db.InTx(ctx, w.Pool, db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, func(tx db.Tx) error {
		rows, err := tx.Query(ctx, `
			UPDATE platform.jobs SET state = 'running', attempt = attempt + 1,
			       locked_until = now() + $3::interval, locked_by = $4
			WHERE id IN (
			  SELECT id FROM platform.jobs
			  WHERE kind = ANY($1) AND (
			        (state = 'available' AND run_at <= now())
			     OR (state = 'running' AND locked_until < now()))
			  ORDER BY run_at
			  LIMIT $2
			  FOR UPDATE SKIP LOCKED)
			RETURNING id, kind, payload, attempt`,
			w.kinds(), n, fmt.Sprintf("%d milliseconds", lease.Milliseconds()), w.id())
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var j Job
			if err := rows.Scan(&j.ID, &j.Kind, &j.Payload, &j.Attempt); err != nil {
				return err
			}
			out = append(out, j)
		}
		return rows.Err()
	})
	return out, err
}

func (w *Worker) finish(ctx context.Context, j Job, runErr error) error {
	ctx = context.WithoutCancel(ctx)
	return db.InTx(ctx, w.Pool, db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, func(tx db.Tx) error {
		if runErr == nil {
			_, err := tx.Exec(ctx, `UPDATE platform.jobs SET state = 'succeeded', finished_at = now(),
				locked_until = NULL, last_error = NULL WHERE id = $1 AND locked_by = $2`, j.ID, w.id())
			return err
		}
		var perm permanentError
		msg := runErr.Error()
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		delay := backoff(j.Attempt, w.MaxBackoff)
		_, err := tx.Exec(ctx, `UPDATE platform.jobs SET
				state = CASE WHEN $3 OR attempt >= max_attempts THEN 'dead' ELSE 'available' END,
				finished_at = CASE WHEN $3 OR attempt >= max_attempts THEN now() END,
				run_at = now() + $4::interval, locked_until = NULL, last_error = $5
			WHERE id = $1 AND locked_by = $2`,
			j.ID, w.id(), errors.As(runErr, &perm), fmt.Sprintf("%d milliseconds", delay.Milliseconds()), msg)
		return err
	})
}

// RunOnce claims up to n jobs and executes them concurrently; it returns the number executed.
func (w *Worker) RunOnce(ctx context.Context, n int) (int, error) {
	if n <= 0 {
		n = 10
	}
	claimed, err := w.claim(ctx, n)
	if err != nil {
		return 0, fmt.Errorf("jobs: claim: %w", err)
	}
	var wg sync.WaitGroup
	errs := make([]error, len(claimed))
	for i, j := range claimed {
		wg.Add(1)
		go func(i int, j Job) {
			defer wg.Done()
			runErr := w.safeRun(ctx, j)
			if runErr != nil && w.Log != nil {
				w.Log.Warn("job failed", "job_id", j.ID, "kind", j.Kind, "attempt", j.Attempt, "err", runErr)
			}
			errs[i] = w.finish(ctx, j, runErr)
		}(i, j)
	}
	wg.Wait()
	return len(claimed), errors.Join(errs...)
}

func (w *Worker) safeRun(ctx context.Context, j Job) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	h := w.Handlers[j.Kind]
	if h == nil {
		return Permanent(fmt.Errorf("no handler for kind %q", j.Kind))
	}
	return h(ctx, j)
}

// Run processes jobs until ctx is cancelled.
func (w *Worker) Run(ctx context.Context, concurrency int, idle time.Duration) {
	if idle <= 0 {
		idle = time.Second
	}
	for {
		n, err := w.RunOnce(ctx, concurrency)
		if err != nil && ctx.Err() == nil && w.Log != nil {
			w.Log.Error("job worker", "err", err)
		}
		if n > 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(idle):
		}
	}
}

func backoff(attempt int, max time.Duration) time.Duration {
	if max <= 0 {
		max = time.Hour
	}
	d := time.Duration(attempt*attempt) * 5 * time.Second
	if d > max {
		d = max
	}
	return d
}

// Periodic enqueues a job once per period using the period start as unique key (safe with many workers).
type Periodic struct {
	Kind  string
	Every time.Duration
}

// SchedulePeriodic enqueues each periodic job for the current period if not already present.
func SchedulePeriodic(ctx context.Context, pool *db.Pool, now time.Time, ps []Periodic) error {
	return db.InTx(ctx, pool, db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, func(tx db.Tx) error {
		for _, p := range ps {
			key := now.Truncate(p.Every).UTC().Format(time.RFC3339)
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM platform.jobs WHERE kind = $1 AND unique_key = $2)`,
				p.Kind, key).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO platform.jobs (id, kind, payload, unique_key, max_attempts)
				VALUES ($1, $2, '{}', $3, 5) ON CONFLICT DO NOTHING`, ids.New(), p.Kind, key); err != nil {
				return err
			}
		}
		return nil
	})
}
