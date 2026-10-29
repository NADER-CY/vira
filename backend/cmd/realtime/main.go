// Command realtime is the WebSocket gateway runtime (doc 07 §3).
package main

import (
	"os"

	"github.com/NADER-CY/vira/backend/internal/app/realtime"
	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
)

func main() {
	cfg, log := runtime.Start("realtime")
	ctx, stop := runtime.SignalContext()
	defer stop()

	_, verifier, err := runtime.JWTKeys(cfg, log, false)
	if err != nil {
		log.Error("jwt keys", "err", err)
		os.Exit(1)
	}
	checks := runtime.DependencyChecks(cfg, nil)
	if cfg.NATSURL != "" {
		checks = append(checks, health.TCP("nats", natsAddr(cfg.NATSURL)))
	}
	deps := realtime.Deps{
		Verifier:       verifier,
		Health:         &health.Handler{Service: "realtime", Version: runtime.Version, Checks: checks},
		Log:            log,
		AllowedOrigins: originHosts(cfg.AllowedOrigins),
	}
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, realtime.Handler(deps, realtime.New(deps))); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
