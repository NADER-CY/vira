package adminapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NADER-CY/vira/backend/internal/app/adminapi"
	"github.com/NADER-CY/vira/backend/internal/platform/adminauth"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/secure"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
)

type env struct {
	srv *httptest.Server
	svc *adminauth.Service
	d   *testdb.DB
	now time.Time
}

func newEnv(t *testing.T) *env {
	d := testdb.New(t, true)
	e := &env{d: d, now: time.Now()}
	e.svc = &adminauth.Service{Pool: d.Pool("app_admin"), TOTPKey: bytes.Repeat([]byte{1}, 32),
		SessionTTL: 8 * time.Hour, IdleTimeout: 15 * time.Minute, Now: func() time.Time { return e.now }}
	deps := adminapi.Deps{Pool: d.Pool("app_admin"), Auth: e.svc, Health: &health.Handler{Service: "admin-api"},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Version: "test", Env: "test"}
	e.srv = httptest.NewServer(adminapi.Handler(deps, adminapi.New(deps)))
	t.Cleanup(e.srv.Close)
	return e
}

type client struct {
	t   *testing.T
	h   *http.Client
	url string
}

func (e *env) client(t *testing.T) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, h: &http.Client{Jar: jar}, url: e.srv.URL}
}

func (c *client) do(method, path string, body any, hdr map[string]string) (int, map[string]any, http.Header) {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.url+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set(adminapi.CSRFHeader, "1")
	}
	for k, v := range hdr {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.h.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out, resp.Header
}

func (e *env) createAdmin(t *testing.T, email string, roles ...string) adminauth.Enrollment {
	enr, err := e.svc.CreateAdmin(context.Background(), email, "Test Admin", "a-strong-password-123", roles, "test")
	if err != nil {
		t.Fatal(err)
	}
	return enr
}

func (e *env) login(t *testing.T, c *client, email string, enr adminauth.Enrollment, stepOffset int64) {
	t.Helper()
	st, body, _ := c.do("POST", "/v1/auth/login", map[string]string{"email": email, "password": "a-strong-password-123"}, nil)
	if st != 200 || body["mfa_required"] != true {
		t.Fatalf("login: %d %v", st, body)
	}
	code, _ := secure.TOTPCode(enr.TOTPSecret, secure.TOTPStep(e.now)+stepOffset)
	st, body, hdr := c.do("POST", "/v1/auth/mfa", map[string]string{"mfa_token": body["mfa_token"].(string), "code": code}, nil)
	if st != 200 {
		t.Fatalf("mfa: %d %v", st, body)
	}
	sc := hdr.Get("Set-Cookie")
	if !strings.Contains(sc, "HttpOnly") || !strings.Contains(sc, "SameSite=Strict") {
		t.Fatalf("session cookie flags: %s", sc)
	}
}

func TestFullLoginFlowAndPermissions(t *testing.T) {
	e := newEnv(t)
	enr := e.createAdmin(t, "root@vira.test", "super_admin")
	c := e.client(t)

	if st, _, _ := c.do("GET", "/v1/auth/me", nil, nil); st != 401 {
		t.Fatalf("unauthenticated me: %d", st)
	}
	e.login(t, c, "root@vira.test", enr, 0)
	st, me, _ := c.do("GET", "/v1/auth/me", nil, nil)
	if st != 200 || me["email"] != "root@vira.test" || len(me["permissions"].([]any)) != 6 {
		t.Fatalf("me: %d %v", st, me)
	}

	// Flags: list, then change a non-regulated flag with optimistic concurrency.
	st, fl, _ := c.do("GET", "/v1/admin/flags", nil, nil)
	if st != 200 || len(fl["data"].([]any)) != 10 {
		t.Fatalf("flags: %d %v", st, fl)
	}
	if st, _, _ := c.do("PUT", "/v1/admin/flags/app.maintenance_mode", map[string]any{"enabled": true, "reason": "test"}, nil); st != 428 {
		t.Fatalf("missing If-Match: %d", st)
	}
	if st, _, _ := c.do("PUT", "/v1/admin/flags/app.maintenance_mode", map[string]any{"enabled": true, "reason": "test"}, map[string]string{"If-Match": `"7"`}); st != 412 {
		t.Fatalf("stale If-Match: %d", st)
	}
	st, f, hdr := c.do("PUT", "/v1/admin/flags/app.maintenance_mode", map[string]any{"enabled": true, "reason": "planned upgrade"}, map[string]string{"If-Match": `"1"`})
	if st != 200 || f["enabled"] != true || hdr.Get("ETag") != `"2"` {
		t.Fatalf("set flag: %d %v etag=%s", st, f, hdr.Get("ETag"))
	}

	// Regulated flags cannot be enabled by anyone through the API.
	st, p, _ := c.do("PUT", "/v1/admin/flags/payments.real_money", map[string]any{"enabled": true, "reason": "try"}, map[string]string{"If-Match": `"1"`})
	if st != 409 || p["code"] != "feature_locked" || p["gate"] != "FG-1b" {
		t.Fatalf("regulated flag: %d %v", st, p)
	}

	// CSRF header is required on state-changing requests.
	if st, _, _ := c.do("POST", "/v1/auth/logout", nil, map[string]string{adminapi.CSRFHeader: ""}); st != 403 {
		t.Fatalf("missing csrf header: %d", st)
	}

	// Audit log contains the login, the flag change and the denied regulated attempt.
	st, au, _ := c.do("GET", "/v1/admin/audit-events?limit=100", nil, nil)
	if st != 200 {
		t.Fatalf("audit: %d", st)
	}
	actions := map[string]string{}
	for _, ev := range au["data"].([]any) {
		m := ev.(map[string]any)
		actions[m["action"].(string)+"/"+m["outcome"].(string)] = fmt.Sprint(m["target_id"])
	}
	for _, want := range []string{"admin.login/success", "flags.update/success", "flags.update/denied", "admin.create/success"} {
		if _, ok := actions[want]; !ok {
			t.Errorf("audit log missing %s (have %v)", want, actions)
		}
	}

	if st, _, _ := c.do("POST", "/v1/auth/logout", nil, nil); st != 204 {
		t.Fatalf("logout: %d", st)
	}
	if st, _, _ := c.do("GET", "/v1/auth/me", nil, nil); st != 401 {
		t.Fatalf("after logout: %d", st)
	}
}

func TestOperatorCannotWriteFlags(t *testing.T) {
	e := newEnv(t)
	enr := e.createAdmin(t, "ops@vira.test", "operator")
	c := e.client(t)
	e.login(t, c, "ops@vira.test", enr, 0)
	if st, _, _ := c.do("GET", "/v1/admin/flags", nil, nil); st != 200 {
		t.Fatalf("operator read flags: %d", st)
	}
	if st, _, _ := c.do("PUT", "/v1/admin/flags/app.maintenance_mode", map[string]any{"enabled": true, "reason": "x"}, map[string]string{"If-Match": `"1"`}); st != 403 {
		t.Fatalf("operator write flags: %d", st)
	}
	if st, _, _ := c.do("GET", "/v1/admin/audit-events", nil, nil); st != 403 {
		t.Fatalf("operator audit: %d", st)
	}
}

func TestTOTPReplayAndLockout(t *testing.T) {
	e := newEnv(t)
	enr := e.createAdmin(t, "a@vira.test", "auditor")
	c := e.client(t)
	e.login(t, c, "a@vira.test", enr, 0)

	// The same TOTP code cannot be used for a second login.
	c2 := e.client(t)
	_, body, _ := c2.do("POST", "/v1/auth/login", map[string]string{"email": "a@vira.test", "password": "a-strong-password-123"}, nil)
	code, _ := secure.TOTPCode(enr.TOTPSecret, secure.TOTPStep(e.now))
	if st, p, _ := c2.do("POST", "/v1/auth/mfa", map[string]string{"mfa_token": body["mfa_token"].(string), "code": code}, nil); st != 401 || p["code"] != "mfa_invalid" {
		t.Fatalf("replayed code: %d %v", st, p)
	}

	// Wrong passwords lock the account; even the right password is then refused.
	for i := 0; i < adminauth.MaxFailedLogins; i++ {
		c2.do("POST", "/v1/auth/login", map[string]string{"email": "a@vira.test", "password": "wrong-password-xx"}, nil)
	}
	if st, p, _ := c2.do("POST", "/v1/auth/login", map[string]string{"email": "a@vira.test", "password": "a-strong-password-123"}, nil); st != 403 || p["code"] != "account_restricted" {
		t.Fatalf("locked account: %d %v", st, p)
	}
	// Unknown accounts answer exactly like wrong passwords (no enumeration).
	st1, p1, _ := c2.do("POST", "/v1/auth/login", map[string]string{"email": "nobody@vira.test", "password": "whatever-123456"}, nil)
	if st1 != 401 || p1["code"] != "unauthenticated" {
		t.Fatalf("unknown account: %d %v", st1, p1)
	}
}

func TestIdleTimeout(t *testing.T) {
	e := newEnv(t)
	enr := e.createAdmin(t, "idle@vira.test", "operator")
	c := e.client(t)
	e.login(t, c, "idle@vira.test", enr, 0)
	e.now = e.now.Add(16 * time.Minute)
	if st, _, _ := c.do("GET", "/v1/auth/me", nil, nil); st != 401 {
		t.Fatalf("idle session still valid: %d", st)
	}
}

func TestRegulatedFlagsLockedInDatabase(t *testing.T) {
	d := testdb.New(t, true)
	// Even the admin DB role (BYPASSRLS) cannot enable a regulated flag: the CHECK constraint refuses it.
	_, err := d.Pool("app_admin").Exec(context.Background(), `UPDATE platform.feature_flags SET enabled = true WHERE key = 'usdt.conversion'`)
	if !db.IsCheckViolation(err) {
		t.Fatalf("expected check violation, got %v", err)
	}
	// And regulated flags cannot be added or reclassified as unregulated by the admin role.
	_, err = d.Pool("app_admin").Exec(context.Background(), `INSERT INTO platform.feature_flags (key, description) VALUES ('x.y','z')`)
	if err == nil {
		t.Fatal("app_admin must not insert flags")
	}
}

func TestAuditLogIsAppendOnly(t *testing.T) {
	e := newEnv(t)
	e.createAdmin(t, "x@vira.test", "operator")
	for _, sql := range []string{`UPDATE platform.audit_events SET action = 'tampered'`, `DELETE FROM platform.audit_events`} {
		if _, err := e.d.Pool("superuser").Exec(context.Background(), sql); err == nil {
			t.Fatalf("%s succeeded on the append-only audit log (even as superuser)", sql)
		}
	}
	if _, err := e.d.Pool("app_admin").Exec(context.Background(), `DELETE FROM platform.audit_events`); err == nil {
		t.Fatal("app_admin deleted audit events")
	}
}
