// Command worker runs background work: the outbox relay (→ NATS), the job queue and periodic maintenance.
package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/idempotency"
	"github.com/NADER-CY/vira/backend/internal/platform/inbox"
	"github.com/NADER-CY/vira/backend/internal/platform/jobs"
	"github.com/NADER-CY/vira/backend/internal/platform/outbox"
)

func main() {
	cfg, log := runtime.Start("worker")
	ctx, stop := runtime.SignalContext()
	defer stop()

	pool, err := runtime.OpenDB(ctx, cfg)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.NATSURL == "" {
		log.Error("VIRA_NATS_URL is required for the worker (outbox relay)")
		os.Exit(2)
	}
	nc, err := nats.Connect(cfg.NATSURL, nats.Name("vira-worker"), nats.MaxReconnects(-1))
	if err != nil {
		log.Error("nats", "err", err)
		os.Exit(1)
	}
	defer nc.Drain() //nolint:errcheck

	checks := append(runtime.DependencyChecks(cfg, pool),
		health.Check{Name: "nats", Run: func(context.Context) error {
			if !nc.IsConnected() {
				return nats.ErrConnectionClosed
			}
			return nil
		}},
		health.Check{Name: "outbox_lag", Run: func(ctx context.Context) error {
			lag, err := outbox.Lag(ctx, pool)
			if err != nil {
				return err
			}
			if lag > 60*time.Second { // doc 07 §6: alert at lag > 60 s
				return &lagError{lag}
			}
			return nil
		}},
	)

	relay := &outbox.Relay{Pool: pool, Publisher: &outbox.NATSPublisher{Conn: nc}, BatchSize: 200, Log: log}
	worker := &jobs.Worker{Pool: pool, Log: log, Handlers: maintenanceHandlers(pool)}
	periodic := []jobs.Periodic{
		{Kind: "platform.prune_outbox", Every: time.Hour},
		{Kind: "platform.prune_inbox", Every: 24 * time.Hour},
		{Kind: "platform.purge_idempotency_keys", Every: time.Hour},
		{Kind: "platform.ensure_audit_partitions", Every: 24 * time.Hour},
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); relay.Run(ctx, 250*time.Millisecond) }()
	go func() { defer wg.Done(); worker.Run(ctx, 8, time.Second) }()
	go func() {
		defer wg.Done()
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			if err := jobs.SchedulePeriodic(ctx, pool, time.Now(), periodic); err != nil && ctx.Err() == nil {
				log.Error("schedule periodic", "err", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	r := httpx.NewRouter()
	(&health.Handler{Service: "worker", Version: runtime.Version, Checks: checks}).Register(r)
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, httpx.Chain(r, httpx.RequestContext(log), httpx.Recover)); err != nil {
		log.Error("server", "err", err)
	}
	stop()
	wg.Wait()
}

type lagError struct{ lag time.Duration }

func (e *lagError) Error() string { return "outbox lag " + e.lag.Round(time.Second).String() }

func inTx(ctx context.Context, pool *db.Pool, fn func(db.Tx) error) error {
	return db.InTx(ctx, pool, db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, fn)
}

func maintenanceHandlers(pool *db.Pool) map[string]jobs.Handler {
	return map[string]jobs.Handler{
		"platform.prune_outbox": func(ctx context.Context, _ jobs.Job) error {
			return inTx(ctx, pool, func(tx db.Tx) error {
				_, err := outbox.PruneDispatched(ctx, tx, 7*24*time.Hour, 10_000)
				return err
			})
		},
		"platform.prune_inbox": func(ctx context.Context, _ jobs.Job) error {
			return inTx(ctx, pool, func(tx db.Tx) error {
				_, err := inbox.Prune(ctx, tx, 30*24*time.Hour)
				return err
			})
		},
		"platform.purge_idempotency_keys": func(ctx context.Context, _ jobs.Job) error {
			return inTx(ctx, pool, func(tx db.Tx) error {
				_, err := idempotency.Purge(ctx, tx, 10_000)
				return err
			})
		},
		"platform.ensure_audit_partitions": func(ctx context.Context, _ jobs.Job) error {
			return inTx(ctx, pool, func(tx db.Tx) error {
				_, err := tx.Exec(ctx, `SELECT platform.ensure_audit_partitions(3)`)
				return err
			})
		},
	}
}
