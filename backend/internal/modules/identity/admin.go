package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Staff operations for the Super Admin (admin realm, app_admin pool). Phone numbers are never returned in clear
// text: lookups use the blind index and lists show the masked form (doc 08).

// AdminUser is a row of the user list.
type AdminUser struct {
	ID          string     `json:"id"`
	Handle      string     `json:"handle"`
	DisplayName string     `json:"display_name"`
	PhoneMasked string     `json:"phone_masked"`
	Status      string     `json:"status"`
	IsPrivate   bool       `json:"is_private"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
	Posts       int        `json:"posts"`
	Followers   int        `json:"followers"`
	OpenReports int        `json:"open_reports"`
	Sessions    int        `json:"active_sessions"`
}

// AdminQuery filters the user list. Q matches a handle prefix, a user id, or an exact phone number.
type AdminQuery struct {
	Q        string
	Status   string
	Before   *time.Time
	BeforeID string
	Limit    int
}

// AdminService implements staff user management.
type AdminService struct {
	Pool *db.Pool
	PII  contract.PII
}

// ListUsers returns users matching q, newest first.
func (a *AdminService) ListUsers(ctx context.Context, q AdminQuery) ([]AdminUser, error) {
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
	switch q.Status {
	case "", contract.StatusActive, contract.StatusRestricted, contract.StatusSuspended, contract.StatusDeletionPending, contract.StatusDeleted:
	default:
		return nil, apperr.Validation(apperr.FieldError{Field: "status", Code: "invalid"})
	}
	var byID, byPhone any
	handle := ""
	term := strings.TrimSpace(q.Q)
	switch {
	case term == "":
	case ids.Valid(term):
		byID = term
	default:
		if phone, ok := contract.NormalizePhone(term); ok {
			byPhone = a.PII.Index(phone)
		} else {
			handle = strings.ToLower(strings.TrimPrefix(term, "@"))
			handle = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(handle)
		}
	}
	var before, beforeID any
	if q.Before != nil {
		before, beforeID = *q.Before, q.BeforeID
	}
	rows, err := a.Pool.Query(ctx, `SELECT u.id, COALESCE(p.handle, ''), COALESCE(p.display_name, ''), COALESCE(ph.phone_last4, ''),
		u.status, COALESCE(p.is_private, false), u.created_at, u.last_login_at,
		(SELECT count(*) FROM social.posts sp WHERE sp.author_id = u.id AND sp.state = 'published'),
		(SELECT count(*) FROM social.follows f WHERE f.followee_id = u.id),
		(SELECT count(*) FROM moderation.reports r WHERE r.target_owner_id = u.id AND r.state = 'open'),
		(SELECT count(*) FROM identity.sessions s WHERE s.user_id = u.id AND s.revoked_at IS NULL AND s.expires_at > now())
		FROM identity.users u
		LEFT JOIN social.profiles p ON p.user_id = u.id
		LEFT JOIN identity.phone_identities ph ON ph.user_id = u.id
		WHERE ($1 = '' OR u.status = $1)
		  AND ($2::uuid IS NULL OR u.id = $2::uuid)
		  AND ($3::bytea IS NULL OR ph.phone_bidx = $3::bytea)
		  AND ($4 = '' OR p.handle_norm LIKE $4 || '%')
		  AND ($5::timestamptz IS NULL OR (u.created_at, u.id) < ($5, $6::uuid))
		ORDER BY u.created_at DESC, u.id DESC LIMIT $7`, q.Status, byID, byPhone, handle, before, beforeID, q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminUser{}
	for rows.Next() {
		var u AdminUser
		var last4 string
		if err := rows.Scan(&u.ID, &u.Handle, &u.DisplayName, &last4, &u.Status, &u.IsPrivate, &u.CreatedAt, &u.LastLoginAt,
			&u.Posts, &u.Followers, &u.OpenReports, &u.Sessions); err != nil {
			return nil, err
		}
		if last4 != "" {
			u.PhoneMasked = contract.MaskPhone(last4)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// AdminSummary feeds the KPI cards of the users console. "Today" is the calendar day in Tehran.
type AdminSummary struct {
	Total          int `json:"total"`
	Active         int `json:"active"`
	SignupsToday   int `json:"signups_today"`
	Restricted     int `json:"restricted"`
	Suspended      int `json:"suspended"`
	ActiveSessions int `json:"active_sessions"`
}

// Summary counts accounts by status (deleted accounts excluded) and live sessions.
func (a *AdminService) Summary(ctx context.Context) (AdminSummary, error) {
	var s AdminSummary
	err := a.Pool.QueryRow(ctx, `SELECT count(*),
		count(*) FILTER (WHERE status = 'active'),
		count(*) FILTER (WHERE created_at >= (date_trunc('day', now() AT TIME ZONE 'Asia/Tehran') AT TIME ZONE 'Asia/Tehran')),
		count(*) FILTER (WHERE status = 'restricted'),
		count(*) FILTER (WHERE status = 'suspended'),
		(SELECT count(*) FROM identity.sessions WHERE revoked_at IS NULL AND expires_at > now())
		FROM identity.users WHERE status <> 'deleted'`).
		Scan(&s.Total, &s.Active, &s.SignupsToday, &s.Restricted, &s.Suspended, &s.ActiveSessions)
	return s, err
}

// StatusChange is one entry of a user's status history.
type StatusChange struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Reason    string    `json:"reason"`
	ActorType string    `json:"actor_type"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminSession is a session as shown to staff.
type AdminSession struct {
	ID         string     `json:"id"`
	Platform   string     `json:"platform"`
	Model      string     `json:"model,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt time.Time  `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	Reason     string     `json:"revoke_reason,omitempty"`
}

// AdminUserDetail is the staff view of one account.
type AdminUserDetail struct {
	AdminUser
	Bio           string         `json:"bio"`
	StatusReason  string         `json:"status_reason,omitempty"`
	IntendsToSell bool           `json:"intends_to_sell"`
	Following     int            `json:"following"`
	History       []StatusChange `json:"status_history"`
	SessionList   []AdminSession `json:"sessions"`
}

// User returns the staff view of userID.
func (a *AdminService) User(ctx context.Context, userID string) (AdminUserDetail, error) {
	if !ids.Valid(userID) {
		return AdminUserDetail{}, apperr.New(apperr.NotFound, "")
	}
	list, err := a.ListUsers(ctx, AdminQuery{Q: userID, Limit: 1})
	if err != nil {
		return AdminUserDetail{}, err
	}
	if len(list) == 0 {
		return AdminUserDetail{}, apperr.New(apperr.NotFound, "")
	}
	d := AdminUserDetail{AdminUser: list[0]}
	if err := a.Pool.QueryRow(ctx, `SELECT COALESCE(p.bio, ''), COALESCE(u.status_reason, ''), u.intends_to_sell,
		(SELECT count(*) FROM social.follows f WHERE f.follower_id = u.id)
		FROM identity.users u LEFT JOIN social.profiles p ON p.user_id = u.id WHERE u.id = $1`, userID).
		Scan(&d.Bio, &d.StatusReason, &d.IntendsToSell, &d.Following); err != nil {
		return AdminUserDetail{}, err
	}
	rows, err := a.Pool.Query(ctx, `SELECT from_status, to_status, reason, actor_type, created_at FROM identity.status_history
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return AdminUserDetail{}, err
	}
	d.History = []StatusChange{}
	for rows.Next() {
		var c StatusChange
		if err := rows.Scan(&c.From, &c.To, &c.Reason, &c.ActorType, &c.CreatedAt); err != nil {
			rows.Close()
			return AdminUserDetail{}, err
		}
		d.History = append(d.History, c)
	}
	rows.Close()
	rows, err = a.Pool.Query(ctx, `SELECT s.id, d.platform, COALESCE(d.model, ''), s.created_at, s.last_used_at, s.revoked_at,
		COALESCE(s.revoke_reason, '') FROM identity.sessions s JOIN identity.devices d ON d.id = s.device_id
		WHERE s.user_id = $1 ORDER BY s.created_at DESC LIMIT 30`, userID)
	if err != nil {
		return AdminUserDetail{}, err
	}
	defer rows.Close()
	d.SessionList = []AdminSession{}
	for rows.Next() {
		var x AdminSession
		if err := rows.Scan(&x.ID, &x.Platform, &x.Model, &x.CreatedAt, &x.LastUsedAt, &x.RevokedAt, &x.Reason); err != nil {
			return AdminUserDetail{}, err
		}
		d.SessionList = append(d.SessionList, x)
	}
	return d, rows.Err()
}

// SetStatus changes an account's status on behalf of staff (inside the caller's transaction, which also writes
// the admin audit event). Returns the previous status.
func (a *AdminService) SetStatus(ctx context.Context, tx db.Tx, userID, status, reason, adminID string) (string, error) {
	reason = strings.TrimSpace(reason)
	var fe []apperr.FieldError
	switch status {
	case contract.StatusActive, contract.StatusRestricted, contract.StatusSuspended:
	default:
		// deletion is the user's own decision; staff cannot set deletion states.
		fe = append(fe, apperr.FieldError{Field: "status", Code: "invalid"})
	}
	if l := len([]rune(reason)); l == 0 || l > 500 {
		fe = append(fe, apperr.FieldError{Field: "reason", Code: "required", Detail: "1-500 characters"})
	}
	if !ids.Valid(userID) {
		return "", apperr.New(apperr.NotFound, "")
	}
	if fe != nil {
		return "", apperr.Validation(fe...)
	}
	prev, err := contract.SetStatus(ctx, tx, userID, status, reason, "admin", adminID)
	switch {
	case errors.Is(err, contract.ErrNotFound):
		return "", apperr.New(apperr.NotFound, "")
	case errors.Is(err, contract.ErrNoChange):
		return prev, apperr.New(apperr.Conflict, "the account already has this status")
	}
	return prev, err
}

// RevokeSessions signs a user out everywhere.
func (a *AdminService) RevokeSessions(ctx context.Context, tx db.Tx, userID string) (int64, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM identity.users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, apperr.New(apperr.NotFound, "")
		}
		return 0, err
	}
	if !exists {
		return 0, apperr.New(apperr.NotFound, "")
	}
	var n int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM identity.sessions WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n, contract.RevokeAllSessions(ctx, tx, userID, "admin", "")
}
