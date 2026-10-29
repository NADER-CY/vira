// Package outbox implements the transactional outbox (ADR-006, doc 07 §5).
//
// Producers call Write inside the same database transaction as their state change, so an event exists
// if and only if the change committed. The Relay (worker runtime) claims pending rows with
// FOR UPDATE SKIP LOCKED, publishes CloudEvents-compatible envelopes and marks them dispatched in the same
// transaction. If the process dies after publishing but before commit, the row is published again:
// delivery is at-least-once, and consumers de-duplicate with the inbox (package inbox).
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Event is a domain event to publish.
type Event struct {
	Module           string // e.g. "identity"
	AggregateType    string // e.g. "user"
	AggregateID      string
	AggregateVersion int64
	Type             string // vira.<module>.<event>.v<N>
	Data             any
	Traceparent      string
}

var typeRe = regexp.MustCompile(`^vira\.[a-z_]+\.[a-z_]+\.v[0-9]+$`)

// Write stores the event in the caller's transaction and returns its id.
func Write(ctx context.Context, tx db.Tx, e Event) (string, error) {
	if !typeRe.MatchString(e.Type) {
		return "", fmt.Errorf("outbox: invalid event type %q (want vira.<module>.<event>.v<N>)", e.Type)
	}
	if e.Module == "" || e.AggregateType == "" || e.AggregateID == "" {
		return "", errors.New("outbox: module, aggregate type and aggregate id are required")
	}
	payload, err := json.Marshal(e.Data)
	if err != nil {
		return "", fmt.Errorf("outbox: marshal data: %w", err)
	}
	id := ids.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO platform.outbox (event_id, module, aggregate_type, aggregate_id, aggregate_version, type, payload, traceparent)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, NULLIF($8, ''))`,
		id, e.Module, e.AggregateType, e.AggregateID, e.AggregateVersion, e.Type, string(payload), e.Traceparent)
	if err != nil {
		return "", fmt.Errorf("outbox: insert: %w", err)
	}
	return id, nil
}

// Envelope is the published message (CloudEvents 1.0 compatible, doc 07 §5.1).
type Envelope struct {
	SpecVersion      string          `json:"specversion"`
	ID               string          `json:"id"`
	Source           string          `json:"source"`
	Type             string          `json:"type"`
	Subject          string          `json:"subject"`
	Time             time.Time       `json:"time"`
	DataContentType  string          `json:"datacontenttype"`
	Traceparent      string          `json:"traceparent,omitempty"`
	AggregateID      string          `json:"aggregateid"`
	AggregateVersion int64           `json:"aggregateversion"`
	Data             json.RawMessage `json:"data"`
}

// Publisher delivers an envelope to the message bus. Subject equals the event type.
type Publisher interface {
	Publish(ctx context.Context, subject string, env Envelope, body []byte) error
}

// Relay moves pending outbox rows to the Publisher.
type Relay struct {
	Pool      *db.Pool
	Publisher Publisher
	BatchSize int
	Log       *slog.Logger
	// MaxBackoff caps the retry delay after publish failures.
	MaxBackoff time.Duration
}

// RunOnce dispatches up to BatchSize pending events and returns how many were published.
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	batch := r.BatchSize
	if batch <= 0 {
		batch = 100
	}
	published := 0
	err := db.InTx(ctx, r.Pool, db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, func(tx db.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT event_id, module, aggregate_type, aggregate_id, aggregate_version, type, payload,
			       COALESCE(traceparent, ''), created_at, attempts
			FROM platform.outbox
			WHERE dispatched_at IS NULL AND available_at <= now()
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED`, batch)
		if err != nil {
			return err
		}
		type row struct {
			env      Envelope
			attempts int
		}
		var pending []row
		for rows.Next() {
			var rw row
			var module, aggType string
			var payload []byte
			if err := rows.Scan(&rw.env.ID, &module, &aggType, &rw.env.AggregateID, &rw.env.AggregateVersion,
				&rw.env.Type, &payload, &rw.env.Traceparent, &rw.env.Time, &rw.attempts); err != nil {
				rows.Close()
				return err
			}
			rw.env.SpecVersion = "1.0"
			rw.env.Source = "vira/" + module
			rw.env.Subject = aggType + "/" + rw.env.AggregateID
			rw.env.DataContentType = "application/json"
			rw.env.Time = rw.env.Time.UTC()
			rw.env.Data = payload
			pending = append(pending, rw)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, p := range pending {
			body, err := json.Marshal(p.env)
			if err != nil {
				return err
			}
			if perr := r.Publisher.Publish(ctx, p.env.Type, p.env, body); perr != nil {
				delay := backoff(p.attempts+1, r.MaxBackoff)
				if _, err := tx.Exec(ctx, `UPDATE platform.outbox SET attempts = attempts + 1, last_error = $2,
					available_at = now() + $3::interval WHERE event_id = $1`,
					p.env.ID, truncate(perr.Error(), 500), fmt.Sprintf("%d milliseconds", delay.Milliseconds())); err != nil {
					return err
				}
				if r.Log != nil {
					r.Log.Warn("outbox publish failed", "event_id", p.env.ID, "type", p.env.Type, "attempt", p.attempts+1, "err", perr)
				}
				continue
			}
			if _, err := tx.Exec(ctx, `UPDATE platform.outbox SET dispatched_at = now(), attempts = attempts + 1, last_error = NULL
				WHERE event_id = $1`, p.env.ID); err != nil {
				return err
			}
			published++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("outbox relay: %w", err)
	}
	return published, nil
}

// Run dispatches continuously until ctx is cancelled.
func (r *Relay) Run(ctx context.Context, idle time.Duration) {
	if idle <= 0 {
		idle = 500 * time.Millisecond
	}
	for {
		n, err := r.RunOnce(ctx)
		if err != nil && ctx.Err() == nil && r.Log != nil {
			r.Log.Error("outbox relay", "err", err)
		}
		if n > 0 && err == nil {
			continue // drain quickly while there is work
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(idle):
		}
	}
}

// Lag returns the age of the oldest undispatched event (0 when none). Alert when > 60 s (doc 07 §6).
func Lag(ctx context.Context, pool *db.Pool) (time.Duration, error) {
	var secs float64
	err := pool.QueryRow(ctx, `SELECT COALESCE(EXTRACT(EPOCH FROM now() - min(created_at)), 0)
		FROM platform.outbox WHERE dispatched_at IS NULL`).Scan(&secs)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// PruneDispatched deletes events dispatched more than `olderThan` ago (doc 06 §8: 7 days), in batches.
func PruneDispatched(ctx context.Context, tx db.Tx, olderThan time.Duration, limit int) (int64, error) {
	tag, err := tx.Exec(ctx, `DELETE FROM platform.outbox WHERE event_id IN (
		SELECT event_id FROM platform.outbox WHERE dispatched_at < now() - $1::interval LIMIT $2)`,
		fmt.Sprintf("%d seconds", int64(olderThan.Seconds())), limit)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func backoff(attempt int, max time.Duration) time.Duration {
	if max <= 0 {
		max = 5 * time.Minute
	}
	d := time.Second << min(attempt, 20)
	if d > max || d <= 0 {
		d = max
	}
	return d
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
