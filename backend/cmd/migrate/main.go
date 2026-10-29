// Command migrate applies database migrations as the migrator role.
//
//	migrate up            apply all pending migrations
//	migrate up 2          apply up to version 0002
//	migrate down 1        revert the last migration (development/test only)
//	migrate status        list migrations and their state
//	migrate bootstrap-sql print the cluster role bootstrap script (run once by a superuser)
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/NADER-CY/vira/backend/internal/platform/config"
	"github.com/NADER-CY/vira/backend/internal/platform/migrate"
	"github.com/NADER-CY/vira/backend/migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate up [version] | down <steps> | status | bootstrap-sql")
		os.Exit(2)
	}
	if os.Args[1] == "bootstrap-sql" {
		fmt.Print(migrations.BootstrapRoles)
		return
	}
	cfg, err := config.Load("migrate")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ms, err := migrate.Load(migrations.FS)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	r := &migrate.Runner{Conn: conn, Migrations: ms, AllowDown: !cfg.Env.IsDeployed(),
		Log: func(f string, a ...any) { fmt.Printf(f+"\n", a...) }}

	arg := func(i, def int) int {
		if len(os.Args) > i {
			n, err := strconv.Atoi(os.Args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, "not a number:", os.Args[i])
				os.Exit(2)
			}
			return n
		}
		return def
	}
	switch os.Args[1] {
	case "up":
		n, err := r.Up(ctx, arg(2, 0))
		exitOn(err)
		fmt.Printf("%d migration(s) applied\n", n)
	case "down":
		n, err := r.Down(ctx, arg(2, 1))
		exitOn(err)
		fmt.Printf("%d migration(s) reverted\n", n)
	case "status":
		st, err := r.Status(ctx)
		exitOn(err)
		for _, s := range st {
			state := "pending"
			if s.Applied {
				state = "applied " + s.AppliedAt.UTC().Format(time.RFC3339)
			}
			fmt.Printf("%04d  %-30s %s\n", s.Version, s.Name, state)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command", os.Args[1])
		os.Exit(2)
	}
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
