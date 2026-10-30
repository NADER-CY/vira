package identity

import (
	"context"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	socialc "github.com/NADER-CY/vira/backend/internal/modules/social/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/ratelimit"
	"github.com/NADER-CY/vira/backend/internal/platform/secure"
	"github.com/NADER-CY/vira/backend/internal/platform/sms"
)

// Policy values (doc 07 §4.1, doc 08).
const (
	OTPLength      = 6
	OTPTTL         = 120 * time.Second
	OTPResendAfter = 60 * time.Second
	OTPMaxAttempts = 5
	SessionTTL     = 90 * 24 * time.Hour
	PoPMaxSkew     = 5 * time.Minute
)

// Rate-limit rules (doc 07 §4.1: phone 3/10 min and 10/day; IP and device limits against enumeration and SMS
// pumping).
var (
	RuleOTPPhone10m  = ratelimit.Rule{Name: "otp_phone_10m", Limit: 3, Window: 10 * time.Minute}
	RuleOTPPhoneDay  = ratelimit.Rule{Name: "otp_phone_day", Limit: 10, Window: 24 * time.Hour}
	RuleOTPIP        = ratelimit.Rule{Name: "otp_ip", Limit: 20, Window: 10 * time.Minute}
	RuleOTPDevice    = ratelimit.Rule{Name: "otp_device", Limit: 10, Window: time.Hour}
	RuleVerifyIP     = ratelimit.Rule{Name: "otp_verify_ip", Limit: 30, Window: 10 * time.Minute}
	RuleRefreshIP    = ratelimit.Rule{Name: "refresh_ip", Limit: 120, Window: 10 * time.Minute}
	RuleRecoveryIP   = ratelimit.Rule{Name: "recovery_ip", Limit: 5, Window: 24 * time.Hour}
	RuleRecoveryUser = ratelimit.Rule{Name: "recovery_phone", Limit: 2, Window: 24 * time.Hour}
)

// Service implements sign-in, sessions and account operations.
type Service struct {
	Pool    *db.Pool
	Signer  *auth.Signer
	Limiter *ratelimit.Limiter
	SMS     sms.Sender
	PII     contract.PII
	Log     *slog.Logger
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// RequestMeta carries request attributes recorded with security events.
type RequestMeta struct {
	IP        string
	UserAgent string
	TraceID   string
	DeviceID  string
}

// OTPChallenge is returned by RequestOTP.
type OTPChallenge struct {
	ChallengeID string `json:"challenge_id"`
	ExpiresIn   int    `json:"expires_in"`
	ResendAfter int    `json:"resend_after"`
	CodeLength  int    `json:"code_length"`
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func codeHash(salt []byte, code string) []byte {
	m := hmac.New(sha256.New, salt)
	m.Write([]byte(code))
	return m.Sum(nil)
}

// RequestOTP creates a one-time code challenge and sends the code by SMS. The response never reveals whether
// the number has an account (doc 07 §4.1: no user enumeration).
func (s *Service) RequestOTP(ctx context.Context, rawPhone string, m RequestMeta) (OTPChallenge, error) {
	phone, ok := contract.NormalizePhone(rawPhone)
	if !ok {
		return OTPChallenge{}, apperr.Validation(apperr.FieldError{Field: "phone", Code: "invalid", Detail: "Iranian mobile number expected"})
	}
	bidx := s.PII.Index(phone)
	key := base64.RawURLEncoding.EncodeToString(bidx[:12])
	// Client-side abuse (one IP or device hammering) gets an explicit 429; the per-number limits are applied
	// below and stay silent so that they cannot be used to probe numbers.
	if err := s.Limiter.Check(ctx, true, ratelimit.Keyed{Rule: RuleOTPIP, Key: m.IP}, ratelimit.Keyed{Rule: RuleOTPDevice, Key: m.DeviceID}); err != nil {
		return OTPChallenge{}, err
	}
	resp := OTPChallenge{ChallengeID: ids.New(), ExpiresIn: int(OTPTTL.Seconds()), ResendAfter: int(OTPResendAfter.Seconds()), CodeLength: OTPLength}

	// Resend cooldown: a pending challenge younger than OTPResendAfter means the user must wait.
	var lastAt time.Time
	err := s.Pool.QueryRow(ctx, `SELECT created_at FROM identity.otp_challenges WHERE phone_bidx = $1 AND state = 'pending'
		ORDER BY created_at DESC LIMIT 1`, bidx).Scan(&lastAt)
	if err == nil && s.now().Sub(lastAt) < OTPResendAfter {
		e := apperr.New(apperr.RateLimited, "")
		e.Extra = map[string]any{"retry_after": int((OTPResendAfter - s.now().Sub(lastAt)).Seconds()) + 1}
		return OTPChallenge{}, e
	}
	if err := s.Limiter.Check(ctx, true, ratelimit.Keyed{Rule: RuleOTPPhone10m, Key: key}, ratelimit.Keyed{Rule: RuleOTPPhoneDay, Key: key}); err != nil {
		if e := apperr.As(err); e.Code == apperr.RateLimited {
			s.Log.Warn("otp: per-number limit reached; challenge not sent", "phone_last4", phone[7:])
			return resp, nil // silently not sent (doc 07 §4.1)
		}
		return OTPChallenge{}, err
	}

	code, err := randomCode()
	if err != nil {
		return OTPChallenge{}, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return OTPChallenge{}, err
	}
	enc, err := s.PII.Seal(phone)
	if err != nil {
		return OTPChallenge{}, err
	}
	var ip any
	if m.IP != "" {
		ip = m.IP
	}
	err = db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE identity.otp_challenges SET state = 'superseded' WHERE phone_bidx = $1 AND state = 'pending'`, bidx); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO identity.otp_challenges (id, phone_bidx, phone_enc, phone_last4, code_hash, salt,
			max_attempts, ip, device_id, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet, NULLIF($9, ''), $10)`,
			resp.ChallengeID, bidx, enc, phone[7:], codeHash(salt, code), salt, OTPMaxAttempts, ip, m.DeviceID, s.now().Add(OTPTTL))
		if err != nil {
			return err
		}
		_, err = audit.Record(ctx, tx, audit.Event{ActorType: "anonymous", Action: "auth.otp_requested", TargetType: "otp_challenge",
			TargetID: resp.ChallengeID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID,
			Metadata: map[string]any{"phone_last4": phone[7:]}})
		return err
	})
	if err != nil {
		return OTPChallenge{}, err
	}
	text := "کد ورود ویرا: " + code + "\nاین کد را در اختیار هیچ‌کس قرار ندهید."
	if err := s.SMS.Send(ctx, phone, text); err != nil {
		s.Log.Error("otp: sms delivery failed", "err", err)
		return OTPChallenge{}, apperr.Wrap(apperr.Unavailable, err)
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE identity.otp_challenges SET delivered = true WHERE id = $1`, resp.ChallengeID)
	return resp, nil
}

// DeviceInfo identifies the client device at sign-in. PublicKey (base64 SPKI, P-256) enables proof-of-possession
// on refresh (doc 07 §1).
type DeviceInfo struct {
	ID         string `json:"id"`
	Platform   string `json:"platform"`
	Model      string `json:"model"`
	AppVersion string `json:"app_version"`
	PublicKey  string `json:"public_key"`
}

// Tokens is a session's credential set.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"session_id"`
}

// SignIn is the result of a successful verification.
type SignIn struct {
	Tokens
	UserID string `json:"user_id"`
	Handle string `json:"handle"`
	IsNew  bool   `json:"is_new"`
}

func validDevice(d DeviceInfo) []apperr.FieldError {
	var fe []apperr.FieldError
	if l := len(d.ID); l < 8 || l > 128 {
		fe = append(fe, apperr.FieldError{Field: "device.id", Code: "invalid"})
	}
	switch d.Platform {
	case "android", "ios", "web":
	default:
		fe = append(fe, apperr.FieldError{Field: "device.platform", Code: "invalid"})
	}
	if len(d.Model) > 100 || len(d.AppVersion) > 40 {
		fe = append(fe, apperr.FieldError{Field: "device", Code: "too_long"})
	}
	if d.PublicKey != "" {
		if _, err := parsePublicKey(d.PublicKey); err != nil {
			fe = append(fe, apperr.FieldError{Field: "device.public_key", Code: "invalid"})
		}
	}
	return fe
}

func parsePublicKey(b64 string) (*ecdsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		if der, err = base64.RawURLEncoding.DecodeString(b64); err != nil {
			return nil, err
		}
	}
	k, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	pk, ok := k.(*ecdsa.PublicKey)
	if !ok || pk.Curve.Params().Name != "P-256" {
		return nil, errors.New("not a P-256 key")
	}
	return pk, nil
}

func normalizeDigits(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Consents are the explicit sign-up choices (doc 19 UC-5): each must be true and carries the text version shown.
type Consents struct {
	Terms           bool   `json:"terms"`
	Privacy         bool   `json:"privacy"`
	AgeConfirmation bool   `json:"age_confirmation"`
	TextVersion     string `json:"text_version"`
}

func (c Consents) problems() []apperr.FieldError {
	var fe []apperr.FieldError
	if !c.Terms {
		fe = append(fe, apperr.FieldError{Field: "consents.terms", Code: "required"})
	}
	if !c.Privacy {
		fe = append(fe, apperr.FieldError{Field: "consents.privacy", Code: "required"})
	}
	if !c.AgeConfirmation {
		fe = append(fe, apperr.FieldError{Field: "consents.age_confirmation", Code: "required"})
	}
	if l := len(c.TextVersion); l == 0 || l > 40 {
		fe = append(fe, apperr.FieldError{Field: "consents.text_version", Code: "invalid"})
	}
	return fe
}

// recordConsents stores each purpose once per text version (re-consent after a new version adds a row).
func recordConsents(ctx context.Context, tx db.Tx, userID string, c Consents, surface string, m RequestMeta) error {
	var ip any
	if m.IP != "" {
		ip = m.IP
	}
	for _, purpose := range []string{"terms", "privacy", "age_confirmation"} {
		if _, err := tx.Exec(ctx, `INSERT INTO identity.consents (id, user_id, purpose, text_version, surface, ip, device_id)
			VALUES ($1, $2, $3, $4, $5, $6::inet, NULLIF($7, '')) ON CONFLICT (user_id, purpose, text_version) WHERE withdrawn_at IS NULL DO NOTHING`,
			ids.New(), userID, purpose, c.TextVersion, surface, ip, m.DeviceID); err != nil {
			return err
		}
	}
	return nil
}

// VerifyOTP checks a code and signs the user in (creating the account on first sign-in).
func (s *Service) VerifyOTP(ctx context.Context, challengeID, code string, dev DeviceInfo, intendsToSell bool, consents Consents, m RequestMeta) (SignIn, error) {
	code = normalizeDigits(code)
	fe := consents.problems()
	if !ids.Valid(challengeID) {
		fe = append(fe, apperr.FieldError{Field: "challenge_id", Code: "invalid"})
	}
	if len(code) != OTPLength || strings.Trim(code, "0123456789") != "" {
		fe = append(fe, apperr.FieldError{Field: "code", Code: "invalid"})
	}
	fe = append(fe, validDevice(dev)...)
	if fe != nil {
		return SignIn{}, apperr.Validation(fe...)
	}
	if err := s.Limiter.Check(ctx, true, ratelimit.Keyed{Rule: RuleVerifyIP, Key: m.IP}); err != nil {
		return SignIn{}, err
	}
	var out SignIn
	var refresh string
	// failure is set when verification fails after state was written (attempt counter, lock, audit record):
	// the transaction then commits and the failure is returned afterwards.
	var failure *apperr.Error
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		var (
			bidx, phoneEnc, hash, salt []byte
			last4, state               string
			attempts, maxAttempts      int
			expires                    time.Time
		)
		err := tx.QueryRow(ctx, `SELECT phone_bidx, phone_enc, phone_last4, code_hash, salt, state, attempts, max_attempts, expires_at
			FROM identity.otp_challenges WHERE id = $1 FOR UPDATE`, challengeID).
			Scan(&bidx, &phoneEnc, &last4, &hash, &salt, &state, &attempts, &maxAttempts, &expires)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.OTPInvalid, "")
		}
		if err != nil {
			return err
		}
		fail := func(code apperr.Code, action string, meta map[string]any) error {
			_, _ = audit.Record(ctx, tx, audit.Event{ActorType: "anonymous", Action: action, TargetType: "otp_challenge",
				TargetID: challengeID, Outcome: audit.Failure, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID, Metadata: meta})
			failure = apperr.New(code, "")
			if r, ok := meta["attempts_remaining"]; ok {
				failure.Extra = map[string]any{"attempts_remaining": r}
			}
			return nil
		}
		switch {
		case state == "locked":
			return apperr.New(apperr.OTPLocked, "")
		case state != "pending":
			return apperr.New(apperr.OTPInvalid, "")
		case !s.now().Before(expires):
			if _, err := tx.Exec(ctx, `UPDATE identity.otp_challenges SET state = 'expired' WHERE id = $1`, challengeID); err != nil {
				return err
			}
			return fail(apperr.OTPExpired, "auth.otp_failed", map[string]any{"reason": "expired"})
		}
		attempts++
		if !hmac.Equal(codeHash(salt, code), hash) {
			newState := "pending"
			if attempts >= maxAttempts {
				newState = "locked"
			}
			if _, err := tx.Exec(ctx, `UPDATE identity.otp_challenges SET attempts = $2, state = $3 WHERE id = $1`, challengeID, attempts, newState); err != nil {
				return err
			}
			if newState == "locked" {
				return fail(apperr.OTPLocked, "auth.otp_locked", map[string]any{"attempts": attempts})
			}
			return fail(apperr.OTPInvalid, "auth.otp_failed", map[string]any{"reason": "mismatch", "attempts_remaining": maxAttempts - attempts})
		}
		if _, err := tx.Exec(ctx, `UPDATE identity.otp_challenges SET attempts = $2, state = 'verified', consumed_at = now() WHERE id = $1`,
			challengeID, attempts); err != nil {
			return err
		}

		// Resolve or create the account.
		var userID, status string
		err = tx.QueryRow(ctx, `SELECT u.id, u.status FROM identity.phone_identities p JOIN identity.users u ON u.id = p.user_id
			WHERE p.phone_bidx = $1 FOR UPDATE OF u`, bidx).Scan(&userID, &status)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			userID, status, out.IsNew = ids.New(), contract.StatusActive, true
			if _, err := tx.Exec(ctx, `INSERT INTO identity.users (id, intends_to_sell) VALUES ($1, $2)`, userID, intendsToSell); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO identity.phone_identities (id, user_id, phone_enc, phone_bidx, phone_last4, verified_at)
				VALUES ($1, $2, $3, $4, $5, now())`, ids.New(), userID, phoneEnc, bidx, last4); err != nil {
				return err
			}
			if out.Handle, err = socialc.CreateProfile(ctx, tx, userID); err != nil {
				return err
			}
			if _, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "account.registered",
				TargetType: "user", TargetID: userID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID}); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		if err := recordConsents(ctx, tx, userID, consents, dev.Platform, m); err != nil {
			return err
		}
		switch status {
		case contract.StatusSuspended, contract.StatusDeleted:
			return fail(apperr.AccountRestricted, "auth.login_refused", map[string]any{"status": status, "user_id": userID})
		case contract.StatusDeletionPending:
			// Signing in during the grace period cancels the deletion request.
			if _, err := contract.SetStatus(ctx, tx, userID, contract.StatusActive, "deletion cancelled by sign-in", "user", userID); err != nil {
				return err
			}
		}
		if out.Handle == "" {
			sum, err := socialc.Summaries(ctx, tx, []string{userID})
			if err != nil {
				return err
			}
			out.Handle = sum[userID].Handle
		}

		// Device and session.
		var pub []byte
		if dev.PublicKey != "" {
			k, _ := parsePublicKey(dev.PublicKey)
			pub, _ = x509.MarshalPKIXPublicKey(k)
		}
		var deviceID string
		if err := tx.QueryRow(ctx, `INSERT INTO identity.devices (id, user_id, client_device_id, platform, model, app_version, public_key)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7)
			ON CONFLICT (user_id, client_device_id) DO UPDATE SET platform = EXCLUDED.platform, model = EXCLUDED.model,
			  app_version = EXCLUDED.app_version, public_key = EXCLUDED.public_key, last_seen_at = now()
			RETURNING id`, ids.New(), userID, dev.ID, dev.Platform, dev.Model, dev.AppVersion, pub).Scan(&deviceID); err != nil {
			return err
		}
		out.SessionID = ids.New()
		var ip any
		if m.IP != "" {
			ip = m.IP
		}
		if _, err := tx.Exec(ctx, `INSERT INTO identity.sessions (id, user_id, device_id, expires_at, ip, user_agent)
			VALUES ($1, $2, $3, $4, $5::inet, NULLIF(left($6, 300), ''))`, out.SessionID, userID, deviceID, s.now().Add(SessionTTL), ip, m.UserAgent); err != nil {
			return err
		}
		var rhash []byte
		refresh, rhash = secure.NewToken()
		if _, err := tx.Exec(ctx, `INSERT INTO identity.refresh_tokens (token_hash, session_id) VALUES ($1, $2)`, rhash, out.SessionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE identity.users SET last_login_at = now() WHERE id = $1`, userID); err != nil {
			return err
		}
		out.UserID = userID
		_, err = audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "auth.login", TargetType: "session",
			TargetID: out.SessionID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID,
			Metadata: map[string]any{"platform": dev.Platform, "new_account": out.IsNew}})
		return err
	})
	if err != nil {
		return SignIn{}, err
	}
	if failure != nil {
		return SignIn{}, failure
	}
	tok, err := s.issue(out.UserID, out.SessionID, refresh)
	if err != nil {
		return SignIn{}, err
	}
	out.Tokens = tok
	return out, nil
}

func (s *Service) issue(userID, sessionID, refresh string) (Tokens, error) {
	at, _, err := s.Signer.IssueForSession(userID, "user", sessionID, nil)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: at, TokenType: "Bearer", ExpiresIn: int(s.Signer.TTL.Seconds()), RefreshToken: refresh, SessionID: sessionID}, nil
}

// PoPMessage is the message a device signs to prove possession of its key on refresh.
func PoPMessage(ts int64, refreshToken string) []byte {
	return []byte("VIRA-REFRESH\n" + strconv.FormatInt(ts, 10) + "\n" + refreshToken)
}

func verifyPoP(pubDER []byte, header, refresh string, now time.Time) bool {
	tsStr, sigStr, ok := strings.Cut(header, ".")
	if !ok {
		return false
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}
	if d := now.Sub(time.Unix(ts, 0)); d > PoPMaxSkew || d < -PoPMaxSkew {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil {
		return false
	}
	k, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		return false
	}
	pk, ok := k.(*ecdsa.PublicKey)
	if !ok {
		return false
	}
	digest := sha256.Sum256(PoPMessage(ts, refresh))
	return ecdsa.VerifyASN1(pk, digest[:], sig)
}

// Refresh rotates a refresh token. A token presented after it was rotated is treated as stolen: the whole
// session is revoked (doc 08). Devices that registered a key must sign the request (proof of possession).
func (s *Service) Refresh(ctx context.Context, refresh, pop string, m RequestMeta) (Tokens, error) {
	if refresh == "" || len(refresh) > 200 {
		return Tokens{}, apperr.New(apperr.Unauthenticated, "")
	}
	if err := s.Limiter.Check(ctx, false, ratelimit.Keyed{Rule: RuleRefreshIP, Key: m.IP}); err != nil {
		return Tokens{}, err
	}
	var (
		userID, sessionID, newRefresh string
		replay, popFailed             bool
	)
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		var (
			state, status string
			revoked       *time.Time
			expires       time.Time
			pub           []byte
		)
		err := tx.QueryRow(ctx, `SELECT rt.state, s.id, s.user_id, s.revoked_at, s.expires_at, u.status, d.public_key
			FROM identity.refresh_tokens rt
			JOIN identity.sessions s ON s.id = rt.session_id
			JOIN identity.users u ON u.id = s.user_id
			JOIN identity.devices d ON d.id = s.device_id
			WHERE rt.token_hash = $1 FOR UPDATE OF rt, s`, secure.HashToken(refresh)).
			Scan(&state, &sessionID, &userID, &revoked, &expires, &status, &pub)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.Unauthenticated, "")
		}
		if err != nil {
			return err
		}
		if revoked != nil || !s.now().Before(expires) {
			return apperr.New(apperr.Unauthenticated, "")
		}
		if state != "active" {
			replay = true
			if err := revokeSession(ctx, tx, sessionID, "refresh_replay"); err != nil {
				return err
			}
			_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "auth.refresh_replay_detected",
				TargetType: "session", TargetID: sessionID, Outcome: audit.Denied, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
			return err
		}
		if status != contract.StatusActive && status != contract.StatusRestricted {
			return apperr.New(apperr.Unauthenticated, "")
		}
		if len(pub) > 0 && !verifyPoP(pub, pop, refresh, s.now()) {
			popFailed = true
			_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "auth.refresh_pop_failed",
				TargetType: "session", TargetID: sessionID, Outcome: audit.Denied, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
			return err
		}
		var h []byte
		newRefresh, h = secure.NewToken()
		if _, err := tx.Exec(ctx, `UPDATE identity.refresh_tokens SET state = 'rotated', rotated_at = now() WHERE token_hash = $1`,
			secure.HashToken(refresh)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO identity.refresh_tokens (token_hash, session_id) VALUES ($1, $2)`, h, sessionID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE identity.sessions SET last_used_at = now() WHERE id = $1`, sessionID)
		return err
	})
	if err != nil {
		return Tokens{}, err
	}
	if replay || popFailed {
		return Tokens{}, apperr.New(apperr.Unauthenticated, "")
	}
	return s.issue(userID, sessionID, newRefresh)
}

func revokeSession(ctx context.Context, tx db.Tx, sessionID, reason string) error {
	if _, err := tx.Exec(ctx, `UPDATE identity.refresh_tokens SET state = 'revoked' WHERE session_id = $1 AND state = 'active'`, sessionID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE identity.sessions SET revoked_at = COALESCE(revoked_at, now()), revoke_reason = COALESCE(revoke_reason, $2)
		WHERE id = $1`, sessionID, reason)
	return err
}

// SessionState is what the per-request session check needs.
type SessionState struct {
	Status string
}

// CheckSession verifies that the token's session is still valid and returns the account status. Revoked
// sessions (logout, revocation, replay, suspension) are refused immediately even though access tokens are
// stateless JWTs.
func (s *Service) CheckSession(ctx context.Context, userID, sessionID string) (SessionState, error) {
	if !ids.Valid(sessionID) {
		return SessionState{}, apperr.New(apperr.Unauthenticated, "")
	}
	var st SessionState
	var revoked *time.Time
	var expires, lastUsed time.Time
	err := s.Pool.QueryRow(ctx, `SELECT s.revoked_at, s.expires_at, s.last_used_at, u.status FROM identity.sessions s
		JOIN identity.users u ON u.id = s.user_id WHERE s.id = $1 AND s.user_id = $2`, sessionID, userID).
		Scan(&revoked, &expires, &lastUsed, &st.Status)
	if err != nil || revoked != nil || !s.now().Before(expires) {
		return SessionState{}, apperr.New(apperr.Unauthenticated, "")
	}
	switch st.Status {
	case contract.StatusActive, contract.StatusRestricted:
	default:
		return SessionState{}, apperr.New(apperr.Unauthenticated, "")
	}
	if s.now().Sub(lastUsed) > 5*time.Minute {
		_, _ = s.Pool.Exec(ctx, `UPDATE identity.sessions SET last_used_at = now() WHERE id = $1`, sessionID)
	}
	return st, nil
}

// Logout revokes the current session.
func (s *Service) Logout(ctx context.Context, userID, sessionID string, m RequestMeta) error {
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		if err := revokeSession(ctx, tx, sessionID, "logout"); err != nil {
			return err
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "auth.logout", TargetType: "session",
			TargetID: sessionID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
}

// Session is one active session as shown to its owner.
type Session struct {
	ID         string    `json:"id"`
	Platform   string    `json:"platform"`
	Model      string    `json:"model,omitempty"`
	AppVersion string    `json:"app_version,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	Current    bool      `json:"current"`
}

// Sessions lists the user's active sessions.
func (s *Service) Sessions(ctx context.Context, userID, currentID string) ([]Session, error) {
	rows, err := s.Pool.Query(ctx, `SELECT s.id, d.platform, COALESCE(d.model, ''), COALESCE(d.app_version, ''), s.created_at, s.last_used_at
		FROM identity.sessions s JOIN identity.devices d ON d.id = s.device_id
		WHERE s.user_id = $1 AND s.revoked_at IS NULL AND s.expires_at > now() ORDER BY s.last_used_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var x Session
		if err := rows.Scan(&x.ID, &x.Platform, &x.Model, &x.AppVersion, &x.CreatedAt, &x.LastUsedAt); err != nil {
			return nil, err
		}
		x.Current = x.ID == currentID
		out = append(out, x)
	}
	return out, rows.Err()
}

// RevokeSession revokes one of the user's own sessions.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string, m RequestMeta) error {
	if !ids.Valid(sessionID) {
		return apperr.New(apperr.NotFound, "")
	}
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		var owner string
		if err := tx.QueryRow(ctx, `SELECT user_id FROM identity.sessions WHERE id = $1 AND revoked_at IS NULL`, sessionID).Scan(&owner); err != nil || owner != userID {
			return apperr.New(apperr.NotFound, "")
		}
		if err := revokeSession(ctx, tx, sessionID, "revoked_by_user"); err != nil {
			return err
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "session.revoked", TargetType: "session",
			TargetID: sessionID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
}

// RevokeOtherSessions signs out every other device.
func (s *Service) RevokeOtherSessions(ctx context.Context, userID, currentID string, m RequestMeta) error {
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		if err := contract.RevokeAllSessions(ctx, tx, userID, "revoked_others", currentID); err != nil {
			return err
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "session.revoked_others", TargetType: "user",
			TargetID: userID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
}

// Account is the signed-in user's account view.
type Account struct {
	UserID              string     `json:"user_id"`
	Status              string     `json:"status"`
	PhoneMasked         string     `json:"phone_masked"`
	IntendsToSell       bool       `json:"intends_to_sell"`
	CreatedAt           time.Time  `json:"created_at"`
	DeletionEffectiveAt *time.Time `json:"deletion_effective_at"`
}

// AccountOf returns the account of userID.
func (s *Service) AccountOf(ctx context.Context, userID string) (Account, error) {
	a := Account{UserID: userID}
	var last4 string
	err := s.Pool.QueryRow(ctx, `SELECT u.status, u.intends_to_sell, u.created_at, u.deletion_effective_at, COALESCE(p.phone_last4, '')
		FROM identity.users u LEFT JOIN identity.phone_identities p ON p.user_id = u.id WHERE u.id = $1`, userID).
		Scan(&a.Status, &a.IntendsToSell, &a.CreatedAt, &a.DeletionEffectiveAt, &last4)
	if err != nil {
		return Account{}, apperr.New(apperr.NotFound, "")
	}
	a.PhoneMasked = contract.MaskPhone(last4)
	return a, nil
}

// RequestDeletion starts the 30-day deletion grace period and signs out every device (doc 07: POST /me/deletion).
func (s *Service) RequestDeletion(ctx context.Context, userID string, m RequestMeta) (time.Time, error) {
	var eff time.Time
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		if _, err := contract.SetStatus(ctx, tx, userID, contract.StatusDeletionPending, "requested by user", "user", userID); err != nil {
			if errors.Is(err, contract.ErrNoChange) {
				return apperr.New(apperr.Conflict, "deletion already requested")
			}
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT deletion_effective_at FROM identity.users WHERE id = $1`, userID).Scan(&eff); err != nil {
			return err
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "account.deletion_requested",
			TargetType: "user", TargetID: userID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
	return eff, err
}

// RequestRecovery files an account-recovery request for a lost number (foundation; decided by staff, P4 adds
// identity proofing). Always answers the same way to avoid revealing which numbers have accounts.
func (s *Service) RequestRecovery(ctx context.Context, oldPhone, newPhone, note string, m RequestMeta) error {
	oldN, ok1 := contract.NormalizePhone(oldPhone)
	newN, ok2 := contract.NormalizePhone(newPhone)
	var fe []apperr.FieldError
	if !ok1 {
		fe = append(fe, apperr.FieldError{Field: "old_phone", Code: "invalid"})
	}
	if !ok2 {
		fe = append(fe, apperr.FieldError{Field: "new_phone", Code: "invalid"})
	}
	if len(note) > 500 {
		fe = append(fe, apperr.FieldError{Field: "note", Code: "too_long"})
	}
	if fe != nil {
		return apperr.Validation(fe...)
	}
	newIdx := s.PII.Index(newN)
	if err := s.Limiter.Check(ctx, true, ratelimit.Keyed{Rule: RuleRecoveryIP, Key: m.IP},
		ratelimit.Keyed{Rule: RuleRecoveryUser, Key: base64.RawURLEncoding.EncodeToString(newIdx[:12])}); err != nil {
		return err
	}
	enc, err := s.PII.Seal(newN)
	if err != nil {
		return err
	}
	var ip any
	if m.IP != "" {
		ip = m.IP
	}
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic}, db.TxOptions{}, func(tx db.Tx) error {
		id := ids.New()
		if _, err := tx.Exec(ctx, `INSERT INTO identity.recovery_requests (id, old_phone_bidx, new_phone_enc, new_phone_bidx, new_phone_last4, note, ip)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7::inet)`, id, s.PII.Index(oldN), enc, newIdx, newN[7:], note, ip); err != nil {
			return err
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "anonymous", Action: "account.recovery_requested", TargetType: "recovery_request",
			TargetID: id, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
}
