// Package adminapi wires the Super Admin HTTP API (separate binary, ADR-016).
package adminapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NADER-CY/vira/backend/internal/modules/identity"
	"github.com/NADER-CY/vira/backend/internal/modules/moderation"
	"github.com/NADER-CY/vira/backend/internal/platform/adminauth"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/flags"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
)

// SessionCookie is the admin session cookie (HttpOnly, SameSite=Strict).
const SessionCookie = "vira_admin_session"

// CSRFHeader must be present on state-changing requests (forces a CORS preflight; SameSite=Strict cookies).
const CSRFHeader = "X-VIRA-CSRF"

// Deps are the API's dependencies.
type Deps struct {
	Pool           *db.Pool
	Auth           *adminauth.Service
	Health         *health.Handler
	Log            *slog.Logger
	CookieSecure   bool
	AllowedOrigins []string
	Version        string
	Env            string
	Users          *identity.AdminService
	Moderation     *moderation.Service
}

// New builds the admin router.
func New(d Deps) *httpx.Router {
	r := httpx.NewRouter()
	d.Health.Register(r)
	a := &api{d}
	authn := a.requireSession
	perm := a.requirePermission
	r.HandleFunc(http.MethodPost, "/v1/auth/login", a.login)
	r.HandleFunc(http.MethodPost, "/v1/auth/mfa", a.mfa)
	r.HandleFunc(http.MethodPost, "/v1/auth/logout", a.logout, authn)
	r.HandleFunc(http.MethodGet, "/v1/auth/me", a.me, authn)
	r.HandleFunc(http.MethodGet, "/v1/admin/system/status", a.systemStatus, authn, perm("system.read"))
	r.HandleFunc(http.MethodGet, "/v1/admin/flags", a.listFlags, authn, perm("flags.read"))
	r.HandleFunc(http.MethodPut, "/v1/admin/flags/{key}", a.setFlag, authn, perm("flags.write"))
	r.HandleFunc(http.MethodGet, "/v1/admin/config", a.listConfig, authn, perm("config.read"))
	r.HandleFunc(http.MethodGet, "/v1/admin/audit-events", a.listAudit, authn, perm("audit.read"))
	r.HandleFunc(http.MethodGet, "/v1/admin/roles", a.listRoles, authn, perm("admins.read"))
	a.routesP2(r)
	return r
}

// Handler wraps the router with the standard middleware stack.
func Handler(d Deps, r *httpx.Router) http.Handler {
	return httpx.Chain(r,
		httpx.RequestContext(d.Log),
		httpx.Recover,
		httpx.AccessLog,
		httpx.SecurityHeaders,
		httpx.CORS(d.AllowedOrigins),
		csrfGuard,
	)
}

type api struct{ d Deps }

type adminKey struct{}

func adminFrom(ctx context.Context) adminauth.Admin {
	a, _ := ctx.Value(adminKey{}).(adminauth.Admin)
	return a
}

func meta(r *http.Request) adminauth.RequestMeta {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := ""
	if parsed := net.ParseIP(host); parsed != nil {
		ip = parsed.String()
	}
	return adminauth.RequestMeta{IP: ip, UA: r.UserAgent(), TraceID: httpx.TraceID(r.Context())}
}

func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get(CSRFHeader) != "1" {
				httpx.WriteError(w, r, apperr.New(apperr.Forbidden, "missing "+CSRFHeader+" header"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *api) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookie)
		if err != nil || c.Value == "" {
			httpx.WriteError(w, r, apperr.New(apperr.Unauthenticated, ""))
			return
		}
		ad, err := a.d.Auth.Authenticate(r.Context(), c.Value)
		if err != nil {
			a.clearCookie(w)
			httpx.WriteError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), adminKey{}, ad)))
	})
}

func (a *api) requirePermission(p string) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ad := adminFrom(r.Context())
			if !ad.Can(p) {
				m := meta(r)
				_ = db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
					_, err := audit.Record(r.Context(), tx, audit.Event{ActorType: "admin", ActorID: ad.ID, Action: "admin.authorize",
						Outcome: audit.Denied, IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID,
						Metadata: map[string]any{"permission": p, "path": r.URL.Path}})
					return err
				})
				httpx.WriteError(w, r, apperr.New(apperr.Forbidden, ""))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *api) setCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, Secure: a.d.CookieSecure, SameSite: http.SameSiteStrictMode})
}

func (a *api) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.d.CookieSecure, SameSite: http.SameSiteStrictMode})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var fe []apperr.FieldError
	if req.Email == "" {
		fe = append(fe, apperr.FieldError{Field: "email", Code: "required"})
	}
	if req.Password == "" {
		fe = append(fe, apperr.FieldError{Field: "password", Code: "required"})
	}
	if fe != nil {
		httpx.WriteError(w, r, apperr.Validation(fe...))
		return
	}
	tok, err := a.d.Auth.Login(r.Context(), req.Email, req.Password, meta(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"mfa_required": true, "mfa_token": tok,
		"mfa_expires_in": int(adminauth.MFAStepTTL.Seconds())})
}

type mfaReq struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

func (a *api) mfa(w http.ResponseWriter, r *http.Request) {
	var req mfaReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.MFAToken == "" || len(req.Code) != 6 {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "code", Code: "invalid"}))
		return
	}
	tok, exp, err := a.d.Auth.VerifyMFA(r.Context(), req.MFAToken, req.Code, meta(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ad, err := a.d.Auth.Authenticate(r.Context(), tok)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	a.setCookie(w, tok, exp)
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, ad)
}

func (a *api) logout(w http.ResponseWriter, r *http.Request) {
	if err := a.d.Auth.Logout(r.Context(), adminFrom(r.Context()), meta(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	a.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, adminFrom(r.Context()))
}

func (a *api) systemStatus(w http.ResponseWriter, r *http.Request) {
	rep := a.d.Health.Evaluate(r.Context())
	var regulated []map[string]string
	for k, g := range flags.Regulated {
		regulated = append(regulated, map[string]string{"key": k, "gate": g, "state": "locked_off"})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"service": "admin-api", "version": a.d.Version, "environment": a.d.Env,
		"time": time.Now().UTC(), "readiness": rep, "regulated_capabilities": regulated,
	})
}

func (a *api) listFlags(w http.ResponseWriter, r *http.Request) {
	ad := adminFrom(r.Context())
	var out []flags.Flag
	err := db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{ReadOnly: true}, func(tx db.Tx) error {
		var err error
		out, err = flags.List(r.Context(), tx)
		return err
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

type setFlagReq struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

func (a *api) setFlag(w http.ResponseWriter, r *http.Request) {
	ad := adminFrom(r.Context())
	key := r.PathValue("key")
	ifMatch := strings.Trim(r.Header.Get("If-Match"), `"`)
	var req setFlagReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	m := meta(r)
	if gate, reg := flags.Regulated[key]; reg {
		_ = db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
			_, err := audit.Record(r.Context(), tx, audit.Event{ActorType: "admin", ActorID: ad.ID, Action: "flags.update",
				TargetType: "feature_flag", TargetID: key, Outcome: audit.Denied, IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID,
				Metadata: map[string]any{"requested_enabled": req.Enabled, "reason": "regulated", "gate": gate}})
			return err
		})
		e := apperr.New(apperr.FeatureLocked, "regulated capability; requires gate "+gate)
		e.Extra = map[string]any{"gate": gate}
		httpx.WriteError(w, r, e)
		return
	}
	if ifMatch == "" {
		httpx.WriteError(w, r, apperr.New(apperr.PreconditionRequired, ""))
		return
	}
	version, err := strconv.Atoi(ifMatch)
	if err != nil {
		httpx.WriteError(w, r, apperr.New(apperr.PreconditionFailed, "If-Match must be the flag version"))
		return
	}
	var f flags.Flag
	err = db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		f, err = flags.Set(r.Context(), tx, flags.SetRequest{Key: key, Enabled: req.Enabled, Version: version, Reason: req.Reason,
			ActorID: ad.ID, IP: m.IP, UA: m.UA, TraceID: m.TraceID})
		return err
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, f.Version))
	httpx.WriteJSON(w, http.StatusOK, f)
}

func (a *api) listConfig(w http.ResponseWriter, r *http.Request) {
	type item struct {
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		Public    bool            `json:"public"`
		UpdatedAt time.Time       `json:"updated_at"`
	}
	var out []item
	err := db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin}, db.TxOptions{ReadOnly: true}, func(tx db.Tx) error {
		rows, err := tx.Query(r.Context(), `SELECT key, value, public, updated_at FROM platform.remote_config ORDER BY key`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.Key, &it.Value, &it.Public, &it.UpdatedAt); err != nil {
				return err
			}
			out = append(out, it)
		}
		return rows.Err()
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// cursor = base64(created_at RFC3339Nano + "|" + id)
func (a *api) listAudit(w http.ResponseWriter, r *http.Request) {
	q := audit.Query{Action: r.URL.Query().Get("action"), ActorID: r.URL.Query().Get("actor_id")}
	if l := r.URL.Query().Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil || n < 1 || n > 100 {
			httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "limit", Code: "out_of_range"}))
			return
		}
		q.Limit = n
	}
	if c := r.URL.Query().Get("cursor"); c != "" {
		raw, err := base64.RawURLEncoding.DecodeString(c)
		parts := strings.SplitN(string(raw), "|", 2)
		if err != nil || len(parts) != 2 {
			httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "cursor", Code: "invalid"}))
			return
		}
		t, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "cursor", Code: "invalid"}))
			return
		}
		q.Before, q.BeforeID = t, parts[1]
	}
	if q.Limit == 0 {
		q.Limit = 50
	}
	ad := adminFrom(r.Context())
	var evs []audit.Event
	err := db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		if evs, err = audit.List(r.Context(), tx, q); err != nil {
			return err
		}
		// Admin list views are themselves audited, one event per query (doc 06 §8).
		m := meta(r)
		_, err = audit.Record(r.Context(), tx, audit.Event{ActorType: "admin", ActorID: ad.ID, Action: "audit.list",
			IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID,
			Metadata: map[string]any{"filter_action": q.Action, "filter_actor": q.ActorID, "rows": len(evs)}})
		return err
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var next any
	if len(evs) == q.Limit {
		last := evs[len(evs)-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID))
	}
	if evs == nil {
		evs = []audit.Event{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": evs, "next_cursor": next})
}

func (a *api) listRoles(w http.ResponseWriter, r *http.Request) {
	type role struct {
		Key         string   `json:"key"`
		NameFa      string   `json:"name_fa"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	var out []role
	rows, err := a.d.Pool.Query(r.Context(), `SELECT r.key, r.name_fa, r.description,
		COALESCE(array_agg(rp.permission_key ORDER BY rp.permission_key) FILTER (WHERE rp.permission_key IS NOT NULL), '{}')
		FROM authz.roles r LEFT JOIN authz.role_permissions rp ON rp.role_key = r.key GROUP BY r.key ORDER BY r.key`)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ro role
		if err := rows.Scan(&ro.Key, &ro.NameFa, &ro.Description, &ro.Permissions); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		out = append(out, ro)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}
