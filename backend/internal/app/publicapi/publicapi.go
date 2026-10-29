// Package publicapi wires the public HTTP API used by the mobile app and the seller web (doc 07).
// Phase 1 exposes the platform surface only; business endpoints are added by their phases' modules.
package publicapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/flags"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/idempotency"
)

// AppVersionHeader carries the mobile app version for the minimum-version gate (doc 07 §1).
const AppVersionHeader = "X-App-Version"

// Deps are the API's dependencies.
type Deps struct {
	Pool           *db.Pool
	Verifier       *auth.Verifier
	Flags          *flags.Reader
	Health         *health.Handler
	Log            *slog.Logger
	AllowedOrigins []string
	Version        string
}

// New builds the public router.
func New(d Deps) *httpx.Router {
	r := httpx.NewRouter()
	d.Health.Register(r)
	a := &api{d}
	r.HandleFunc(http.MethodGet, "/v1/config", a.config)
	r.HandleFunc(http.MethodGet, "/v1/me", a.me, d.Verifier.Middleware(true))
	return r
}

// Handler wraps the router with the standard middleware stack.
func Handler(d Deps, r *httpx.Router) http.Handler {
	idem := &idempotency.Middleware{Pool: d.Pool, Principal: func(r *http.Request) string {
		if p, ok := auth.FromContext(r.Context()); ok {
			return "user:" + p.Subject
		}
		return ""
	}}
	return httpx.Chain(r,
		httpx.RequestContext(d.Log),
		httpx.Recover,
		httpx.AccessLog,
		httpx.SecurityHeaders,
		httpx.CORS(d.AllowedOrigins),
		d.Verifier.Middleware(false), // attaches the principal when a token is present
		maintenance(d.Flags),
		minVersion(d.Flags),
		unsafeOnly(idem.Wrap),
	)
}

type api struct{ d Deps }

// config returns client configuration: non-regulated flag states and public remote config.
// Regulated capabilities are reported as disabled with their gate, so clients can hide them.
func (a *api) config(w http.ResponseWriter, r *http.Request) {
	en, cfg, err := a.d.Flags.Snapshot(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	locked := map[string]string{}
	for k, g := range flags.Regulated {
		locked[k] = g
		en[k] = false
	}
	if cfg == nil {
		cfg = map[string]json.RawMessage{}
	}
	w.Header().Set("Cache-Control", "public, max-age=30")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"api_version": httpx.APIVersion, "server_version": a.d.Version,
		"flags": en, "regulated_locked": locked, "config": cfg,
	})
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"subject": p.Subject, "kind": p.Kind, "scopes": p.Scopes,
		"expires_at": p.Expires.UTC()})
}

func exempt(path string) bool {
	return strings.HasPrefix(path, "/health/") || path == "/v1/config"
}

// maintenance answers 503 maintenance when app.maintenance_mode is on (health and config stay available).
func maintenance(fr *flags.Reader) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !exempt(r.URL.Path) {
				if on, err := fr.Enabled(r.Context(), flags.MaintenanceMode); err == nil && on {
					w.Header().Set("Retry-After", "300")
					httpx.WriteError(w, r, apperr.New(apperr.Maintenance, ""))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// minVersion rejects mobile clients older than mobile.min_supported_version with 426 upgrade_required.
func minVersion(fr *flags.Reader) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v := r.Header.Get(AppVersionHeader)
			if v != "" && !exempt(r.URL.Path) {
				if min, err := fr.ConfigString(r.Context(), "mobile.min_supported_version"); err == nil && CompareVersions(v, min) < 0 {
					e := apperr.New(apperr.UpgradeRequired, "")
					e.Extra = map[string]any{"min_supported_version": min}
					httpx.WriteError(w, r, e)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func unsafeOnly(mw httpx.Middleware) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				wrapped.ServeHTTP(w, r)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

// CompareVersions compares dotted numeric versions ("1.2.10" > "1.2.9"); non-numeric parts compare as 0.
func CompareVersions(a, b string) int {
	pa, pb := strings.Split(strings.SplitN(a, "+", 2)[0], "."), strings.Split(strings.SplitN(b, "+", 2)[0], ".")
	for i := 0; i < max(len(pa), len(pb)); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
