// Command api is the public API runtime (mobile app and seller web).
package main

import (
	"context"
	"os"
	"time"

	"github.com/NADER-CY/vira/backend/internal/app/publicapi"
	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/modules/identity"
	idc "github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	"github.com/NADER-CY/vira/backend/internal/modules/media"
	"github.com/NADER-CY/vira/backend/internal/modules/moderation"
	"github.com/NADER-CY/vira/backend/internal/modules/social"
	"github.com/NADER-CY/vira/backend/internal/platform/flags"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/objectstore"
	"github.com/NADER-CY/vira/backend/internal/platform/ratelimit"
	"github.com/NADER-CY/vira/backend/internal/platform/sms"
)

func main() {
	cfg, log := runtime.Start("api")
	ctx, stop := runtime.SignalContext()
	defer stop()

	pool, err := runtime.OpenDB(ctx, cfg)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	signer, verifier, err := runtime.JWTKeys(cfg, log, true)
	if err != nil {
		log.Error("jwt keys", "err", err)
		os.Exit(1)
	}
	if cfg.ValkeyAddr == "" {
		log.Error("VIRA_VALKEY_ADDR is required (rate limits for sign-in and content creation)")
		os.Exit(2)
	}
	limiter := ratelimit.New(cfg.ValkeyAddr)

	var sender sms.Sender
	switch cfg.ProviderSMS {
	case "devsim":
		sender = &sms.Devsim{URL: cfg.SMSDevsimURL}
	default: // "mock": codes are only logged as delivered to nobody (tests and offline development)
		sender = &sms.Mock{}
	}

	var store objectstore.Store = &objectstore.Memory{}
	if cfg.S3Endpoint != "" {
		s3 := &objectstore.S3{Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey}
		bctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		if err := s3.EnsureBucket(bctx); err != nil {
			cancel()
			log.Error("object storage", "err", err)
			os.Exit(1)
		}
		cancel()
		store = s3
	} else if cfg.Env.IsDeployed() {
		log.Error("VIRA_S3_ENDPOINT is required outside development")
		os.Exit(2)
	}

	pii := idc.PII{EncKey: cfg.PIIEncKey, IndexKey: cfg.PIIIndexKey}
	checks := runtime.DependencyChecks(cfg, pool)
	deps := publicapi.Deps{
		Pool:     pool,
		Verifier: verifier,
		Flags:    &flags.Reader{Pool: pool},
		Health:   &health.Handler{Service: "api", Version: runtime.Version, Checks: checks},
		Log:      log,
		Version:  runtime.Version,

		AllowedOrigins: cfg.AllowedOrigins,

		Identity:   &identity.Service{Pool: pool, Signer: signer, Limiter: limiter, SMS: sender, PII: pii, Log: log},
		Social:     &social.Service{Pool: pool, Limiter: limiter, Log: log},
		Media:      &media.Service{Pool: pool, Store: store, Limiter: limiter, Log: log},
		Moderation: &moderation.Service{Pool: pool, Limiter: limiter, Log: log},
	}
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, publicapi.Handler(deps, publicapi.New(deps))); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
