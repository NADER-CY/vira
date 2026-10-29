// Command api is the public API runtime (mobile app and seller web).
package main

import (
	"os"

	"github.com/NADER-CY/vira/backend/internal/app/publicapi"
	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/platform/flags"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
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
	_, verifier, err := runtime.JWTKeys(cfg, log, true)
	if err != nil {
		log.Error("jwt keys", "err", err)
		os.Exit(1)
	}
	deps := publicapi.Deps{
		Pool:     pool,
		Verifier: verifier,
		Flags:    &flags.Reader{Pool: pool},
		Health:   &health.Handler{Service: "api", Version: runtime.Version, Checks: runtime.DependencyChecks(cfg, pool)},
		Log:      log,
		Version:  runtime.Version,

		AllowedOrigins: cfg.AllowedOrigins,
	}
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, publicapi.Handler(deps, publicapi.New(deps))); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
