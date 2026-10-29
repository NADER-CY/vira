// Package adminauth is the Super Admin realm (ADR-016): separate accounts (admin users are not platform
// users), password + mandatory TOTP, opaque session tokens stored as SHA-256 hashes, sessions of at most
// 8 hours with a 15-minute idle timeout, account lockout, RBAC permissions and an audit record for every
// authentication event.
package adminauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/secure"
)

// Policy constants.
const (
	MaxFailedLogins = 5
	LockoutDuration = 15 * time.Minute
	MFAStepTTL      = 5 * time.Minute
	touchInterval   = 30 * time.Second
	totpIssuer      = "VIRA Admin"
)

// Service implements the admin realm.
type Service struct {
	Pool        *db.Pool
	TOTPKey     []byte // AES-256 key for TOTP secrets at rest
	SessionTTL  time.Duration
	IdleTimeout time.Duration
	Now         func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Admin is the authenticated admin principal.
type Admin struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	SessionID   string    `json:"-"`
	ExpiresAt   time.Time `json:"session_expires_at"`
}

// Can reports whether the admin holds permission p.
func (a Admin) Can(p string) bool {
	for _, x := range a.Permissions {
		if x == p {
			return true
		}
	}
	return false
}

// RequestMeta carries client details for audit records.
type RequestMeta struct {
	IP      string
	UA      string
	TraceID string
}

// Enrollment is returned once when an admin is created; the secret is never shown again.
type Enrollment struct {
	AdminID    string `json:"admin_id"`
	TOTPSecret string `json:"totp_secret"`
	TOTPURI    string `json:"totp_uri"`
}

// CreateAdmin creates an admin account with a fresh TOTP secret and role bindings.
func (s *Service) CreateAdmin(ctx context.Context, email, name, password string, roles []string, createdBy string) (Enrollment, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") {
		return Enrollment{}, apperr.Validation(apperr.FieldError{Field: "email", Code: "invalid"})
	}
	hash, err := secure.HashPassword(password)
	if err != nil {
		return Enrollment{}, apperr.Validation(apperr.FieldError{Field: "password", Code: "too_short", Detail: "minimum 12 characters"})
	}
	if len(roles) == 0 {
		return Enrollment{}, apperr.Validation(apperr.FieldError{Field: "roles", Code: "required"})
	}
	id := ids.New()
	secret := secure.NewTOTPSecret()
	sealed, err := secure.Seal(s.TOTPKey, []byte(secret), []byte("admin:"+id))
	if err != nil {
		return Enrollment{}, err
	}
	err = db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: createdBy}, db.TxOptions{}, func(tx db.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO authz.admin_users (id, email, display_name, password_hash, totp_secret_enc)
			VALUES ($1, $2, $3, $4, $5)`, id, email, name, hash, sealed); err != nil {
			if db.IsUniqueViolation(err) {
				return apperr.New(apperr.Conflict, "email already registered")
			}
			return err
		}
		for _, r := range roles {
			if _, err := tx.Exec(ctx, `INSERT INTO authz.admin_role_bindings (admin_id, role_key, granted_by) VALUES ($1, $2, $3)`,
				id, r, createdBy); err != nil {
				return apperr.Validation(apperr.FieldError{Field: "roles", Code: "unknown_role", Detail: r})
			}
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "system", ActorID: createdBy, Action: "admin.create",
			TargetType: "admin_user", TargetID: id, Metadata: map[string]any{"roles": roles}})
		return err
	})
	if err != nil {
		return Enrollment{}, err
	}
	return Enrollment{AdminID: id, TOTPSecret: secret, TOTPURI: secure.TOTPURI(totpIssuer, email, secret)}, nil
}

// recordFailure writes an audit event in its own transaction (the failing transaction rolls back).
func (s *Service) recordFailure(ctx context.Context, action, adminID string, m RequestMeta, outcome string, meta map[string]any) {
	_ = db.InTx(context.WithoutCancel(ctx), s.Pool, db.Scope{Audience: db.AudienceAdmin}, db.TxOptions{}, func(tx db.Tx) error {
		actorType := "admin"
		if adminID == "" {
			actorType = "anonymous"
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: actorType, ActorID: adminID, Action: action, Outcome: outcome,
			IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID, Metadata: meta})
		return err
	})
}

var errBadCredentials = apperr.New(apperr.Unauthenticated, "invalid credentials")

// Login checks email and password and returns a short-lived MFA token (stage "mfa").
func (s *Service) Login(ctx context.Context, email, password string, m RequestMeta) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var id, hash, status string
	var locked bool
	err := s.Pool.QueryRow(ctx, `SELECT id, password_hash, status, COALESCE(locked_until > now(), false)
		FROM authz.admin_users WHERE email = $1`, email).Scan(&id, &hash, &status, &locked)
	if errors.Is(err, db.ErrNoRows) {
		secure.DummyVerify(password)
		s.recordFailure(ctx, "admin.login", "", m, audit.Failure, map[string]any{"reason": "unknown_account"})
		return "", errBadCredentials
	}
	if err != nil {
		return "", err
	}
	if status != "active" || locked {
		secure.DummyVerify(password)
		s.recordFailure(ctx, "admin.login", id, m, audit.Denied, map[string]any{"reason": "locked_or_disabled"})
		return "", apperr.New(apperr.AccountRestricted, "")
	}
	if !secure.VerifyPassword(hash, password) {
		s.registerFailure(ctx, id)
		s.recordFailure(ctx, "admin.login", id, m, audit.Failure, map[string]any{"reason": "bad_password"})
		return "", errBadCredentials
	}
	token, th := secure.NewToken()
	err = db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: id}, db.TxOptions{}, func(tx db.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO authz.admin_sessions (id, admin_id, token_hash, stage, expires_at, ip, user_agent)
			VALUES ($1, $2, $3, 'mfa', $4, $5::inet, $6)`, ids.New(), id, th, s.now().Add(MFAStepTTL), nullIP(m.IP), m.UA)
		if err != nil {
			return err
		}
		_, err = audit.Record(ctx, tx, audit.Event{ActorType: "admin", ActorID: id, Action: "admin.login.password",
			IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID})
		return err
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) registerFailure(ctx context.Context, adminID string) {
	_ = db.InTx(context.WithoutCancel(ctx), s.Pool, db.Scope{Audience: db.AudienceAdmin}, db.TxOptions{}, func(tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE authz.admin_users SET failed_logins = failed_logins + 1,
			locked_until = CASE WHEN failed_logins + 1 >= $2 THEN now() + $3::interval ELSE locked_until END,
			updated_at = now() WHERE id = $1`, adminID, MaxFailedLogins, fmt.Sprintf("%d seconds", int(LockoutDuration.Seconds())))
		return err
	})
}

// VerifyMFA exchanges an MFA token and a TOTP code for an active session token.
func (s *Service) VerifyMFA(ctx context.Context, mfaToken, code string, m RequestMeta) (string, time.Time, error) {
	var sessionID, adminID string
	var sealed []byte
	var lastStep int64
	err := s.Pool.QueryRow(ctx, `SELECT s.id, s.admin_id, u.totp_secret_enc, u.totp_last_step
		FROM authz.admin_sessions s JOIN authz.admin_users u ON u.id = s.admin_id
		WHERE s.token_hash = $1 AND s.stage = 'mfa' AND s.revoked_at IS NULL AND s.expires_at > now()
		  AND u.status = 'active' AND (u.locked_until IS NULL OR u.locked_until <= now())`,
		secure.HashToken(mfaToken)).Scan(&sessionID, &adminID, &sealed, &lastStep)
	if errors.Is(err, db.ErrNoRows) {
		s.recordFailure(ctx, "admin.login.mfa", "", m, audit.Failure, map[string]any{"reason": "unknown_or_expired_mfa_token"})
		return "", time.Time{}, apperr.New(apperr.Unauthenticated, "")
	}
	if err != nil {
		return "", time.Time{}, err
	}
	secret, err := secure.Open(s.TOTPKey, sealed, []byte("admin:"+adminID))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("adminauth: decrypt totp secret: %w", err)
	}
	step, ok := secure.VerifyTOTP(string(secret), code, s.now(), lastStep)
	if !ok {
		s.registerFailure(ctx, adminID)
		s.recordFailure(ctx, "admin.login.mfa", adminID, m, audit.Failure, map[string]any{"reason": "bad_totp"})
		return "", time.Time{}, apperr.New(apperr.MFAInvalid, "")
	}
	token, th := secure.NewToken()
	expires := s.now().Add(s.SessionTTL)
	err = db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: adminID}, db.TxOptions{}, func(tx db.Tx) error {
		// Single-use step-up token: rotate the token and promote the session in one statement.
		tag, err := tx.Exec(ctx, `UPDATE authz.admin_sessions SET stage = 'active', token_hash = $2, expires_at = $3,
			last_seen_at = now() WHERE id = $1 AND stage = 'mfa'`, sessionID, th, expires)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return apperr.New(apperr.Unauthenticated, "")
		}
		tag, err = tx.Exec(ctx, `UPDATE authz.admin_users SET totp_last_step = $2, failed_logins = 0, locked_until = NULL,
			last_login_at = now(), updated_at = now() WHERE id = $1 AND totp_last_step < $2`, adminID, step)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 { // concurrent use of the same code
			return apperr.New(apperr.MFAInvalid, "")
		}
		_, err = audit.Record(ctx, tx, audit.Event{ActorType: "admin", ActorID: adminID, Action: "admin.login",
			TargetType: "admin_session", TargetID: sessionID, IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID})
		return err
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// Authenticate resolves an active session token, enforcing absolute and idle expiry.
func (s *Service) Authenticate(ctx context.Context, token string) (Admin, error) {
	var a Admin
	var lastSeen time.Time
	err := s.Pool.QueryRow(ctx, `SELECT u.id, u.email, u.display_name, s.id, s.expires_at, s.last_seen_at
		FROM authz.admin_sessions s JOIN authz.admin_users u ON u.id = s.admin_id
		WHERE s.token_hash = $1 AND s.stage = 'active' AND s.revoked_at IS NULL AND s.expires_at > now()
		  AND u.status = 'active'`, secure.HashToken(token)).Scan(&a.ID, &a.Email, &a.DisplayName, &a.SessionID, &a.ExpiresAt, &lastSeen)
	if errors.Is(err, db.ErrNoRows) {
		return Admin{}, apperr.New(apperr.Unauthenticated, "")
	}
	if err != nil {
		return Admin{}, err
	}
	now := s.now()
	if now.Sub(lastSeen) > s.IdleTimeout {
		_, _ = s.Pool.Exec(ctx, `UPDATE authz.admin_sessions SET revoked_at = now() WHERE id = $1`, a.SessionID)
		return Admin{}, apperr.New(apperr.Unauthenticated, "session idle timeout")
	}
	if now.Sub(lastSeen) > touchInterval {
		_, _ = s.Pool.Exec(ctx, `UPDATE authz.admin_sessions SET last_seen_at = now() WHERE id = $1`, a.SessionID)
	}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT b.role_key, rp.permission_key
		FROM authz.admin_role_bindings b JOIN authz.role_permissions rp ON rp.role_key = b.role_key
		WHERE b.admin_id = $1 ORDER BY 1, 2`, a.ID)
	if err != nil {
		return Admin{}, err
	}
	defer rows.Close()
	seenRole := map[string]bool{}
	for rows.Next() {
		var role, perm string
		if err := rows.Scan(&role, &perm); err != nil {
			return Admin{}, err
		}
		if !seenRole[role] {
			seenRole[role] = true
			a.Roles = append(a.Roles, role)
		}
		a.Permissions = append(a.Permissions, perm)
	}
	return a, rows.Err()
}

// Logout revokes the session.
func (s *Service) Logout(ctx context.Context, a Admin, m RequestMeta) error {
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: a.ID}, db.TxOptions{}, func(tx db.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE authz.admin_sessions SET revoked_at = now() WHERE id = $1`, a.SessionID); err != nil {
			return err
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "admin", ActorID: a.ID, Action: "admin.logout",
			TargetType: "admin_session", TargetID: a.SessionID, IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID})
		return err
	})
}

func nullIP(ip string) any {
	if ip == "" {
		return nil
	}
	return ip
}
