// Package config loads runtime configuration from environment variables.
//
// Every runtime reads the same variable names (see .env.example at the repository root).
// Validation is strict for staging/production: mock provider adapters, ephemeral signing keys
// and missing secrets are refused at start-up, so a misconfigured deployment cannot boot.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env is the deployment environment.
type Env string

const (
	EnvDevelopment Env = "development"
	EnvTest        Env = "test"
	EnvStaging     Env = "staging"
	EnvProduction  Env = "production"
)

// IsDeployed reports whether the environment is a shared deployment (staging or production).
func (e Env) IsDeployed() bool { return e == EnvStaging || e == EnvProduction }

// Config is the full runtime configuration. Each runtime uses the subset it needs.
type Config struct {
	Env      Env
	Runtime  string // api | admin-api | worker | realtime | media | migrate
	HTTPAddr string
	LogLevel string

	// Database: one DSN per runtime role (doc 06 §1). The migrate runtime uses the migrator DSN.
	DatabaseURL string
	// PgBouncer transaction pooling: disables server-side prepared statement caching.
	DatabasePgBouncer bool
	DatabaseMaxConns  int32

	NATSURL    string
	ValkeyAddr string
	S3Endpoint string // e.g. http://localhost:9000 (MinIO) — readiness only in Phase 1
	PublicBase string // public API base URL, used in links

	// Auth (public realm): ES256 key pair in PEM files. Empty in development = ephemeral key.
	JWTPrivateKeyFile string
	JWTPublicKeyFile  string
	JWTIssuer         string
	JWTAudience       string
	AccessTokenTTL    time.Duration

	// Admin realm (ADR-016).
	AdminTOTPKey      []byte // 32-byte AES key that encrypts TOTP secrets at rest
	AdminSessionTTL   time.Duration
	AdminIdleTimeout  time.Duration
	AdminCookieSecure bool
	AllowedOrigins    []string // browser origins allowed by CORS / WebSocket (seller web, admin web)

	// Provider adapters. "mock" is allowed only in development/test (and staging for not-yet-contracted providers).
	ProviderSMS  string
	ProviderPSP  string
	ProviderPush string
}

// Load reads configuration for the named runtime from the process environment.
func Load(runtime string) (*Config, error) { return LoadFrom(runtime, os.Getenv) }

// LoadFrom reads configuration using the supplied lookup function (used by tests).
func LoadFrom(runtime string, get func(string) string) (*Config, error) {
	v := func(key, def string) string {
		if s := strings.TrimSpace(get(key)); s != "" {
			return s
		}
		return def
	}
	c := &Config{
		Env:               Env(v("VIRA_ENV", string(EnvDevelopment))),
		Runtime:           runtime,
		HTTPAddr:          v("VIRA_HTTP_ADDR", defaultAddr(runtime)),
		LogLevel:          v("VIRA_LOG_LEVEL", "info"),
		DatabaseURL:       v(dbVar(runtime), ""),
		NATSURL:           v("VIRA_NATS_URL", ""),
		ValkeyAddr:        v("VIRA_VALKEY_ADDR", ""),
		S3Endpoint:        v("VIRA_S3_ENDPOINT", ""),
		PublicBase:        v("VIRA_PUBLIC_BASE_URL", "http://localhost:8080"),
		JWTPrivateKeyFile: v("VIRA_JWT_PRIVATE_KEY_FILE", ""),
		JWTPublicKeyFile:  v("VIRA_JWT_PUBLIC_KEY_FILE", ""),
		JWTIssuer:         v("VIRA_JWT_ISSUER", "vira"),
		JWTAudience:       v("VIRA_JWT_AUDIENCE", "vira-api"),
		ProviderSMS:       v("VIRA_PROVIDER_SMS", "mock"),
		ProviderPSP:       v("VIRA_PROVIDER_PSP", "simulator"),
		ProviderPush:      v("VIRA_PROVIDER_PUSH", "mock"),
	}
	var errs []error
	switch c.Env {
	case EnvDevelopment, EnvTest, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("VIRA_ENV: unknown environment %q", c.Env))
	}
	var err error
	if c.DatabasePgBouncer, err = strconv.ParseBool(v("VIRA_DB_PGBOUNCER", "false")); err != nil {
		errs = append(errs, fmt.Errorf("VIRA_DB_PGBOUNCER: %w", err))
	}
	maxConns, err := strconv.Atoi(v("VIRA_DB_MAX_CONNS", "10"))
	if err != nil || maxConns < 1 || maxConns > 200 {
		errs = append(errs, fmt.Errorf("VIRA_DB_MAX_CONNS: must be 1..200"))
	}
	c.DatabaseMaxConns = int32(maxConns)
	if c.AccessTokenTTL, err = time.ParseDuration(v("VIRA_ACCESS_TOKEN_TTL", "10m")); err != nil {
		errs = append(errs, fmt.Errorf("VIRA_ACCESS_TOKEN_TTL: %w", err))
	}
	if c.AdminSessionTTL, err = time.ParseDuration(v("VIRA_ADMIN_SESSION_TTL", "8h")); err != nil || c.AdminSessionTTL > 8*time.Hour {
		errs = append(errs, errors.New("VIRA_ADMIN_SESSION_TTL: must be a duration of at most 8h (ADR-016)"))
	}
	if c.AdminIdleTimeout, err = time.ParseDuration(v("VIRA_ADMIN_IDLE_TIMEOUT", "15m")); err != nil || c.AdminIdleTimeout > 15*time.Minute {
		errs = append(errs, errors.New("VIRA_ADMIN_IDLE_TIMEOUT: must be a duration of at most 15m (ADR-016)"))
	}
	c.AdminCookieSecure = v("VIRA_ADMIN_COOKIE_SECURE", strconv.FormatBool(c.Env.IsDeployed())) == "true"
	if origins := v("VIRA_ALLOWED_ORIGINS", ""); origins != "" {
		for _, o := range strings.Split(origins, ",") {
			c.AllowedOrigins = append(c.AllowedOrigins, strings.TrimSpace(o))
		}
	}
	if k := v("VIRA_ADMIN_TOTP_KEY", ""); k != "" {
		key, err := base64.StdEncoding.DecodeString(k)
		if err != nil || len(key) != 32 {
			errs = append(errs, errors.New("VIRA_ADMIN_TOTP_KEY: must be base64 of exactly 32 bytes"))
		}
		c.AdminTOTPKey = key
	}

	if needsDB(runtime) && c.DatabaseURL == "" {
		errs = append(errs, fmt.Errorf("%s: required for runtime %s", dbVar(runtime), runtime))
	}
	if runtime == "admin-api" && len(c.AdminTOTPKey) == 0 {
		errs = append(errs, errors.New("VIRA_ADMIN_TOTP_KEY: required for admin-api"))
	}
	errs = append(errs, c.validateDeployed()...)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return c, nil
}

// validateDeployed enforces the rules that make production refuse to start with unsafe settings
// (doc 20 §1.3: "production builds fail to start if a mock adapter is configured").
func (c *Config) validateDeployed() []error {
	if !c.Env.IsDeployed() {
		return nil
	}
	var errs []error
	if c.Env == EnvProduction {
		for name, val := range map[string]string{"VIRA_PROVIDER_SMS": c.ProviderSMS, "VIRA_PROVIDER_PSP": c.ProviderPSP, "VIRA_PROVIDER_PUSH": c.ProviderPush} {
			if val == "mock" || val == "simulator" {
				errs = append(errs, fmt.Errorf("%s=%s is not allowed in production", name, val))
			}
		}
	}
	missingKeys := (c.Runtime == "api" && (c.JWTPrivateKeyFile == "" || c.JWTPublicKeyFile == "")) ||
		(c.Runtime == "realtime" && c.JWTPublicKeyFile == "")
	if missingKeys {
		errs = append(errs, errors.New("VIRA_JWT_PRIVATE_KEY_FILE/VIRA_JWT_PUBLIC_KEY_FILE: required outside development (no ephemeral keys)"))
	}
	if c.Runtime == "admin-api" && !c.AdminCookieSecure {
		errs = append(errs, errors.New("VIRA_ADMIN_COOKIE_SECURE must be true outside development"))
	}
	return errs
}

func needsDB(runtime string) bool {
	switch runtime {
	case "api", "admin-api", "worker", "migrate":
		return true
	}
	return false
}

// dbVar returns the DSN variable for a runtime: each runtime connects with its own DB role (doc 06 §1).
func dbVar(runtime string) string {
	switch runtime {
	case "admin-api":
		return "VIRA_DB_ADMIN_URL"
	case "worker":
		return "VIRA_DB_WORKER_URL"
	case "migrate":
		return "VIRA_DB_MIGRATOR_URL"
	default:
		return "VIRA_DB_PUBLIC_URL"
	}
}

func defaultAddr(runtime string) string {
	switch runtime {
	case "admin-api":
		return ":8081"
	case "worker":
		return ":8082"
	case "realtime":
		return ":8083"
	case "media":
		return ":8084"
	case "devsim":
		return ":8090"
	default:
		return ":8080"
	}
}
