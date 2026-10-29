// Command media is the media-processing runtime. Phase 1 provides the runtime, configuration and readiness
// against object storage; the upload pipeline and the VideoProvider adapter (ADR-009) arrive in Phase 3.
package main

import (
	"os"

	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
)

func main() {
	cfg, log := runtime.Start("media")
	ctx, stop := runtime.SignalContext()
	defer stop()

	r := httpx.NewRouter()
	(&health.Handler{Service: "media", Version: runtime.Version, Checks: runtime.DependencyChecks(cfg, nil)}).Register(r)
	h := httpx.Chain(r, httpx.RequestContext(log), httpx.Recover, httpx.AccessLog, httpx.SecurityHeaders)
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, h); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
