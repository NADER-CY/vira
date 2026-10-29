// Package audit writes and reads the append-only audit log (platform.audit_events, doc 06 §8).
// Audit records are written in the same transaction as the action they describe, so an action cannot
// commit without its record.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Outcomes.
const (
	Success = "success"
	Failure = "failure"
	Denied  = "denied"
)

// Event is one audit record.
type Event struct {
	ID         string         `json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	ActorType  string         `json:"actor_type"` // admin | user | seller_staff | system | anonymous
	ActorID    string         `json:"actor_id,omitempty"`
	Action     string         `json:"action"` // e.g. admin.login, flags.update
	TargetType string         `json:"target_type,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	Outcome    string         `json:"outcome"`
	IP         string         `json:"ip,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// Record appends an event in the caller's transaction.
func Record(ctx context.Context, tx db.Tx, e Event) (string, error) {
	if e.Outcome == "" {
		e.Outcome = Success
	}
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
	meta, err := json.Marshal(e.Metadata)
	if err != nil {
		return "", err
	}
	var ip any
	if parsed := net.ParseIP(e.IP); parsed != nil {
		ip = parsed.String()
	}
	id := ids.New()
	_, err = tx.Exec(ctx, `INSERT INTO platform.audit_events
		(id, actor_type, actor_id, action, target_type, target_id, outcome, ip, user_agent, trace_id, metadata)
		VALUES ($1, $2, NULLIF($3,''), $4, NULLIF($5,''), NULLIF($6,''), $7, $8::inet, NULLIF($9,''), NULLIF($10,''), $11::jsonb)`,
		id, e.ActorType, e.ActorID, e.Action, e.TargetType, e.TargetID, e.Outcome, ip, truncate(e.UserAgent, 300), e.TraceID, string(meta))
	if err != nil {
		return "", fmt.Errorf("audit: record: %w", err)
	}
	return id, nil
}

// Query filters the audit log (admin API).
type Query struct {
	Action   string
	ActorID  string
	Before   time.Time // cursor: created_at of the last row seen
	BeforeID string
	Limit    int
}

// List returns events newest first.
func List(ctx context.Context, tx db.Tx, q Query) ([]Event, error) {
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
	var before any
	if !q.Before.IsZero() {
		before = q.Before
	}
	rows, err := tx.Query(ctx, `
		SELECT id, created_at, actor_type, COALESCE(actor_id,''), action, COALESCE(target_type,''), COALESCE(target_id,''),
		       outcome, COALESCE(host(ip),''), COALESCE(user_agent,''), COALESCE(trace_id,''), metadata
		FROM platform.audit_events
		WHERE ($1 = '' OR action = $1)
		  AND ($2 = '' OR actor_id = $2)
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3::timestamptz, $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $5`, q.Action, q.ActorID, before, nullUUID(q.BeforeID), q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var meta []byte
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.ActorType, &e.ActorID, &e.Action, &e.TargetType, &e.TargetID,
			&e.Outcome, &e.IP, &e.UserAgent, &e.TraceID, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &e.Metadata)
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullUUID(s string) any {
	if s == "" {
		return "ffffffff-ffff-ffff-ffff-ffffffffffff"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
