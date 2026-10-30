// Package contract is the identity module's public surface for other modules: account status changes (used by
// moderation and the admin realm) and phone helpers. Functions take the caller's transaction (doc 06 §5).
package contract

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	socialc "github.com/NADER-CY/vira/backend/internal/modules/social/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/secure"
)

// Account statuses.
const (
	StatusActive          = "active"
	StatusRestricted      = "restricted" // can sign in and read; cannot post, comment or follow
	StatusSuspended       = "suspended"  // cannot sign in; content hidden
	StatusDeletionPending = "deletion_pending"
	StatusDeleted         = "deleted"
)

// ErrNotFound is returned for unknown users.
var ErrNotFound = errors.New("identity: user not found")

// ErrNoChange is returned when the account already has the requested status.
var ErrNoChange = errors.New("identity: status unchanged")

// SetStatus changes an account's status, records the history, mirrors it to the social profile and revokes
// sessions when the account may no longer sign in. Returns the previous status.
func SetStatus(ctx context.Context, tx db.Tx, userID, status, reason, actorType, actorID string) (string, error) {
	switch status {
	case StatusActive, StatusRestricted, StatusSuspended, StatusDeletionPending, StatusDeleted:
	default:
		return "", fmt.Errorf("identity: invalid status %q", status)
	}
	var prev string
	err := tx.QueryRow(ctx, `SELECT status FROM identity.users WHERE id = $1 FOR UPDATE`, userID).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if prev == status {
		return prev, ErrNoChange
	}
	if _, err := tx.Exec(ctx, `UPDATE identity.users SET status = $2, status_reason = $3, status_changed_at = now(),
		deletion_effective_at = CASE WHEN $2 = 'deletion_pending' THEN now() + interval '30 days' ELSE NULL END
		WHERE id = $1`, userID, status, reason); err != nil {
		return "", err
	}
	var actor any
	if actorID != "" {
		actor = actorID
	}
	if _, err := tx.Exec(ctx, `INSERT INTO identity.status_history (id, user_id, from_status, to_status, reason, actor_type, actor_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, ids.New(), userID, prev, status, reason, actorType, actor); err != nil {
		return "", err
	}
	if err := socialc.SetAccountStatus(ctx, tx, userID, status); err != nil {
		return "", err
	}
	if status == StatusSuspended || status == StatusDeleted || status == StatusDeletionPending {
		reasonCode := "account_status"
		if status == StatusDeletionPending {
			reasonCode = "deletion"
		}
		if err := RevokeAllSessions(ctx, tx, userID, reasonCode, ""); err != nil {
			return "", err
		}
	}
	return prev, nil
}

// RevokeAllSessions revokes every active session of a user except keepSessionID (may be empty).
func RevokeAllSessions(ctx context.Context, tx db.Tx, userID, reason, keepSessionID string) error {
	var keep any
	if keepSessionID != "" {
		keep = keepSessionID
	}
	if _, err := tx.Exec(ctx, `UPDATE identity.refresh_tokens rt SET state = 'revoked'
		FROM identity.sessions s WHERE rt.session_id = s.id AND s.user_id = $1 AND rt.state = 'active'
		AND ($2::uuid IS NULL OR s.id <> $2::uuid)`, userID, keep); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE identity.sessions SET revoked_at = now(), revoke_reason = $2
		WHERE user_id = $1 AND revoked_at IS NULL AND ($3::uuid IS NULL OR id <> $3::uuid)`, userID, reason, keep)
	return err
}

// PII holds the keys protecting phone numbers (doc 08): encryption and blind index.
type PII struct {
	EncKey   []byte
	IndexKey []byte
}

// NormalizePhone parses an Iranian mobile number written with Persian, Arabic or ASCII digits, with or without
// +98/0098/0, and returns it as 09xxxxxxxxx. ok=false when it is not a mobile number.
func NormalizePhone(in string) (string, bool) {
	var b strings.Builder
	for _, r := range in {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r == '+':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return "", false
		}
	}
	s := b.String()
	switch {
	case strings.HasPrefix(s, "+98"):
		s = "0" + s[3:]
	case strings.HasPrefix(s, "0098"):
		s = "0" + s[4:]
	case strings.HasPrefix(s, "9") && len(s) == 10:
		s = "0" + s
	}
	if len(s) != 11 || !strings.HasPrefix(s, "09") || strings.Contains(s, "+") {
		return "", false
	}
	return s, true
}

// Index returns the blind index of a normalized phone number.
func (p PII) Index(phone string) []byte {
	m := hmac.New(sha256.New, p.IndexKey)
	m.Write([]byte("phone:" + phone))
	return m.Sum(nil)
}

// Seal encrypts a normalized phone number.
func (p PII) Seal(phone string) ([]byte, error) {
	return secure.Seal(p.EncKey, []byte(phone), []byte("phone"))
}

// Open decrypts a phone number.
func (p PII) Open(sealed []byte) (string, error) {
	b, err := secure.Open(p.EncKey, sealed, []byte("phone"))
	return string(b), err
}

// MaskPhone renders a phone for display from its last four digits: 0912***4567 style ("09*****4567").
func MaskPhone(last4 string) string { return "09*****" + last4 }

type statusKey struct{}

// WithAccountStatus records the signed-in account's status in the request context (set by the session check).
func WithAccountStatus(ctx context.Context, status string) context.Context {
	return context.WithValue(ctx, statusKey{}, status)
}

// AccountStatusFrom returns the signed-in account's status ("" when unauthenticated).
func AccountStatusFrom(ctx context.Context) string {
	s, _ := ctx.Value(statusKey{}).(string)
	return s
}

// CanWrite reports whether the signed-in account may create content or relationships (restricted accounts
// can read only).
func CanWrite(ctx context.Context) bool { return AccountStatusFrom(ctx) == StatusActive }
