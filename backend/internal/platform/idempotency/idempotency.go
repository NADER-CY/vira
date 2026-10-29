// Package idempotency de-duplicates client retries of unsafe requests (doc 06 §8, doc 07 §1, audit F-23).
//
//	same key + same request, completed      → the stored response is replayed (Idempotent-Replayed: true)
//	same key + different request            → 409 idempotency_conflict
//	same key while the first is in progress → 409 request_in_progress + Retry-After
//	first attempt failed transiently (5xx, 408, 429) → the key is reusable; failures are never stored as final
package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/logx"
)

// Header is the request header carrying the key.
const Header = "Idempotency-Key"

// Middleware configuration.
type Middleware struct {
	Pool *db.Pool
	// Principal returns the authenticated principal id for the request ("" = anonymous).
	Principal func(r *http.Request) string
	// Required makes the header mandatory (doc 07 §1: orders, payments, refunds, payouts, messages, posts).
	Required bool
	Lease    time.Duration // default 30 s
	TTL      time.Duration // default 24 h (payments: 30 days)
}

// MaxStoredBody bounds stored response bodies.
const MaxStoredBody = 256 << 10

type captured struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (c *captured) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *captured) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	if c.buf.Len() < MaxStoredBody {
		c.buf.Write(b)
	}
	return c.ResponseWriter.Write(b)
}

func retryable(status int) bool {
	return status >= 500 || status == http.StatusRequestTimeout || status == http.StatusTooManyRequests
}

// Wrap returns the idempotency middleware.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	lease, ttl := m.Lease, m.TTL
	if lease == 0 {
		lease = 30 * time.Second
	}
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(Header)
		if key == "" {
			if m.Required {
				httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: Header, Code: "required"}))
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if len(key) < 8 || len(key) > 128 {
			httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: Header, Code: "invalid_length"}))
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, httpx.MaxBodyBytes+1))
		if err != nil {
			httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "body", Code: "unreadable"}))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		principal := "anonymous"
		if m.Principal != nil {
			if p := m.Principal(r); p != "" {
				principal = p
			}
		}
		h := sha256.New()
		fmt.Fprintf(h, "%s\n%s\n", r.Method, r.URL.Path)
		h.Write(body)
		hash := hex.EncodeToString(h.Sum(nil))

		decision, stored, err := m.claim(r.Context(), principal, key, hash, r, lease, ttl)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		switch decision {
		case replay:
			for k, vs := range stored.headers {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("Idempotent-Replayed", "true")
			w.WriteHeader(stored.status)
			_, _ = w.Write(stored.body)
			return
		case inProgress:
			w.Header().Set("Retry-After", strconv.Itoa(int(lease.Seconds())))
			httpx.WriteError(w, r, apperr.New(apperr.RequestInProgress, ""))
			return
		case mismatch:
			httpx.WriteError(w, r, apperr.New(apperr.IdempotencyConflict, ""))
			return
		}

		rec := &captured{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		m.complete(context.WithoutCancel(r.Context()), principal, key, rec)
	})
}

type decision int

const (
	execute decision = iota
	replay
	inProgress
	mismatch
)

type storedResponse struct {
	status  int
	headers http.Header
	body    []byte
}

func (m *Middleware) claim(ctx context.Context, principal, key, hash string, r *http.Request, lease, ttl time.Duration) (decision, storedResponse, error) {
	var d decision
	var s storedResponse
	err := db.InTx(ctx, m.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: principal}, db.TxOptions{}, func(tx db.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO platform.idempotency_keys
			(principal_id, key, request_hash, method, path, state, lease_until, expires_at)
			VALUES ($1, $2, $3, $4, $5, 'in_progress', now() + $6::interval, now() + $7::interval)
			ON CONFLICT (principal_id, key) DO NOTHING`,
			principal, key, hash, r.Method, r.URL.Path, interval(lease), interval(ttl))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			d = execute
			return nil
		}
		var state, storedHash string
		var leaseValid, expired bool
		var status *int
		var headers, body []byte
		err = tx.QueryRow(ctx, `SELECT state, request_hash, lease_until > now(), expires_at < now(),
				response_status, response_headers, response_body
			FROM platform.idempotency_keys WHERE principal_id = $1 AND key = $2 FOR UPDATE`, principal, key).
			Scan(&state, &storedHash, &leaseValid, &expired, &status, &headers, &body)
		if err != nil {
			return err
		}
		takeOver := func() error {
			_, err := tx.Exec(ctx, `UPDATE platform.idempotency_keys SET state = 'in_progress', request_hash = $3,
				lease_until = now() + $4::interval, expires_at = now() + $5::interval,
				response_status = NULL, response_headers = NULL, response_body = NULL
				WHERE principal_id = $1 AND key = $2`, principal, key, hash, interval(lease), interval(ttl))
			d = execute
			return err
		}
		if expired {
			return takeOver()
		}
		if storedHash != hash {
			d = mismatch
			return nil
		}
		switch {
		case state == "completed" && status != nil:
			d = replay
			s.status, s.body = *status, body
			s.headers = http.Header{}
			_ = json.Unmarshal(headers, &s.headers)
			return nil
		case state == "in_progress" && leaseValid:
			d = inProgress
			return nil
		default: // failed_retryable, or in_progress with an expired lease (the first attempt died)
			return takeOver()
		}
	})
	return d, s, err
}

// replayed headers: only content headers and Location are stored.
var keptHeaders = []string{"Content-Type", "Location", "ETag"}

func (m *Middleware) complete(ctx context.Context, principal, key string, rec *captured) {
	state := "completed"
	if retryable(rec.status) || rec.buf.Len() >= MaxStoredBody {
		state = "failed_retryable"
	}
	hdr := http.Header{}
	for _, k := range keptHeaders {
		if v := rec.Header().Get(k); v != "" {
			hdr.Set(k, v)
		}
	}
	hj, _ := json.Marshal(hdr)
	err := db.InTx(ctx, m.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: principal}, db.TxOptions{}, func(tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE platform.idempotency_keys SET state = $3, response_status = $4,
			response_headers = $5, response_body = $6, lease_until = now()
			WHERE principal_id = $1 AND key = $2`, principal, key, state, rec.status, hj, rec.buf.Bytes())
		return err
	})
	if err != nil {
		// The lease expires on its own and a retry re-executes; record the failure for operators.
		logx.From(ctx).Error("idempotency: store response", "err", err)
	}
}

// Purge deletes expired keys (worker job).
func Purge(ctx context.Context, tx db.Tx, limit int) (int64, error) {
	tag, err := tx.Exec(ctx, `DELETE FROM platform.idempotency_keys WHERE (principal_id, key) IN (
		SELECT principal_id, key FROM platform.idempotency_keys WHERE expires_at < now() LIMIT $1)`, limit)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func interval(d time.Duration) string { return fmt.Sprintf("%d milliseconds", d.Milliseconds()) }
