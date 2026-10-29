package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func lookup(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

var key32 = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))

func TestDevelopmentDefaults(t *testing.T) {
	c, err := LoadFrom("api", lookup(map[string]string{"VIRA_DB_PUBLIC_URL": "postgres://x"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Env != EnvDevelopment || c.HTTPAddr != ":8080" || c.ProviderSMS != "mock" || c.DatabaseMaxConns != 10 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestEachRuntimeUsesItsOwnDatabaseRole(t *testing.T) {
	cases := map[string]string{"api": "VIRA_DB_PUBLIC_URL", "admin-api": "VIRA_DB_ADMIN_URL", "worker": "VIRA_DB_WORKER_URL", "migrate": "VIRA_DB_MIGRATOR_URL"}
	for runtime, v := range cases {
		env := map[string]string{v: "postgres://" + runtime, "VIRA_ADMIN_TOTP_KEY": key32}
		c, err := LoadFrom(runtime, lookup(env))
		if err != nil || c.DatabaseURL != "postgres://"+runtime {
			t.Errorf("%s: url=%q err=%v", runtime, c.DatabaseURL, err)
		}
		if _, err := LoadFrom(runtime, lookup(map[string]string{"VIRA_ADMIN_TOTP_KEY": key32})); err == nil {
			t.Errorf("%s: missing %s accepted", runtime, v)
		}
	}
}

func TestProductionRefusesMocksAndEphemeralKeys(t *testing.T) {
	env := map[string]string{"VIRA_ENV": "production", "VIRA_DB_PUBLIC_URL": "postgres://x"}
	_, err := LoadFrom("api", lookup(env))
	if err == nil {
		t.Fatal("production with mock providers and no JWT keys accepted")
	}
	for _, want := range []string{"VIRA_PROVIDER_SMS=mock", "VIRA_PROVIDER_PSP=simulator", "VIRA_JWT_PRIVATE_KEY_FILE"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
	env["VIRA_PROVIDER_SMS"], env["VIRA_PROVIDER_PSP"], env["VIRA_PROVIDER_PUSH"] = "kavenegar", "psp-x", "pushe"
	env["VIRA_JWT_PRIVATE_KEY_FILE"], env["VIRA_JWT_PUBLIC_KEY_FILE"] = "/k/priv.pem", "/k/pub.pem"
	if _, err := LoadFrom("api", lookup(env)); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestAdminSessionLimitsAreCapped(t *testing.T) {
	base := map[string]string{"VIRA_DB_ADMIN_URL": "postgres://x", "VIRA_ADMIN_TOTP_KEY": key32}
	for k, v := range map[string]string{"VIRA_ADMIN_SESSION_TTL": "9h", "VIRA_ADMIN_IDLE_TIMEOUT": "30m"} {
		env := map[string]string{k: v}
		for bk, bv := range base {
			env[bk] = bv
		}
		if _, err := LoadFrom("admin-api", lookup(env)); err == nil {
			t.Errorf("%s=%s accepted (ADR-016 limits)", k, v)
		}
	}
	bad := map[string]string{"VIRA_DB_ADMIN_URL": "postgres://x", "VIRA_ADMIN_TOTP_KEY": "c2hvcnQ="}
	if _, err := LoadFrom("admin-api", lookup(bad)); err == nil {
		t.Error("short TOTP key accepted")
	}
	staging := map[string]string{"VIRA_ENV": "staging", "VIRA_DB_ADMIN_URL": "postgres://x", "VIRA_ADMIN_TOTP_KEY": key32, "VIRA_ADMIN_COOKIE_SECURE": "false"}
	if _, err := LoadFrom("admin-api", lookup(staging)); err == nil {
		t.Error("insecure admin cookie accepted outside development")
	}
}

func TestUnknownEnvironment(t *testing.T) {
	if _, err := LoadFrom("media", lookup(map[string]string{"VIRA_ENV": "prod"})); err == nil {
		t.Fatal("unknown environment accepted")
	}
}
