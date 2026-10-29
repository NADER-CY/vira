package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

// lookup serves m, plus valid PII keys unless the map sets them (use "-" to leave a key unset).
func lookup(m map[string]string) func(string) string {
	return func(k string) string {
		if v, ok := m[k]; ok {
			if v == "-" {
				return ""
			}
			return v
		}
		switch k {
		case "VIRA_PII_ENC_KEY":
			return key32
		case "VIRA_PII_INDEX_KEY":
			return keyB
		}
		return ""
	}
}

var (
	key32 = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	keyB  = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("i", 32)))
)

func TestPIIKeysRequiredAndDistinct(t *testing.T) {
	if _, err := LoadFrom("api", lookup(map[string]string{"VIRA_DB_PUBLIC_URL": "postgres://x", "VIRA_PII_ENC_KEY": "-"})); err == nil {
		t.Error("api without VIRA_PII_ENC_KEY accepted")
	}
	same := map[string]string{"VIRA_DB_PUBLIC_URL": "postgres://x", "VIRA_PII_INDEX_KEY": key32}
	if _, err := LoadFrom("api", lookup(same)); err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Errorf("identical PII keys accepted: %v", err)
	}
	if _, err := LoadFrom("worker", lookup(map[string]string{"VIRA_DB_WORKER_URL": "postgres://x", "VIRA_PII_ENC_KEY": "-", "VIRA_PII_INDEX_KEY": "-"})); err != nil {
		t.Errorf("worker should not need PII keys: %v", err)
	}
}

func TestDevsimSMSRefusedInProduction(t *testing.T) {
	env := map[string]string{"VIRA_ENV": "production", "VIRA_DB_PUBLIC_URL": "postgres://x", "VIRA_PROVIDER_SMS": "devsim"}
	if _, err := LoadFrom("api", lookup(env)); err == nil || !strings.Contains(err.Error(), "VIRA_PROVIDER_SMS=devsim") {
		t.Errorf("devsim SMS accepted in production: %v", err)
	}
}

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
	env["VIRA_PROVIDER_SMS"], env["VIRA_PROVIDER_PSP"], env["VIRA_PROVIDER_PUSH"] = "mock", "psp-x", "pushe"
	env["VIRA_JWT_PRIVATE_KEY_FILE"], env["VIRA_JWT_PUBLIC_KEY_FILE"] = "/k/priv.pem", "/k/pub.pem"
	// No real SMS provider adapter exists yet, so production must still refuse to start (fail closed).
	if _, err := LoadFrom("api", lookup(env)); err == nil || !strings.Contains(err.Error(), "VIRA_PROVIDER_SMS=mock") {
		t.Fatalf("production with mock SMS accepted: %v", err)
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
