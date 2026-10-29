// Command smoke runs end-to-end checks against a running VIRA stack (local compose or CI):
// health of every runtime, client config, the full admin login (password + TOTP) flow, regulated-flag lock
// (409 feature_locked + denied audit event) and the SMS simulator.
//
//	VIRA_SMOKE_ADMIN_EMAIL, VIRA_SMOKE_ADMIN_PASSWORD, VIRA_SMOKE_TOTP_SECRET   admin created with `admin-api create-admin`
//	VIRA_SMOKE_BASE                                                              default http://127.0.0.1 (ports 8080–8084, 8090)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/secure"
)

var (
	failures int
	client   = &http.Client{Timeout: 10 * time.Second}
)

func check(name string, ok bool, detail string) {
	if ok {
		fmt.Printf("PASS %s\n", name)
		return
	}
	failures++
	fmt.Printf("FAIL %s: %s\n", name, detail)
}

func call(method, url string, body any, hdr map[string]string) (int, map[string]any, string) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp.StatusCode, m, string(raw)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	base := env("VIRA_SMOKE_BASE", "http://127.0.0.1")
	api, admin, devsim := base+":8080", base+":8081", base+":8090"

	for svc, port := range map[string]string{"api": "8080", "admin-api": "8081", "worker": "8082", "realtime": "8083", "media": "8084"} {
		code, _, raw := call("GET", base+":"+port+"/health/ready", nil, nil)
		check("ready "+svc, code == 200, fmt.Sprintf("%d %s", code, raw))
	}
	code, _, raw := call("GET", devsim+"/health/live", nil, nil)
	check("live devsim", code == 200, fmt.Sprintf("%d %s", code, raw))

	code, cfg, raw := call("GET", api+"/v1/config", nil, nil)
	locked := fmt.Sprint(cfg["regulated_locked"])
	check("public config reports regulated capabilities locked", code == 200 && strings.Contains(locked, "payments.real_money") &&
		strings.Contains(locked, "usdt.conversion"), raw)

	jar, _ := cookiejar.New(nil)
	client.Jar = jar
	csrf := map[string]string{"X-VIRA-CSRF": "1"}
	code, login, raw := call("POST", admin+"/v1/auth/login", map[string]string{
		"email": os.Getenv("VIRA_SMOKE_ADMIN_EMAIL"), "password": os.Getenv("VIRA_SMOKE_ADMIN_PASSWORD")}, csrf)
	check("admin login (password step)", code == 200 && login["mfa_token"] != nil, raw)
	code, _, raw = call("POST", admin+"/v1/auth/login", map[string]string{
		"email": os.Getenv("VIRA_SMOKE_ADMIN_EMAIL"), "password": "wrong-password-123"}, csrf)
	check("admin login rejects a wrong password", code == 401, fmt.Sprintf("%d %s", code, raw))

	totp, err := secure.TOTPCode(os.Getenv("VIRA_SMOKE_TOTP_SECRET"), secure.TOTPStep(time.Now()))
	check("compute TOTP", err == nil, fmt.Sprint(err))
	code, _, raw = call("POST", admin+"/v1/auth/mfa", map[string]string{"mfa_token": fmt.Sprint(login["mfa_token"]), "code": totp}, csrf)
	check("admin MFA (TOTP step) sets session", code == 200, fmt.Sprintf("%d %s", code, raw))

	code, me, raw := call("GET", admin+"/v1/auth/me", nil, nil)
	check("admin session is authenticated", code == 200 && me["email"] == os.Getenv("VIRA_SMOKE_ADMIN_EMAIL"), raw)

	code, fl, raw := call("GET", admin+"/v1/admin/flags", nil, nil)
	ok := code == 200
	if items, _ := fl["data"].([]any); ok {
		for _, it := range items {
			f := it.(map[string]any)
			if f["regulated"] == true && f["enabled"] == true {
				ok = false
			}
		}
		ok = ok && len(items) > 0
	}
	check("admin flag list: no regulated flag enabled", ok, raw)

	code, prob, raw := call("PUT", admin+"/v1/admin/flags/payments.real_money", map[string]any{"enabled": true, "reason": "smoke test"},
		map[string]string{"X-VIRA-CSRF": "1", "If-Match": `"1"`})
	check("enabling a regulated flag is refused (409 feature_locked, gate FG-1b)",
		code == 409 && prob["code"] == "feature_locked" && prob["gate"] == "FG-1b", fmt.Sprintf("%d %s", code, raw))

	code, au, raw := call("GET", admin+"/v1/admin/audit-events?action=flags.update", nil, nil)
	found := false
	if items, _ := au["data"].([]any); code == 200 {
		for _, it := range items {
			e := it.(map[string]any)
			if e["target_id"] == "payments.real_money" && e["outcome"] == "denied" {
				found = true
			}
		}
	}
	check("denied attempt is in the audit log", found, raw)

	code, _, raw = call("POST", admin+"/v1/auth/login", map[string]string{"email": "x@y.z", "password": "p"}, nil)
	check("state-changing admin request without CSRF header is refused", code == 403, fmt.Sprintf("%d %s", code, raw))

	code, _, raw = call("POST", devsim+"/sms/v1/send", map[string]string{"to": "09120000000", "text": "smoke"}, nil)
	check("SMS simulator accepts a message", code == 202, raw)
	code, inbox, raw := call("GET", devsim+"/sms/v1/messages", nil, nil)
	items, _ := inbox["data"].([]any)
	check("SMS simulator inbox shows it", code == 200 && len(items) > 0 &&
		slices.ContainsFunc(items, func(i any) bool { return i.(map[string]any)["body"] == "smoke" }), raw)

	fmt.Printf("smoke: %d failure(s)\n", failures)
	if failures > 0 {
		os.Exit(1)
	}
}
