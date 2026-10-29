// Package flags provides feature flags and remote configuration (doc 06 §8).
//
// Regulated capabilities are locked off twice: by the database constraint regulated_flags_locked_off and by
// this package, which reports every regulated flag as disabled regardless of storage. Enabling one requires a
// reviewed migration and code change recording the approved gate (FG-1b, G1, G2, G3) — never an admin action.
package flags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Regulated lists the flags that must stay disabled until their gate (doc 20 §2). Compiled into the binary.
var Regulated = map[string]string{
	"payments.real_money": "FG-1b",
	"settlement.live":     "FG-1b",
	"payouts.withdrawals": "FG-1b",
	"wallet.topup":        "G1",
	"usdt.deposits":       "G2",
	"usdt.withdrawals":    "G2",
	"usdt.conversion":     "G2",
	"credit.bnpl":         "G3",
}

// Well-known non-regulated flags.
const (
	MaintenanceMode = "app.maintenance_mode"
	SellerWebPortal = "seller.web_portal"
)

// Flag is a stored feature flag.
type Flag struct {
	Key         string    `json:"key"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Regulated   bool      `json:"regulated"`
	Gate        string    `json:"gate,omitempty"`
	Audiences   []string  `json:"audiences"`
	Version     int       `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
}

// List returns all flags (regulated flags always reported disabled).
func List(ctx context.Context, tx db.Tx) ([]Flag, error) {
	rows, err := tx.Query(ctx, `SELECT key, description, enabled, regulated, COALESCE(gate,''), audiences, version,
		updated_at, COALESCE(updated_by,'') FROM platform.feature_flags ORDER BY regulated DESC, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.Key, &f.Description, &f.Enabled, &f.Regulated, &f.Gate, &f.Audiences, &f.Version,
			&f.UpdatedAt, &f.UpdatedBy); err != nil {
			return nil, err
		}
		if _, reg := Regulated[f.Key]; reg || f.Regulated {
			f.Enabled, f.Regulated = false, true
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Get returns one flag.
func Get(ctx context.Context, tx db.Tx, key string) (Flag, error) {
	all, err := List(ctx, tx)
	if err != nil {
		return Flag{}, err
	}
	for _, f := range all {
		if f.Key == key {
			return f, nil
		}
	}
	return Flag{}, apperr.New(apperr.NotFound, "flag "+key)
}

// SetRequest changes a non-regulated flag. Version implements optimistic concurrency (If-Match).
type SetRequest struct {
	Key     string
	Enabled bool
	Version int
	Reason  string
	ActorID string
	IP      string
	UA      string
	TraceID string
}

// Set updates a non-regulated flag, records the change and an audit event in one transaction.
func Set(ctx context.Context, tx db.Tx, r SetRequest) (Flag, error) {
	if gate, reg := Regulated[r.Key]; reg {
		// The caller records the denied attempt in its own transaction (this one rolls back).
		e := apperr.New(apperr.FeatureLocked, "regulated capability; requires gate "+gate)
		e.Extra = map[string]any{"gate": gate}
		return Flag{}, e
	}
	if len(r.Reason) < 3 {
		return Flag{}, apperr.Validation(apperr.FieldError{Field: "reason", Code: "required"})
	}
	cur, err := Get(ctx, tx, r.Key)
	if err != nil {
		return Flag{}, err
	}
	if cur.Regulated {
		return Flag{}, apperr.New(apperr.FeatureLocked, "regulated capability")
	}
	if cur.Version != r.Version {
		return Flag{}, apperr.New(apperr.PreconditionFailed, fmt.Sprintf("current version is %d", cur.Version))
	}
	tag, err := tx.Exec(ctx, `UPDATE platform.feature_flags SET enabled = $2, version = version + 1, updated_at = now(),
		updated_by = $3 WHERE key = $1 AND version = $4 AND NOT regulated`, r.Key, r.Enabled, r.ActorID, r.Version)
	if db.IsCheckViolation(err) {
		return Flag{}, apperr.New(apperr.FeatureLocked, "regulated capability")
	}
	if err != nil {
		return Flag{}, err
	}
	if tag.RowsAffected() != 1 {
		return Flag{}, apperr.New(apperr.PreconditionFailed, "flag changed concurrently")
	}
	if _, err := tx.Exec(ctx, `INSERT INTO platform.feature_flag_changes (id, flag_key, old_enabled, new_enabled, version, reason, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, ids.New(), r.Key, cur.Enabled, r.Enabled, cur.Version+1, r.Reason, r.ActorID); err != nil {
		return Flag{}, err
	}
	if _, err := audit.Record(ctx, tx, audit.Event{ActorType: "admin", ActorID: r.ActorID, Action: "flags.update",
		TargetType: "feature_flag", TargetID: r.Key, IP: r.IP, UserAgent: r.UA, TraceID: r.TraceID,
		Metadata: map[string]any{"from": cur.Enabled, "to": r.Enabled, "reason": r.Reason, "version": cur.Version + 1}}); err != nil {
		return Flag{}, err
	}
	return Get(ctx, tx, r.Key)
}

// Reader is a cached, read-only view of flags and public remote config for request paths.
type Reader struct {
	Pool *db.Pool
	TTL  time.Duration

	mu      sync.Mutex
	loaded  time.Time
	enabled map[string]bool
	config  map[string]json.RawMessage
}

func (r *Reader) refresh(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ttl := r.TTL
	if ttl == 0 {
		ttl = 15 * time.Second
	}
	if time.Since(r.loaded) < ttl && r.enabled != nil {
		return nil
	}
	en := map[string]bool{}
	cfg := map[string]json.RawMessage{}
	err := db.InTx(ctx, r.Pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{ReadOnly: true}, func(tx db.Tx) error {
		fl, err := List(ctx, tx)
		if err != nil {
			return err
		}
		for _, f := range fl {
			en[f.Key] = f.Enabled && !f.Regulated
		}
		rows, err := tx.Query(ctx, `SELECT key, value FROM platform.remote_config WHERE public`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var v []byte
			if err := rows.Scan(&k, &v); err != nil {
				return err
			}
			cfg[k] = v
		}
		return rows.Err()
	})
	if err != nil {
		return err
	}
	r.enabled, r.config, r.loaded = en, cfg, time.Now()
	return nil
}

// Enabled reports whether key is on. Regulated flags are always off; unknown flags are off (fail closed).
func (r *Reader) Enabled(ctx context.Context, key string) (bool, error) {
	if _, reg := Regulated[key]; reg {
		return false, nil
	}
	if err := r.refresh(ctx); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enabled[key], nil
}

// Snapshot returns all flag states and public remote config (for GET /v1/config).
func (r *Reader) Snapshot(ctx context.Context) (map[string]bool, map[string]json.RawMessage, error) {
	if err := r.refresh(ctx); err != nil {
		return nil, nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	en := make(map[string]bool, len(r.enabled))
	for k, v := range r.enabled {
		en[k] = v
	}
	cfg := make(map[string]json.RawMessage, len(r.config))
	for k, v := range r.config {
		cfg[k] = v
	}
	return en, cfg, nil
}

// ConfigString returns a public remote-config string value.
func (r *Reader) ConfigString(ctx context.Context, key string) (string, error) {
	_, cfg, err := r.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	raw, ok := cfg[key]
	if !ok {
		return "", errors.New("flags: config key not found: " + key)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}

// Invalidate forces the next read to reload.
func (r *Reader) Invalidate() {
	r.mu.Lock()
	r.loaded = time.Time{}
	r.mu.Unlock()
}
