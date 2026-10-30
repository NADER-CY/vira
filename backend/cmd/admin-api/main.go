// Command admin-api is the Super Admin API runtime (separate binary, DB role and auth realm — ADR-016).
//
//	admin-api                      serve
//	admin-api create-admin -email a@b.c -name "..." -roles super_admin
//	                               create an admin; the password is read from VIRA_NEW_ADMIN_PASSWORD (never a flag,
//	                               so it does not appear in shell history); prints the TOTP enrolment URI once.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/NADER-CY/vira/backend/internal/app/adminapi"
	"github.com/NADER-CY/vira/backend/internal/app/runtime"
	"github.com/NADER-CY/vira/backend/internal/modules/identity"
	idc "github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	"github.com/NADER-CY/vira/backend/internal/modules/moderation"
	"github.com/NADER-CY/vira/backend/internal/platform/adminauth"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
)

func main() {
	cfg, log := runtime.Start("admin-api")
	ctx, stop := runtime.SignalContext()
	defer stop()

	pool, err := runtime.OpenDB(ctx, cfg)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	svc := &adminauth.Service{Pool: pool, TOTPKey: cfg.AdminTOTPKey, SessionTTL: cfg.AdminSessionTTL, IdleTimeout: cfg.AdminIdleTimeout}

	if len(os.Args) > 1 && os.Args[1] == "create-admin" {
		os.Exit(createAdmin(ctx, svc, os.Args[2:]))
	}

	deps := adminapi.Deps{
		Pool:           pool,
		Auth:           svc,
		Health:         &health.Handler{Service: "admin-api", Version: runtime.Version, Checks: runtime.DependencyChecks(cfg, pool)},
		Log:            log,
		CookieSecure:   cfg.AdminCookieSecure,
		AllowedOrigins: cfg.AllowedOrigins,
		Version:        runtime.Version,
		Env:            string(cfg.Env),
		Users:          &identity.AdminService{Pool: pool, PII: idc.PII{EncKey: cfg.PIIEncKey, IndexKey: cfg.PIIIndexKey}},
		Moderation:     &moderation.Service{Pool: pool, Log: log},
	}
	if err := runtime.Serve(ctx, log, cfg.HTTPAddr, adminapi.Handler(deps, adminapi.New(deps))); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}

func createAdmin(ctx context.Context, svc *adminauth.Service, args []string) int {
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	email := fs.String("email", "", "admin e-mail (login)")
	name := fs.String("name", "", "display name")
	roles := fs.String("roles", "operator", "comma-separated roles: super_admin, auditor, operator")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pw := os.Getenv("VIRA_NEW_ADMIN_PASSWORD")
	if pw == "" {
		fmt.Fprintln(os.Stderr, "set VIRA_NEW_ADMIN_PASSWORD (at least 12 characters)")
		return 2
	}
	enr, err := svc.CreateAdmin(ctx, *email, *name, pw, strings.Split(*roles, ","), "cli:create-admin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create-admin:", err)
		return 1
	}
	// Printed once to the operator's terminal; scan the URI into an authenticator app.
	_ = json.NewEncoder(os.Stdout).Encode(enr)
	return 0
}
