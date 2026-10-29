# VIRA — development guide

Architecture: `docs/architecture/` (Phase 0 v4, approved). Execution strategy and phase scope: `docs/architecture/20_Development_Execution_Strategy.md`. Progress: `docs/progress/`.

## Prerequisites

| Tool | Version |
|---|---|
| Docker + Compose v2 | recent |
| Go | 1.27 (see `backend/go.mod`) |
| Node.js | 22 |
| Flutter | stable channel (Dart ≥ 3.8), Java 17 for Android builds |
| Python | 3.10+ (generators: contracts, tokens) |

## Local stack

```bash
make dev-up          # creates .env + dev JWT keys (random, git-ignored), builds and starts everything, waits for health
make dev-admin EMAIL=you@example.com   # needs VIRA_NEW_ADMIN_PASSWORD in the environment; prints the TOTP URI once
make smoke           # end-to-end checks against the running stack
make dev-down        # stop (keeps data) — make dev-reset deletes the volumes
```

| Service | URL (127.0.0.1 only) | Notes |
|---|---|---|
| Public API (`api`) | http://localhost:8080 | `/health/live`, `/health/ready`, `/v1/config`, `/v1/me` |
| Super Admin API (`admin-api`) | http://localhost:8081 | session cookie + TOTP; `X-VIRA-CSRF: 1` on state-changing requests |
| Worker / realtime / media | :8082 / :8083 / :8084 | health endpoints; realtime WebSocket at `/v1/ws` |
| Provider simulators (`devsim`) | http://localhost:8090 | SMS inbox, push inbox, PSP IPG simulator — **no real money, development only** |
| PostgreSQL / PgBouncer | :5432 / :6432 | runtimes connect through PgBouncer (transaction mode); migrations connect directly |
| Valkey, NATS (monitor :8222), MinIO (console :9001), Mailpit (:8025) | | |

Database roles (doc 06 §1): `migrator` (owns objects; migrations only), `app_public`, `app_admin` (BYPASSRLS, audited), `app_worker`, `app_ledger_writer` (NOLOGIN, Phase 6). Each runtime uses its own role.

Migrations: `backend/migrations/NNNN_name.up.sql` + `.down.sql`, applied by `migrate up` (advisory lock, checksums, one transaction per file). CI runs up → down to zero → up on every change.

## Web (Super Admin, Seller Admin)

```bash
cd web
npm ci
npm run dev -w admin      # http://localhost:5173 (talks to admin-api :8081)
npm run dev -w seller     # http://localhost:5174 (talks to api :8080)
npm run typecheck && npm run lint && npm test && npm run build
```

API types are generated from `contracts/openapi/*.json` (`npm run gen:api`); change the contract with `python3 scripts/gen_openapi.py`, never by hand.

## Mobile (Flutter)

```bash
cd mobile
flutter pub get
flutter run --dart-define=VIRA_API_BASE_URL=http://10.0.2.2:8080   # Android emulator → local stack
flutter analyze && flutter test --exclude-tags golden
flutter test --tags golden           # golden screenshots (light/dark); --update-goldens after an intended UI change
flutter build apk --debug
```

## Design tokens

Edit `design/tokens/vira.tokens.json`, then `python3 scripts/gen_tokens.py` (writes the CSS and Dart token files and fails on WCAG AA contrast violations). CI fails if the generated files are stale.

## Tests

| Area | Command | CI job |
|---|---|---|
| Backend unit + integration (real PostgreSQL) | `VIRA_TEST_PG_URL=postgres://postgres:…@localhost:5432/postgres go test ./...` | backend / test |
| Migrations reversibility, regulated-flag lock | see `.github/workflows/backend.yml` | backend / migrations |
| Full stack smoke (admin login + TOTP, locked flags, simulators) | `make smoke` | backend / dev-stack-smoke |
| Contracts (OpenAPI lint, generated up to date, event schema) | `python3 scripts/gen_openapi.py --check` | contracts / validate |
| Web type check, lint, tests, build, screenshots | `cd web && npm test` | web / build-test |
| Flutter analyze, tests, goldens, debug APK | `cd mobile && flutter test` | mobile / build-test |
| Secrets (gitleaks, full history), Go vulnerabilities | — | security |

## Rules

* Never commit secrets: `.env`, keys and keystores are git-ignored; staging/production secrets live in the deployment secret store.
* Regulated capabilities (`payments.real_money`, `settlement.live`, `payouts.withdrawals`, `wallet.topup`, `usdt.*`, `credit.bnpl`) are locked off in code **and** by a database constraint; enabling one needs a reviewed migration + code change recording the approved gate — never an admin action.
* Production refuses to start with mock/simulator providers or ephemeral keys.
* RTL: use logical CSS properties / `EdgeInsetsDirectional`; Persian digits in UI; LTR fragments isolated.
* Feature branches + pull requests; CI must pass before merge.
