# 05 — Architecture Decision Records

**Status of every ADR:** *Proposed* — none is accepted until the product owner approves Phase 0. Ratings use ▲ (advantage), ● (neutral), ▼ (disadvantage) relative to the alternatives in the same table.

## Final acceptance matrix (v3 — 2026-10-29)

Owner decision **B-6**: ADR-001…022 are the baseline; preserve audited decisions unless new evidence shows a material defect. This matrix records the final consistency review. **Acceptance is not authorisation to implement.**

Status codes: **READY** = ready for owner acceptance · **AMENDED → READY** = amended in v3 for new evidence, now ready · **BLOCKED** = blocked by an external decision (provider, legal, financial) — the blocked part is named.

Mapping to the brief’s vocabulary: READY and AMENDED → READY = **READY FOR OWNER ACCEPTANCE** (19 ADRs; the required amendments have been drafted in this document); BLOCKED components = **BLOCKED BY EXTERNAL DECISION** (ADR-011 settlement branch, ADR-012 provider contract, ADR-019 activation). No ADR remains in **REQUIRES AMENDMENT**.

| ADR | Decision | Review result | Evidence / reason | Blocks Phase 1? |
|---|---|---|---|---|
| 001 | Modular monolith | **READY** | Boundaries, lint rules and runtime split consistent across docs 04, 06, 07 after round-1 fixes | — |
| 002 | Go backend | **READY** | No contrary evidence | — |
| 003 | Flutter mobile | **AMENDED → READY** | B-4: Android first, iOS from the start; amendment adds platform-neutrality constraints | — |
| 004 | PostgreSQL | **READY** | ADR text amended (round 1, F-18) for quorum sync + off-provider copy (doc 06 §11); managed-vs-self-managed choice deferred to provider due diligence (doc 18 §5.1) | — |
| 005 | Redis-compatible cache | **READY** | Provider Redis unconfirmed (doc 18) → self-run on VMs; no change | — |
| 006 | Outbox + core NATS, JetStream deferred | **READY** | Consistent in docs 04, 06, 07, 10 | — |
| 007 | PostgreSQL job queue | **READY** | — | — |
| 008 | REST/OpenAPI + WebSocket | **READY** | Seller Admin web and mobile seller mode share the same API (§ADR-017 amendment) | — |
| 009 | Media pipeline | **AMENDED → READY** | Doc 18: domestic video platform with published watch-time pricing → build-or-buy behind an adapter | — |
| 010 | Live and calls | **AMENDED → READY** (implementation gated, P8) | Provider live streaming exists (doc 18) → added as option | — |
| 011 | Double-entry ledger | **READY** for the ledger core; **BLOCKED** for the settlement branch | Platform-collects vs PSP split (F-01, financial gate, doc 06 §6.5) needs counsel + PSP confirmation | No (ledger is P6) |
| 012 | In-country hosting, IaC, conditional Kubernetes | **AMENDED**; **BLOCKED** on provider contract | Doc 18: primary/recovery strategy; ArvanCloud container platform is Kubernetes-based across 3 DCs; vendor answers (doc 18 §6) outstanding | **Partly** — a provider account for dev/CI/staging is needed to start P1 (§ doc 16) |
| 013 | Flutter state management (deferred) | **READY** | Decided by mobile lead in P1 | — |
| 014 | Vazirmatn, self-hosted assets | **READY** | Archive ships Vazirmatn v33.003 (doc 15) | — |
| 015 | Search: PostgreSQL → OpenSearch | **READY** | — | — |
| 016 | Super Admin isolation | **READY** | Separate binary/realm/DB role (round-1 F-20) | — |
| 017 | Authorization (RBAC + tenant ABAC + RLS) | **AMENDED → READY** | Owner seller requirement: shared identity, “Manage My Stall”, Seller Admin web, synchronized notifications | — |
| 018 | Identifiers | **READY** | — | — |
| 019 | Wallet and crypto | **AMENDED**; activation **BLOCKED** | B-2 keeps wallet, USDT payments, USDT↔Toman conversion and USDT seller withdrawals on the long-term roadmap as gated tracks; legal/licensing/provider approvals required before any build | No |
| 020 | Observability | **READY** | — | — |
| 021 | Push abstraction | **READY** | Consistent with B-4 (no hard Google Play Services dependency) | — |
| 022 | React + TypeScript web | **READY** | Seller Admin web confirmed by owner (§2 of the v3 brief) | — |

**Result (22 ADRs):** 15 READY · 4 AMENDED → READY (003, 009, 010, 017) · 3 with a BLOCKED component (011 settlement branch; 012 provider contract — strategy amended; 019 activation — scope amended). No ADR has a material defect requiring reversal.

## Index

| ADR | Decision | Verdict on proposed stack |
|---|---|---|
| 001 | Modular monolith, not microservices | — |
| 002 | Go for backend | ✅ Accept proposed |
| 003 | Flutter for mobile | ✅ Accept proposed, with iOS distribution caveat |
| 004 | PostgreSQL as primary store | ✅ Accept proposed |
| 005 | Redis-compatible cache (Redis or Valkey) — never a source of truth | ✅ Accept with licence note |
| 006 | Transactional outbox; core NATS for realtime now, JetStream at a measured trigger | ⚠️ Accept NATS; **defer JetStream** (amended after audit F-17) |
| 007 | PostgreSQL-backed job queue for background work | ➕ Addition |
| 008 | REST/JSON + OpenAPI 3.1; WebSocket for realtime | ✅ |
| 009 | Object storage + in-country CDN + FFmpeg/libvips pipeline | ✅ Accept proposed, provider TBD |
| 010 | WebRTC SFU for calls/hosts; HLS for live viewers | ✅ Refines “WebSocket/WebRTC where appropriate” |
| 011 | Double-entry ledger in PostgreSQL, integer Rial | ➕ |
| 012 | In-country hosting, containers, IaC; Kubernetes conditional | ➕ |
| 013 | Flutter state-management choice deferred to Phase 1 | ➕ |
| 014 | Vazirmatn, self-hosted fonts and icons | ➕ (corrects design docs) |
| 015 | Search: PostgreSQL first, OpenSearch in Phase 4 | ➕ |
| 016 | Super Admin isolated deployment & auth realm | ➕ |
| 017 | Authorization: RBAC + tenant ABAC in-process + PostgreSQL RLS | ➕ |
| 018 | Identifiers: UUIDv7 internal, opaque public IDs | ➕ |
| 019 | Wallet, USDT and conversion as gated long-term tracks; interfaces reserved, disabled by default | ➕ (amended v3) |
| 020 | Observability: OpenTelemetry + self-hosted Grafana stack | ➕ |
| 021 | Push delivery abstraction with in-app fallback | ➕ |
| 022 | React + TypeScript for admin and seller web | ✅ Accept proposed |

---

## ADR-001 — Modular monolith vs microservices

**Context.** 30+ domain areas, a small founding team (assumed 4–10 engineers, D-14), strict financial correctness, and an in-country hosting market with fewer managed services than hyperscalers.

| Criterion | Modular monolith | Microservices (≈15–25 services) |
|---|---|---|
| Performance | ▲ in-process calls, no network hops on checkout path | ▼ network hops, serialization, tail-latency amplification |
| Reliability | ● one deployable can fail as a whole; mitigated by separate runtimes for realtime/media/workers | ▲ blast-radius isolation *if* done well; ▼ many more partial-failure modes |
| Financial correctness | ▲ ACID transactions across checkout→reservation→payment intent | ▼ sagas everywhere; distributed consistency bugs are the #1 source of money errors |
| Security | ▲ fewer network surfaces and service credentials | ▼ service-to-service auth, mTLS mesh, secret sprawl |
| Scalability | ● scale whole API horizontally; separate runtimes for different load shapes | ▲ independent scaling per service |
| Infrastructure cost | ▲ few nodes, one DB cluster | ▼ per-service DBs, mesh, more observability volume |
| Developer complexity | ▲ one repo, one build, refactors are compiler-checked | ▼ contract versioning, local env with many services |
| Operational risk | ▲ simpler on-call | ▼ requires platform team from day one |

**Decision.** Modular monolith with enforced module boundaries (doc 04 §2.2), plus separate *runtimes* (`api`, `worker`, `realtime`, `media`) from the same repository. Extraction triggers are defined in doc 04 §9.

**Rejected:** microservices from day one (cost and correctness risk without a platform team); a “big ball of mud” monolith (no module discipline).

**Consequences.** Architecture lint is mandatory in CI. Cross-schema transactions are listed and limited. Admin NOC screens showing “microservices health” must be reinterpreted as runtime/module health.

---

## ADR-002 — Backend language: Go

| Criterion | **Go** | Kotlin/JVM (Spring/Ktor) | TypeScript/Node (NestJS) | Elixir/Phoenix | Rust |
|---|---|---|---|---|---|
| Performance | ▲ | ▲ (after warm-up) | ● | ● | ▲▲ |
| Concurrency for realtime | ▲ goroutines | ● virtual threads | ▼ single-thread event loop | ▲▲ BEAM | ▲ |
| Memory / infra cost | ▲ small static binaries | ▼ heavier JVM footprint | ● | ● | ▲ |
| Hiring (Iran market & general) | ▲ good | ▲ good | ▲▲ largest | ▼ small pool | ▼ small pool |
| Type safety for money code | ● (no sum types; mitigated with value types & tests) | ▲ | ● | ▼ dynamic | ▲▲ |
| Developer velocity | ▲ | ● | ▲ | ● | ▼ |
| Ecosystem for this system (Postgres, NATS, WebRTC/LiveKit, OTel) | ▲▲ (NATS and LiveKit are Go-native) | ▲ | ▲ | ● | ● |
| Operational risk | ▲ simple deploys | ● | ● | ● | ● |

**Decision.** Accept Go. Money is represented by a dedicated `Money{Amount int64, Currency}` value type with no float conversions; lint forbids `float64` in money packages.

---

## ADR-003 — Mobile: Flutter + Dart

| Criterion | **Flutter** | React Native | Native (Kotlin + Swift) | Kotlin Multiplatform + native UI |
|---|---|---|---|---|
| RTL & Persian text quality | ▲ mature `Directionality`, good shaping | ● good, more native-module variance | ▲▲ | ▲▲ |
| Video/camera/editor | ● plugins + some native code | ● same | ▲▲ | ▲ |
| One codebase | ▲▲ | ▲▲ | ▼▼ two teams | ● shared logic only |
| Performance (feeds, reels) | ▲ | ● | ▲▲ | ▲▲ |
| Team cost | ▲ | ▲ | ▼ | ▼ |
| Design-system consistency | ▲▲ pixel-identical across OS | ● | ▼ | ▼ |

**Decision.** Accept Flutter. Expect **platform channels** for camera capture, video editing, WebRTC, and PSP return handling; budget ~15–20% native work on media-heavy features.

**Amendment v3 (B-4).** Android is the first release; iOS must remain possible without rework. Constraints: (1) no hard dependency on Google Play Services — push through the ADR-021 abstraction, maps through a domestic provider, no mandatory Play Integrity/SafetyNet, in-app update check via VIRA’s own `config/bootstrap` (min-version gate) rather than Play In-App Updates; (2) every native plugin must have an iOS implementation or an iOS plan before adoption; (3) payment return uses universal links / app links on both platforms; (4) distribution channels (domestic Android stores, direct APK, Google Play, iOS options) are a business decision (D-16) and do not change the code base.

**Caveat (blocking for iOS, D-16):** distribution of an iOS app by an Iran-based entity is constrained by sanctions and App Store policy; Android distribution likely requires domestic stores. This is a business/legal question, not a framework question, but it decides whether iOS is in scope at launch.

---

## ADR-004 — Primary database: PostgreSQL

| Criterion | **PostgreSQL 16/17** | MySQL 8 | CockroachDB / YugabyteDB | MongoDB |
|---|---|---|---|---|
| ACID, serializable isolation for money | ▲▲ | ▲ | ▲▲ | ● (multi-doc tx exist, weaker fit) |
| Constraints, partial indexes, exclusion constraints, RLS | ▲▲ | ● | ● | ▼ |
| JSONB for flexible attributes (product specs) | ▲ | ● | ● | ▲▲ |
| Operational maturity, in-country expertise | ▲▲ | ▲▲ | ▼ | ● |
| Horizontal write scale | ● (partitioning, later Citus-class sharding) | ● | ▲▲ | ▲ |
| Cost | ▲ | ▲ | ▼ (licensing / resource overhead) | ● |

**Decision.** PostgreSQL, one cluster with **schema-per-module**, primary + **quorum-synchronous** standby in a second data centre of the same provider (Patroni-class HA, fencing) + asynchronous replica for reads/analytics, **plus an encrypted off-provider WAL archive and weekly base backup at a second domestic provider** (amended after audit F-18; doc 06 §11, doc 18 §5.2). pgBackRest (or equivalent) for PITR. Partitioning for messages, notifications, audit, outbox, feed items.

---

## ADR-005 — Cache & ephemeral state: Redis-compatible

Uses: response/object cache, rate-limit counters, OTP attempt counters, presence, short-lived idempotency fast-path, leaderboards. **Never** the only copy of anything that matters (sessions are in PostgreSQL; Redis is a cache in front).

**Licence note:** Redis changed licensing in 2024 (source-available) and later added an AGPL option; **Valkey** (Linux Foundation fork, BSD) is wire-compatible. Choose either in Phase 1 after licence review; code depends only on the protocol.

**Rejected:** Memcached (no data structures for rate limiting/presence).

---

## ADR-006 — Domain events: transactional outbox + NATS (JetStream deferred)

| Criterion | **Outbox + NATS JetStream** | Kafka / Redpanda | RabbitMQ | Redis Streams | PostgreSQL only (LISTEN/NOTIFY + polling) |
|---|---|---|---|---|---|
| Ops complexity | ▲ single small binary, 3-node cluster | ▼ (Kafka) / ● (Redpanda) | ● | ▲ | ▲▲ |
| Throughput headroom | ▲ | ▲▲ | ● | ● | ▼ |
| Replay / retention | ▲ streams with retention | ▲▲ | ▼ | ● | ● |
| Realtime pub/sub for WebSocket fan-out | ▲▲ core NATS subjects | ▼ | ● | ● | ▼ |
| Go ecosystem | ▲▲ | ▲ | ▲ | ▲ | ▲ |
| Cost | ▲ | ▼ | ● | ▲ | ▲▲ |

**Decision (amended after audit F-17).** Every module writes events to a **single `platform.outbox` table** (with a `module` column) **in the same transaction** as the state change. At Tier A, a dispatcher in `vira-worker` delivers events to in-process consumers through the PostgreSQL job queue (ADR-007); consumers are idempotent (inbox keyed by `(consumer, event_id)`). **Core NATS** (no persistence) carries realtime fan-out only. **JetStream** is introduced behind the same relay when a trigger in doc 04 §9 fires; producers do not change. Ordering: one dispatcher partition per aggregate (`hash(aggregate_id)`), and the envelope carries `aggregate_version` so consumers can park stale events.

**Why not JetStream on day one:** at Tier A (≈300 orders/day, 300 RPS) a third stateful system adds an on-call surface without a measurable benefit; the outbox design keeps the migration cost near zero.

**Rejected:** Kafka (operational weight unjustified at MVP scale); direct “publish after commit” without outbox (loses events on crash).

---

## ADR-007 — Background jobs: PostgreSQL-backed queue

Jobs that must be enqueued atomically with a business write (e.g. “inquire payment status at +3 min if no callback”, “expire checkout when its TTL passes and the payment is terminal”, “compute hold release”) use a PostgreSQL-backed job queue (River-class library for Go). Benefits: transactional enqueue, unique jobs, scheduled jobs, retries with back-off, visibility in SQL. At Tier A the same queue also dispatches outbox events to consumers (ADR-006). **Rejected:** cron + ad-hoc tables (no retries/visibility).

---

## ADR-008 — API style

* **External:** REST/JSON over HTTPS, OpenAPI 3.1 as source of truth, code-generated clients (Dart, TypeScript) and server stubs. Errors as RFC 9457 `application/problem+json` with stable `code`.
* **Realtime:** one WebSocket per client, JSON frames (protobuf later if bandwidth matters).
* **Internal:** in-process Go interfaces; no internal gRPC until a module is extracted.
* **Rejected:** GraphQL for MVP (authorization per field and caching complexity, N+1 risks; feed and commerce screens map well to purpose-built endpoints). Revisit for admin reporting only.

---

## ADR-009 — Media storage and processing

* S3-compatible object storage (provider-managed in-country, or MinIO-class self-hosted) with buckets: `quarantine` (private), `media-public` (CDN origin, signed URLs for private accounts), `kyc-evidence` (private, separate key, no CDN), `exports` (private, short TTL).
* Images: libvips via an imgproxy-class service; pre-generate 3 sizes, WebP/AVIF with JPEG fallback.
* Video: FFmpeg workers; H.264 ladder (240p–1080p) packaged as HLS (CMAF segments); poster frames; duration and size caps per content type.
* CDN: in-country CDN with signed URLs and token auth for non-public media.
* Moderation hooks: perceptual hash, NSFW classifier (vendor/self-hosted model, D-21), known-bad hash list.

**Amendment v3 (build-or-buy, evidence doc 18).** A domestic managed **video platform** (upload, transcoding to ABR, signed playback, live) is available with published per-hour pricing that is competitive at VIRA’s volumes (doc 17 §3). Decision: video transcoding and delivery go through a `VideoProvider` interface with two implementations — (a) the provider’s video platform (default for Scenarios A/B, lowest operating load), (b) the self-hosted FFmpeg/HLS pipeline above (fallback and exit path). Originals are always kept in VIRA-controlled object storage so the provider can be replaced. Images stay in the self-hosted libvips pipeline. Moderation hooks run on VIRA’s side before a video is published regardless of provider.

---

## ADR-010 — Live video and calls

| Use | Technology | Rationale |
|---|---|---|
| 1:1 voice/video calls | WebRTC, P2P first; TURN (coturn) relay | Lowest latency; TURN essential under restrictive NAT/firewalls |
| Group calls, co-hosting | WebRTC SFU (LiveKit-class, Go, self-hostable) | Scales to small rooms; recording hooks |
| Live shopping to many viewers | Host publishes via WebRTC/WHIP or RTMP → transcode → **LL-HLS via CDN** | CDN-scale fan-out; 2–5 s latency acceptable for shopping; WebRTC to thousands of viewers is far costlier |
| Live chat, product pins, purchases during live | `vira-realtime` WebSocket + normal checkout API | Reuses existing infra |

Phase 8 only. Cost model in doc 10 §5 and doc 17.

**Amendment v3.** For live shopping to many viewers, the provider’s live-streaming service (doc 18) is an alternative to self-run ingest/LL-HLS, behind the same `VideoProvider` interface; WebRTC for calls and co-hosting stays self-hosted (TURN + SFU). Selection at the P8 gate.

---

## ADR-011 — Financial ledger

**Decision.** Double-entry, append-only ledger in its own PostgreSQL schema. Amounts are `BIGINT` **Rial** (IRR minor unit); Toman is a display concern only (1 Toman = 10 Rial). Accounts carry `currency char(3)` with `CHECK (currency = 'IRR')` until a gated track is approved. Every journal entry balances to zero per currency, is immutable, and references a business event (`source_type`, `source_id`) with a unique constraint for idempotency. Balances are derived (materialised with versioned rows, verified nightly). Only accounts that need a non-negative guard (seller available / in-transit / receivable) are updated synchronously, locked in sorted account-ID order; platform aggregate accounts (`PSP_CLEARING`, `COMMISSION_REVENUE`, `TAX_PAYABLE`) are summed asynchronously to avoid hot rows. Serialization retries re-run the whole closure (max 5, jittered) and **never contain external calls**. Corrections are reversing entries, never updates. Details in doc 06 §6.

**Rejected:** balance columns on orders/wallets as the source of truth; floating-point; a third-party ledger SaaS (residency and availability).

**Status v3.** The ledger core (double entry, invariants, idempotency, reconciliation framework) is independent of the settlement model. The **settlement branch** — whether VIRA holds seller funds (platform-collects) or mirrors PSP split settlements (تسهیم) — is **blocked by the financial-model gate FG-1a** (doc 06 §6.5), which must pass before the P6 design freeze; real-money use is further gated by FG-1b. Both branches share the same chart-of-accounts core.

---

## ADR-012 — Hosting & deployment

* **In-country hosting** (residency, PSP IP allow-listing, and resilience to international connectivity disruption) — provider is D-17.
* **Amendment v3 (doc 18):** primary = a full-service domestic cloud (ArvanCloud is the only provider whose full service set and prices could be verified), spread across ≥ 2 of its Tehran data centres; recovery = a second, independent domestic provider (pilot light → warm standby by scenario). Final selection requires the vendor answers in doc 18 §6. ArvanCloud’s Cloud Container is documented as Kubernetes-based across three DCs, so the “managed Kubernetes of acceptable maturity” condition can be evaluated concretely in P1.
* Containers built once, promoted across environments. Infrastructure as code (OpenTofu/Terraform where the provider supports it; Ansible otherwise). GitOps deployment.
* **Kubernetes only if a managed offering of acceptable maturity exists** at the chosen provider; otherwise a fixed fleet of VMs with a simpler scheduler (Nomad or systemd + Docker). Decision record to be finalised in Phase 1 with a short bake-off.
* Databases on dedicated VMs/bare metal with local NVMe, not in Kubernetes, for MVP.

---

## ADR-013 — Flutter state management (deferred)

Riverpod and Bloc are both acceptable. Phase 1 mobile lead chooses using criteria: testability, code-generation needs, team familiarity, offline outbox integration. Choice must be one, documented, and lint-enforced.

---

## ADR-014 — Typography and icon assets

The mobile DESIGN.md files specify Latin families (Noto Sans + Plus Jakarta Sans; Bricolage Grotesque + DM Sans) and 166 of 277 mobile screens load no Persian font. **Decision:** Vazirmatn (SIL OFL) is the single UI family for Persian and Latin in both mobile and web, bundled in the app and self-hosted for web. Optional Latin display face only for brand marks. Material Symbols are self-hosted (subset). No Google Fonts or Tailwind CDN at runtime.

---

## ADR-015 — Search

* MVP: PostgreSQL with a **Persian normalisation function** (Arabic ي/ك → Persian ی/ک, remove tatweel and diacritics, unify ZWNJ variants, Persian/Arabic digits → ASCII) applied on write and query, `pg_trgm` for fuzzy match.
* Phase 4: OpenSearch (Apache-2.0) with the Persian analyser + custom char filters and synonyms, fed by outbox events; Postgres remains the source of truth.
* Rejected: Elasticsearch (licence), Meilisearch (weaker Persian morphology; acceptable fallback).

---

## ADR-016 — Super Admin isolation

Separate **binary** (`vira-admin-api`, own `main`) and deployment for admin routes, separate DB role, separate DNS name not publicly resolvable (or behind VPN), separate auth realm (admin users are **not** platform users), WebAuthn or TOTP mandatory, session lifetime ≤ 8 h with 15-min idle timeout, all actions audited. Seller web is public but its own origin and realm scope.

---

## ADR-017 — Authorization model

* **Admin:** RBAC with fine-grained permissions (`orders.refund.approve`, `kyc.evidence.view`), roles composed of permissions, plus **attribute conditions** (amount thresholds, region) and maker–checker rules.
* **Seller:** tenant ABAC — every seller-scoped request carries `store_id`; permission = staff role on that store. PostgreSQL **Row-Level Security** on seller-scoped tables as defence in depth (`SET LOCAL app.store_id`).
* **Consumer:** relationship-based checks (owner, follower, blocked, close friend) in the social-graph module.
* **Amendment v3 — shared seller identity (owner requirement).** One user account; seller capability is granted on approval; the same person may be owner or staff of one or more stores. Store membership and role are evaluated **per request** with the `X-Store-Id` context on both surfaces (mobile “Manage My Stall” and Seller Admin web), against the same permission table (doc 08 §4.2) and the same RLS policies. Seller Admin web uses the same identity and OTP, with its own browser session (httpOnly, SameSite=strict cookie, CSRF token, 12 h max / 30 min idle) and mandatory TOTP step-up for owner/finance actions. Detail in doc 04 §10.
* Policy code is in-process Go with table-driven tests. **Rejected for MVP:** OPA/Rego or Cedar as external engines (extra moving parts); reconsider if policy authoring moves to non-engineers.

---

## ADR-018 — Identifiers

UUIDv7 primary keys (time-ordered, index-friendly). Public IDs exposed to clients are the same UUIDs or opaque prefixed tokens; **never sequential integers** (enumeration risk, `IDOR`). Human-facing references are separate: order `VR-` + Crockford Base32 (as the UI shows `#VR-884920`), return `RET-…`, ticket `TKT-…`.

---

## ADR-019 — Wallet, crypto and other gated financial products

No buyer stored-value top-up and no crypto functionality is designed in detail or implemented until D-03/D-04 are approved. The architecture reserves: (a) a `PaymentRail` interface in payments (card/IPG is the only implementation), (b) currency as a ledger dimension (IRR only enabled), (c) feature flags that default OFF and are checked **server-side**. Any future crypto design will require a new Phase-0-style review (custody, travel-rule, sanctions, chain monitoring, key management).

**Amendment v3 (B-2).** The owner keeps **wallet functionality, USDT payments, USDT↔Toman conversion in both directions, and USDT withdrawals for eligible sellers** in the long-term roadmap (gated tracks G1/G2 in doc 12). The architecture therefore reserves, without implementing: (1) a `PaymentRail` interface (card IPG implemented; USDT rail reserved); (2) currency as a ledger dimension (`IRR` only enabled; `USDT` reserved, with its own clearing, custody-liability and FX accounts designed at the gate); (3) a `ConversionService` interface for quoted, time-limited conversions with rate locks and limits; (4) a `PayoutRail` interface (Paya/Satna/PSP implemented; USDT reserved); (5) eligibility rules (KYC tier, seller tier) as configuration. **Disabled by default at build, API, ledger and UI layers** (doc 08 §7). Activation of each track requires its own design, legal/licensing, custody and security approvals and the owner’s written authorisation; UI screens for these tracks stay in the reference archive as future references.

---

## ADR-020 — Observability

OpenTelemetry SDK in all runtimes. Self-hosted Prometheus (or VictoriaMetrics), Grafana, Loki, Tempo; GlitchTip/Sentry-compatible self-hosted crash reporting for Flutter. Rationale: in-country residency, no dependency on foreign SaaS. SLOs and alerts in doc 10 §7.

---

## ADR-021 — Push notifications

A `PushProvider` abstraction with: FCM (Android with Play services), APNs (iOS, if in scope), and at least one domestic push provider for devices without Play services or when international routes are degraded. The in-app WebSocket delivers notifications when the app is foregrounded. Push payloads carry **no sensitive content** (IDs only; client fetches details).

---

## ADR-022 — Admin and seller web: React + TypeScript

| Criterion | **React + TS** | Vue 3 + TS | Angular | Server-rendered (Go templates + htmx) |
|---|---|---|---|---|
| Data-dense admin tables, RTL libraries | ▲▲ | ▲ | ▲ | ● |
| Hiring | ▲▲ | ▲ | ● | ● |
| Shared types from OpenAPI | ▲▲ | ▲▲ | ▲▲ | ▼ |
| Security surface | ● (SPA + API) | ● | ● | ▲ |

**Decision.** Accept React + TypeScript for both web apps with a shared, RTL-first component library.
