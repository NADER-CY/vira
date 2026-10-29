// Command devsim runs the development-only provider simulators (SMS, push, PSP IPG). It refuses to start in
// production: simulators must never stand in for real providers there (doc 20 §1.3).
package main

import (
	"os"

	"github.com/NADER-CY/vira/backend/internal/app/devsim"
	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/platform/config"
)

func main() {
	cfg, log := runtime.Start("devsim")
	if cfg.Env == config.EnvProduction {
		log.Error("devsim must not run in production")
		os.Exit(2)
	}
	ctx, stop := runtime.SignalContext()
	defer stop()
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, devsim.New().Handler()); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
