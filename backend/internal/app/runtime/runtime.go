// Package runtime holds the start-up code shared by every binary: configuration, logging, signal
// handling, HTTP server timeouts and graceful shutdown.
package runtime

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/config"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/logx"
)

// Version is set at build time: -ldflags "-X github.com/NADER-CY/vira/backend/internal/app/runtime.Version=..."
var Version = "dev"

// Start loads configuration and a logger, or exits.
func Start(name string) (*config.Config, *slog.Logger) {
	cfg, err := config.Load(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: invalid configuration:\n%v\n", name, err)
		os.Exit(2)
	}
	log := logx.New(name, string(cfg.Env), cfg.LogLevel)
	slog.SetDefault(log)
	log.Info("starting", "version", Version, "addr", cfg.HTTPAddr)
	return cfg, log
}

// SignalContext is cancelled on SIGINT/SIGTERM.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// OpenDB opens the runtime's pool.
func OpenDB(ctx context.Context, cfg *config.Config) (*db.Pool, error) {
	return db.Open(ctx, db.Options{URL: cfg.DatabaseURL, MaxConns: cfg.DatabaseMaxConns, PgBouncer: cfg.DatabasePgBouncer,
		AppName: "vira-" + cfg.Runtime})
}

// Serve runs an HTTP server until ctx is cancelled, then shuts down gracefully.
func Serve(ctx context.Context, log *slog.Logger, addr string, h http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return srv.Shutdown(sctx)
}

// DependencyChecks builds readiness checks for the configured dependencies.
func DependencyChecks(cfg *config.Config, pool *db.Pool) []health.Check {
	var cs []health.Check
	if pool != nil {
		cs = append(cs, health.Check{Name: "postgres", Run: func(ctx context.Context) error { return pool.Ping(ctx) }})
	}
	if cfg.ValkeyAddr != "" {
		cs = append(cs, health.Valkey(cfg.ValkeyAddr))
	}
	if cfg.S3Endpoint != "" {
		cs = append(cs, health.HTTP("object_storage", cfg.S3Endpoint+"/minio/health/ready"))
	}
	return cs
}

// JWTKeys loads the signing key (api only) and verification keys. In development without key files an
// ephemeral key is generated (tokens then only verify inside the same process); configuration validation
// refuses this outside development.
func JWTKeys(cfg *config.Config, log *slog.Logger, needPrivate bool) (*auth.Signer, *auth.Verifier, error) {
	var priv *ecdsa.PrivateKey
	var pub *ecdsa.PublicKey
	var err error
	switch {
	case cfg.JWTPublicKeyFile != "":
		if pub, err = auth.LoadPublicKey(cfg.JWTPublicKeyFile); err != nil {
			return nil, nil, err
		}
		if needPrivate {
			if priv, err = auth.LoadPrivateKey(cfg.JWTPrivateKeyFile); err != nil {
				return nil, nil, err
			}
			if !priv.PublicKey.Equal(pub) {
				return nil, nil, errors.New("JWT private and public keys do not match")
			}
		}
	case cfg.Env == config.EnvDevelopment || cfg.Env == config.EnvTest:
		log.Warn("no JWT key files configured: using an ephemeral development key")
		if priv, err = auth.GenerateKey(); err != nil {
			return nil, nil, err
		}
		pub = &priv.PublicKey
	default:
		return nil, nil, errors.New("JWT key files are required outside development")
	}
	kid := auth.KeyID(pub)
	v := &auth.Verifier{Keys: map[string]*ecdsa.PublicKey{kid: pub}, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Leeway: 30 * time.Second}
	var s *auth.Signer
	if priv != nil {
		s = &auth.Signer{Key: priv, KeyID: kid, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, TTL: cfg.AccessTokenTTL}
	}
	return s, v, nil
}
