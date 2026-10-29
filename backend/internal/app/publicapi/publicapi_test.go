package publicapi_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NADER-CY/vira/backend/internal/app/publicapi"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/flags"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
)

type fixture struct {
	srv    *httptest.Server
	signer *auth.Signer
	d      *testdb.DB
	fr     *flags.Reader
}

func setup(t *testing.T) *fixture {
	d := testdb.New(t, true)
	k, _ := auth.GenerateKey()
	kid := auth.KeyID(&k.PublicKey)
	signer := &auth.Signer{Key: k, KeyID: kid, Issuer: "vira", Audience: "vira-api", TTL: time.Minute}
	fr := &flags.Reader{Pool: d.Pool("app_public"), TTL: time.Nanosecond}
	deps := publicapi.Deps{
		Pool:     d.Pool("app_public"),
		Verifier: &auth.Verifier{Keys: map[string]*ecdsa.PublicKey{kid: &k.PublicKey}, Issuer: "vira", Audience: "vira-api"},
		Flags:    fr,
		Health: &health.Handler{Service: "api", Checks: []health.Check{{Name: "postgres", Run: func(ctx context.Context) error {
			return d.Pool("app_public").Ping(ctx)
		}}}},
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version: "test",
	}
	srv := httptest.NewServer(publicapi.Handler(deps, publicapi.New(deps)))
	t.Cleanup(srv.Close)
	return &fixture{srv: srv, signer: signer, d: d, fr: fr}
}

func get(t *testing.T, url string, hdr map[string]string) (int, map[string]any, http.Header) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	return resp.StatusCode, m, resp.Header
}

func TestHealthAndConfig(t *testing.T) {
	f := setup(t)
	if st, b, _ := get(t, f.srv.URL+"/health/ready", nil); st != 200 || b["status"] != "ok" {
		t.Fatalf("ready: %d %v", st, b)
	}
	st, b, h := get(t, f.srv.URL+"/v1/config", nil)
	if st != 200 || h.Get("API-Version") != "1" {
		t.Fatalf("config: %d %v", st, b)
	}
	fl := b["flags"].(map[string]any)
	for k := range flags.Regulated {
		if fl[k] != false {
			t.Errorf("regulated flag %s reported %v", k, fl[k])
		}
	}
	if b["regulated_locked"].(map[string]any)["usdt.conversion"] != "G2" {
		t.Fatalf("regulated_locked: %v", b["regulated_locked"])
	}
	if b["config"].(map[string]any)["mobile.min_supported_version"] != "1.0.0" {
		t.Fatalf("config: %v", b["config"])
	}
}

func TestProblemResponsesAndTracing(t *testing.T) {
	f := setup(t)
	tp := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	st, b, h := get(t, f.srv.URL+"/v1/nope", map[string]string{"traceparent": tp})
	if st != 404 || b["code"] != "not_found" || b["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" ||
		!strings.HasPrefix(h.Get("Content-Type"), "application/problem+json") || b["title"] == "" {
		t.Fatalf("404: %d %v %v", st, b, h)
	}
	resp, _ := http.Post(f.srv.URL+"/v1/config", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != 405 {
		t.Fatalf("405 expected, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestMeRequiresValidToken(t *testing.T) {
	f := setup(t)
	if st, _, _ := get(t, f.srv.URL+"/v1/me", nil); st != 401 {
		t.Fatalf("anonymous me: %d", st)
	}
	tok, _, _ := f.signer.Issue("user-42", "user", []string{"basic"})
	st, b, _ := get(t, f.srv.URL+"/v1/me", map[string]string{"Authorization": "Bearer " + tok})
	if st != 200 || b["subject"] != "user-42" {
		t.Fatalf("me: %d %v", st, b)
	}
}

func TestMaintenanceModeAndMinVersion(t *testing.T) {
	f := setup(t)
	tok, _, _ := f.signer.Issue("user-1", "user", nil)
	auth := map[string]string{"Authorization": "Bearer " + tok}

	f.d.Pool("superuser").Exec(context.Background(), `UPDATE platform.remote_config SET value = '"2.3.0"' WHERE key = 'mobile.min_supported_version'`) //nolint:errcheck
	f.fr.Invalidate()
	h := map[string]string{"Authorization": auth["Authorization"], publicapi.AppVersionHeader: "2.2.9"}
	if st, b, _ := get(t, f.srv.URL+"/v1/me", h); st != 426 || b["code"] != "upgrade_required" || b["min_supported_version"] != "2.3.0" {
		t.Fatalf("old app: %d %v", st, b)
	}
	h[publicapi.AppVersionHeader] = "2.10.0"
	if st, _, _ := get(t, f.srv.URL+"/v1/me", h); st != 200 {
		t.Fatalf("new app: %d", st)
	}

	f.d.Pool("superuser").Exec(context.Background(), `UPDATE platform.feature_flags SET enabled = true WHERE key = 'app.maintenance_mode'`) //nolint:errcheck
	f.fr.Invalidate()
	if st, b, hdr := get(t, f.srv.URL+"/v1/me", auth); st != 503 || b["code"] != "maintenance" || hdr.Get("Retry-After") == "" {
		t.Fatalf("maintenance: %d %v", st, b)
	}
	if st, _, _ := get(t, f.srv.URL+"/health/live", nil); st != 200 {
		t.Fatal("health must stay available during maintenance")
	}
	if st, _, _ := get(t, f.srv.URL+"/v1/config", nil); st != 200 {
		t.Fatal("config must stay available during maintenance")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"1.0.0", "1.0.0", 0}, {"1.2.10", "1.2.9", 1}, {"1.2", "1.2.1", -1}, {"2.0.0+45", "2.0.0", 0}, {"0.9", "1.0", -1}}
	for _, c := range cases {
		if got := publicapi.CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("%s vs %s = %d want %d", c.a, c.b, got, c.want)
		}
	}
}
