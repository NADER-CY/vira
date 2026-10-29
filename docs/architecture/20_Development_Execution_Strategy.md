# 20 — Development Execution Strategy (Claude-led)

**Date:** 2026-10-29 · **Revision:** v4 · **Status:** final for owner acceptance · **Phase 1 is NOT AUTHORISED.**

This document applies the owner's instruction *"VIRA — Final Development Strategy: Claude-Led End-to-End Development"*. **It supersedes all staffing and team-size assumptions** in docs 01, 12, 13 and 17. The architecture, technical standards, system boundaries, feature roadmap and security requirements are **unchanged**. Docs 00–19 remain the design reference and are not reopened without a concrete technical reason.

It contains the five v4 deliverables:

| § | Deliverable |
|---|---|
| 1 | Updated development execution strategy |
| 2 | Simplified phase-by-phase roadmap |
| 3 | Minimal development infrastructure requirements |
| 4 | Final Phase 0 summary |
| 5 | Phase 1 implementation readiness checklist |

---

## 1. Development execution strategy

### 1.1 Roles

| Role | Responsibilities | Does not do |
|---|---|---|
| **Product owner** | Final decisions and phase authorisations. Owns all accounts, contracts, credentials and legal relationships: repository, hosting, PSP, SMS, app stores, Apple/Google developer accounts. Accepts each phase report. Approves production deployments. | Write code |
| **Claude** (architect and sole implementation owner) | All code: Go backend, PostgreSQL, Flutter app, React Super Admin and Seller Admin. Also tests, CI/CD, migrations, staging deployment scripts, bug fixes, performance work, technical documentation and phase completion reports. The only party that commits to the main branch. | Sign contracts, open provider accounts, pass provider KYC, hold production secrets in chat, activate regulated features |
| **ChatGPT** (independent reviewer) | Reviews when the owner asks. Focus: React/TypeScript and admin frontends, comparison against the UI archive, test results. | Commit code. Findings come back as review comments, and Claude applies them, so there is never more than one writer. |
| **Human specialists** (later, only when necessary) | Independent security audit and pentest. Financial and regulatory verification. Production-readiness review. Critical issues that AI-assisted work cannot resolve reliably. | Day-to-day development |

The first table in §2 records when human involvement is **mandatory**. This follows from the owner's own rule that regulated functionality keeps its legal, security and provider approvals.

### 1.2 How work runs

1. **One repository** (monorepo) is the single source of truth: `backend/`, `mobile/`, `web/admin/`, `web/seller/`, `contracts/` (OpenAPI and event schemas), `deploy/`, `docs/`. Docs 00–20 move into `docs/architecture/`.
2. **Session continuity.** Claude works in sessions. Each phase keeps a tracked `docs/progress/Pn.md` with scope, a checklist, decisions taken, and known issues. Any new session resumes from the repository and that file, not from chat history.
3. **Change flow.** Each work item is a branch and a pull request. CI must be green. Claude reviews its own diff against the architecture rules, then merges.
4. **Owner approval before merge** is needed only for:
   - authentication, authorisation and cryptography changes;
   - ledger and payment code;
   - destructive migrations;
   - anything that deploys outside staging.
5. **Per-phase cycle.** The cycle is the owner's own, kept as written: review architecture → implement → connect → write tests → run → fix → verify on staging → write a completion report (one page: what works, test results, known gaps, what the next phase needs) → wait for authorisation of the next phase.
6. **Decisions.** Implementation choices that the ADRs leave open are made and recorded in one line in the progress file. Examples: ADR-013 state management (Riverpod proposed) and ADR-012's VM-plus-Docker path for staging. A new ADR is written only when an accepted ADR has to change.
7. **UI fidelity.** Screens are built from the archive HTML. The archive's tokens, layouts, RTL and both themes are preserved. The only screens that change are those covered by the doc 19 corrections (UC-1…UC-10). Each screen is traced to its archive file in the inventory CSV (doc 02a).

### 1.3 Definition of done (every phase)

- [ ] Features work end to end against real local services (PostgreSQL, Redis, NATS, object storage), not mock UI.
- [ ] Automated tests at the levels doc 11 sets for the phase, all green in CI:
  - unit;
  - integration with a real PostgreSQL;
  - API contract;
  - Flutter widget and golden tests (RTL, both themes);
  - end-to-end for the phase's main journeys.
- [ ] Mocks only for external providers: SMS, PSP, push, video. Each mock is clearly named and switched off in production configuration. Production builds fail to start if a mock adapter is configured.
- [ ] No fake success paths. Financial flows run only against the PSP simulator or sandbox until FG-1b.
- [ ] Deployed to staging, and the phase demo performed there.
- [ ] Completion report written.

### 1.4 Practical constraints of AI-led development

These are stated plainly so that plans do not rest on wrong assumptions. None of them reopens the architecture.

| Constraint | Consequence | Handling |
|---|---|---|
| Claude acts only inside sessions. It has no 24/7 presence. | Nobody is on call in production. | Alerting goes to the owner. Before public launch, a named human or a contracted operator for on-call and incident response (gate L, §2). |
| Claude cannot hold accounts, sign contracts or pass provider KYC. | Provider integrations wait for owner action. | The owner opens the accounts. Claude builds against mocks or sandboxes until access exists. |
| Production secrets must not pass through chat. | — | Secrets live in the CI or deployment secret store, entered by the owner. Claude references them by name. |
| Self-review has limits. | Defects that the author misses. | Tests first. ChatGPT review on request. **Independent human security audit before real users and before real money** (gates L and FG-1b). |
| Dual control (maker–checker, doc 19 UC-6) needs **two different humans**. | Payouts, refunds and hold releases cannot be operated by one person. | At least two admin operators named before FG-1b. |
| iOS builds need macOS and Xcode, and distribution needs an Apple Developer account. | iOS may not be buildable or distributable from Iran. | Android first. Code stays iOS-compatible (B-4, doc 11 §4.1). iOS builds run in macOS CI when an account exists. |
| The current Phase 0 session has no package-registry access (verified 2026-10-29, §5.2). | No code can be built in this environment. | Implementation sessions need an environment with registry access. The owner sets this up (§5.1, item O-4). |
| Throughput depends on session frequency and owner turnaround. | Calendar estimates would be guesses. | **No calendar durations are given in v4.** Phase sizes are relative (§2). The first forecast is measured at the end of Phase 1 and repeated in every completion report. |

---

## 2. Simplified phase-by-phase roadmap

Phase numbering and content follow doc 12 §v3, which remains authoritative for detailed scope and the ten attributes. Size is relative: S < M < L < XL.

| Phase | Builds (working software) | Size | Owner inputs needed during the phase | Gate to enter |
|---|---|---|---|---|
| **P1 Foundations** | Detail in §5.3: monorepo, local stack, CI, module skeletons, admin auth, Flutter and React shells, staging | M | Repository; staging account by phase end | **Written authorisation** |
| **P2 Identity & social core** | Phone + 6-digit OTP sign-up, profiles, follow, posts, feed, likes and comments, unticked consents (UC-5, UC-9), basic moderation, Super Admin user management | L | SMS provider account (mock until then); D-12, D-13, D-18, D-22 | Authorisation |
| **P3 Media, reels, stories, explore** | Upload, image pipeline, video via the `VideoProvider` adapter, reels, stories, explore, creator studio basics | L | Video platform account, or self-hosted (ADR-009) | Authorisation |
| **P4 Stores, catalog & seller tools** | Stores, products, inventory, "Manage My Stall" in the app, **Seller Admin web** on the same backend with shared identity (doc 04 §10), KYC levels without in-app card capture (UC-2) | XL | D-09, D-10, D-15, D-23; KYC provider | Authorisation |
| **P5 Messaging & notifications** | 1:1 messaging, realtime, push, seller notifications on mobile and web | L | Push setup (domestic option per ADR-021) | Authorisation |
| **FG-1a Financial-model decision** | No build. Owner chooses Model 1 (platform collects) or Model 2 (PSP split) | — | Counsel opinion, PSP confirmation | Owner decision |
| **P6 Cart, checkout, payments & ledger** | Cart, checkout, PSP hand-off (UC-1), double-entry ledger in integer Rial, reconciliation, all against the **simulator/sandbox** | XL | PSP sandbox access | **FG-1a** + authorisation |
| **P7 Orders, fulfilment, returns, settlement** | Orders, shipping by sellers, returns and refunds, seller settlement per the chosen model (sandbox), Super Admin dual-control consoles (UC-6, UC-7) | XL | D-08, D-24 | Authorisation |
| **L Launch readiness** | Hardening, load test at expected launch traffic, backups and restore drill, monitoring and alerting, **independent human security audit and pentest**, named on-call arrangement, store listings (Bazaar, Myket) | M | Production hosting contract; specialists | Owner decision |
| **FG-1b Real-money activation** | Turns on live payments and settlement. No new features. | — | Finance sign-off on posting rules; PSP contract; licences and trust seal; ≥ 2 named admin operators | Counsel, PSP, finance, security, owner |
| **P8 Live shopping & calls** | Live (provider or self-hosted), WebRTC calls, in-call purchase, live auctions (legal review), virtual gifts only after G1 | XL | D-19, D-33; TURN/SFU or provider | Authorisation |
| **P9 Growth** | Ads and ad billing, affiliate, **creator earnings**, AI features, seller plans and VIRA Plus, loyalty, blue badge, recommendations | XL | D-20, D-21, D-30, D-32; tax treatment. Money payouts need FG-1b | Authorisation |
| **P10 Scale** | Triggered by measured load: read replicas, partitioning, search cluster, managed Kubernetes if ADR-012 conditions are met | per trigger | Hosting capacity | Metrics |
| **P11 Advanced logistics** | Express delivery, rider tracking, carrier routing; delivery model chosen at the gate (D-29) | L | Delivery-model decision | Authorisation |
| **P12 Advanced messaging & channels** | Groups, broadcast channels, invoices in chat | L | D-11 | Authorisation |
| **G1 Wallet** | Stored value, top-up, transfers, gift cards, cashback | XL | Legal or licence opinion, security review | Separate written authorisation; FG-1b |
| **G2 USDT** | USDT payments, two-way USDT↔Toman conversion, eligible seller USDT withdrawals | XL | Its own design, legal and AML work, independent security review | Separate written authorisation; G1 |
| **G3 Credit / BNPL** | Instalments | L | Lending licence or bank partnership | Separate written authorisation; FG-1b |

Nothing has been deleted from the roadmap. P3 and P4 may run in either order after P2, but they are built one after another because there is one implementer. Regulated features stay behind feature flags that are **off** and cannot be switched on from the admin UI until their gate is recorded.

---

## 3. Minimal development infrastructure requirements

Start with local development, CI and one small staging server. Nothing production-scale is bought before gate L.

### 3.1 Required now (Phase 1)

| Item | Purpose | Who provides | Cost |
|---|---|---|---|
| **Git repository** (private), access granted to Claude's sessions | Source of truth, continuity, CI | Owner | GitHub: free tier usually enough. Self-hosted Gitea on the staging VM is the alternative if GitHub access becomes unreliable. |
| **Implementation environment with package-registry access**: Go module proxy, npm, pub.dev, Flutter SDK, Android SDK and Gradle/Maven, Docker Hub or a mirror | Build and test | Owner enables network access for Claude's sessions | Included in the environment |
| **CI runner** | Tests, lint, Android debug APK, React builds | GitHub Actions (free minutes for private repos; overage paid) or a self-hosted runner on the staging VM | Low |
| **Local stack** via Docker Compose: PostgreSQL 16, PgBouncer, Valkey, NATS, MinIO (S3-compatible), Mailpit, provider mocks (SMS, PSP simulator, push) | Development and integration tests | Claude (in the repository) | Free |

### 3.2 Required by the end of Phase 1 (does not block starting)

| Item | Sizing | Estimated cost |
|---|---|---|
| **One staging VM** at a domestic provider (doc 18; ArvanCloud primary candidate), run with Docker Compose. This is ADR-012's "VMs + Docker" path. | 4 vCore / 8 GB / 100 GB SSD / 1 IPv4 + 50 GB object storage | **≈ 5.5 M Toman/month** |
| Upgrade the same VM from P3, for media and video tests | 8 vCore / 16 GB / 200 GB SSD + 200 GB object storage | **≈ 11.2 M Toman/month** |
| Domain and TLS certificates (Let's Encrypt or domestic) | — | Small |
| Off-VM backup of the staging database (object storage) | — | Included above |

Costs are list prices from the dated ArvanCloud snapshot (2026-10-10), labelled S+ in doc 18. They are not quotes. Reproduce them with `python3 evidence/cost_model.py`.

### 3.3 Later, only when needed

| When | Item |
|---|---|
| P2 | SMS OTP provider account (until then a mock with codes visible only in dev) |
| P3 | Video platform account, or self-hosted FFmpeg/HLS on the staging VM |
| P4 | KYC provider sandbox |
| P5 | Push provider set-up |
| P6 | PSP sandbox |
| L | Production environment sized from measured load (the doc 17 resource models remain the scaling reference), second-provider recovery copy, monitoring stack (ADR-020), human security audit |
| iOS | Apple Developer account and macOS CI, if B-4 allows |

### 3.4 Recurring costs not priced here

These are the owner's own subscriptions, so no figure is given:

- Claude plan;
- ChatGPT plan;
- specialist fees at gates L and FG-1b.

The doc 17 team salaries **no longer apply** to development.

---

## 4. Final Phase 0 summary

| Item | State |
|---|---|
| Architecture (docs 04–08, 10, 11) | **Unchanged.** Go modular monolith, PostgreSQL, Flutter, React admin and seller web, outbox/NATS, integer-Rial ledger, domestic hosting |
| ADRs (doc 05) | Matrix unchanged: 19 ready, 3 with externally blocked components (011 settlement branch, 012 provider contract, 019 activation). ADR-012's VM path and ADR-013's state-management choice are settled during P1 as allowed. |
| UI (docs 02, 15, 19) | Archive is the design reference; preserve and convert it. Correct only the UC-1…UC-10 patterns. |
| Owner decisions (doc 13 §0) | B-1 confirmed; B-2 scope retained in full; **B-3 resolved by this strategy:** Claude-led development, no development team hired, specialists only at named gates. B-4 Android first; B-5 primary/recovery strategy; B-6 matrix; B-7 corrections. |
| Budget (doc 17) | Team scenarios A/B/C are **superseded for development staffing**. Their infrastructure models are kept as the scaling reference for production. Development infrastructure is in §3. |
| Roadmap | §2 (execution view) plus doc 12 §v3 (detailed scope). Every long-term and gated feature is retained. |
| New risks | R-48…R-52 (doc 13) |
| Phase 0 status | **Ready for final owner acceptance.** |
| Phase 1 status | **NOT AUTHORISED.** It starts only on the owner's written authorisation, then proceeds directly to implementation with no further planning cycle. |

**Still prohibited until separately authorised:**
- Real-money payment, settlement, payout, wallet, custody and USDT functionality.
- The seven unsafe UI patterns (B-7).
- Mandatory foreign runtime dependencies.
- Changing the original UI archive.

---

## 5. Phase 1 implementation readiness checklist

### 5.1 Owner actions

| # | Item | Blocks start? |
|---|---|---|
| O-1 | **Written authorisation of Phase 1** (P1-B1) | **Yes** |
| O-2 | Acceptance of Phase 0, including the ADR matrix in doc 05 (P1-B2) | **Yes** |
| O-3 | **Create a private repository** and give Claude's sessions write access (P1-B3, replaces "staffing") | **Yes** |
| O-4 | **Enable package-registry network access** for Claude's implementation sessions (Go proxy, npm, pub.dev and Flutter SDK, Android SDK and Maven/Google, Docker Hub) (P1-B3) | **Yes** |
| O-5 | Confirm the app identifiers: app name, Android package name (e.g. `ir.vira.app`), web domains for staging | No (placeholders until confirmed) |
| O-6 | Domestic provider account and one staging VM (§3.2) (P1-B4, now needed by **end** of P1) | No |
| O-7 | Decide where ChatGPT reviews are run and how findings are passed back (pasted into the session, or as PR comments) | No |

### 5.2 Environment check performed for this revision (2026-10-29, Phase 0 session)

| Check | Result |
|---|---|
| Go toolchain | Present (1.24) |
| Node.js | Present (22) |
| Java (Android builds) | Present |
| PostgreSQL client | Present (16) |
| Git access to github.com | **Works** |
| Go module proxy, npm, pub.dev, Flutter SDK, Maven/Google, Docker Hub | **Not reachable** (DNS or proxy policy) |
| Docker daemon | **Not running** |
| Flutter SDK | **Not installed** |

Conclusion: this Phase 0 session cannot build the project. Item O-4 is required, or CI on the repository must do the builds. No code was written to perform these checks, and nothing was installed.

### 5.3 Phase 1 scope and acceptance (what "done" means)

**Builds**

1. Monorepo layout (§1.2), with `docs/progress/P1.md`.
2. `docker compose up` brings up the full local stack (§3.1) in one command.
3. **Go backend skeleton:**
   - `api`, `admin-api`, `worker`, `realtime` and `media` binaries;
   - module boundaries enforced by an architecture lint;
   - configuration that refuses mock adapters in production.
4. **Database:**
   - migrations tool;
   - one schema per module;
   - per-runtime DB roles;
   - RLS session-variable pattern;
   - `platform.outbox` and the PostgreSQL job queue.
5. **Cross-cutting backend:**
   - error model;
   - idempotency middleware;
   - audit log;
   - feature flags (regulated flags locked off);
   - OpenAPI contract and generated clients.
6. **Admin auth realm:** password + TOTP, RBAC skeleton, separate binary (ADR-016).
7. **Flutter app shell:**
   - Android first, iOS-compatible code;
   - RTL, Vazirmatn, Jalali dates and Persian digits;
   - light and dark themes from the archive tokens;
   - navigation per doc 09;
   - Riverpod.
8. **React shells:** Super Admin and Seller Admin, with shared tokens and RTL.
9. **CI:** lint, tests, build, Android debug APK.
10. **Staging:** a Compose deployment script to the staging VM, and a backup/restore script.

**Acceptance**

- Clean checkout to running stack with one command.
- CI green and under 15 minutes.
- Outbox crash test: kill the process between commit and publish, and no event is lost.
- Idempotency tests pass.
- Architecture lint blocks a forbidden import.
- RTL golden tests pass in both themes.
- Admin login with TOTP works end to end.
- A "health" request travels app → API → DB on staging (or locally until O-6 is ready).
- Restore drill passes.
- Completion report delivered.

**Deferred from doc 12 P1 to gate L** (simplified under ADR-012 and ADR-020, not removed):
- Kubernetes and full IaC;
- a Vault cluster (CI/host secret store until then);
- the full Prometheus/Grafana/Loki/Tempo stack (structured logs and an OpenTelemetry-ready metrics endpoint until then);
- admin VPN;
- in-country dependency mirrors;
- on-call rota.
