# 10 — Infrastructure, Scalability and Cost Analysis

**Status:** Phase 0 v3. **Monetary estimates and provider evaluation now live in doc 17 (budget & team scenarios, Toman list-price model) and doc 18 (Iranian hosting providers).** This document keeps the sizing tiers, cost drivers, SLOs and DR objectives that those documents build on.

## 1. Environments

| Env | Purpose | Data | Payments | Notes |
|---|---|---|---|---|
| `local` | Developer machine | Synthetic | PSP simulator | Docker Compose: Postgres, Redis, NATS, MinIO, mailhog-style SMS sink |
| `dev` | Integration of merged code | Synthetic | PSP simulator | Auto-deploy from `main` |
| `staging` | Production-like, release candidates, load tests | Synthetic + anonymised samples | **PSP sandbox** — this is where the “test/demo payment” screens (`_76`, `_92`, `_100`, `_153`) belong | Same topology as prod at smaller scale |
| `prod` | Production | Real | Live PSP | Separate cloud project/account, separate secrets, separate admins |
| `restore-drill` | Monthly restore verification | Restored prod copy (access-restricted) | Disabled | Torn down after checks |

## 2. Hosting constraints that shape everything

1. **In-country hosting** is assumed (D-01/D-17): PSPs commonly require in-country server IPs, residency rules are likely, and latency to users matters.
2. **Resilience to international connectivity disruption.** The product must keep working if routes abroad degrade. Therefore **no runtime dependency on foreign services**: fonts, icons, CSS frameworks, images, maps, push, crash reporting, SMS, KYC, AI must each have an in-country path or graceful degradation.
3. **Build-time resilience.** CI depends on Go module proxy, pub.dev, npm, container registries. Run **in-country mirrors/caches** (Go module proxy, npm and pub mirrors, container registry mirror) so builds and hotfixes are possible during disruptions.
4. **Maps:** address pinning and nearby stores need a domestic map SDK/tiles provider (to be evaluated, e.g. Neshan-class providers) rather than Google Maps.
5. **Two failure domains minimum** (two data centres/availability zones of the same provider, or two providers) for database standby and object-storage replication.

## 3. Capacity tiers (planning assumptions — D-14 to confirm)

| Metric | Tier A — Launch | Tier B — Growth | Tier C — Scale |
|---|---|---|---|
| DAU | 10k | 100k | 1M |
| Peak concurrent app sessions | 1k | 10k | 100k |
| Peak API RPS | 300 | 3,000 | 30,000 |
| Reels uploaded / day | 500 | 5,000 | 50,000 |
| Image posts / day | 2k | 20k | 200k |
| Video watched / DAU / day | 15 min | 15 min | 15 min |
| Orders / day | 300 | 3,000 | 30,000 |
| Messages / day | 50k | 1M | 15M |
| Concurrent live sessions (peak) | — (Phase 8) | 20 × 200 viewers | 200 × 500 viewers |

### 3.1 Derived load

**Video egress** (dominant cost). At an average delivered bitrate of ~1 Mbps (480p–720p mobile ladder):
15 min × 60 s × 1 Mbit/s = 900 Mbit ≈ **112 MB per DAU per day**.

| Tier | Egress / day | Egress / month |
|---|---|---|
| A | ≈ 1.1 TB | ≈ 34 TB |
| B | ≈ 11 TB | ≈ 340 TB |
| C | ≈ 112 TB | ≈ 3.4 PB |

Live (Tier B peak): 4,000 viewers × 1.5 Mbps ≈ 6 Gbps while live ≈ 2.7 TB per peak hour.

**Media storage growth** (Tier B): reels 5k × ~50 MB (mezzanine + ladder) ≈ 250 GB/day; images 20k × 3 × ~1.5 MB (all variants) ≈ 90 GB/day → **≈ 10 TB/month**. Lifecycle: delete mezzanine after 30 days unless flagged; cold tier after 180 days.

**Transcoding** (Tier B): 5,000 clips × ~100 vCPU-seconds (30 s clip, 4-rung ladder, x264 `veryfast`) ≈ 140 vCPU-hours/day ≈ 6 vCPU average, ~24 vCPU at peak. GPU transcoding not needed until Tier C.

## 4. Resource sizing (indicative)

| Component | Tier A | Tier B | Tier C |
|---|---|---|---|
| `vira-api` | 3 × 4 vCPU / 8 GB | 8 × 8 vCPU / 16 GB | 40+ × 8 vCPU, autoscaled |
| `vira-worker` | 2 × 4 vCPU | 4 × 8 vCPU | 16+ |
| `vira-realtime` | 2 × 2 vCPU / 4 GB | 3 × 8 vCPU / 16 GB (≈ 50k conns/node) | 10+ nodes |
| `vira-media` | 2 × 8 vCPU | 4–6 × 16 vCPU | 30+ / GPU pool |
| PostgreSQL | Primary 8 vCPU / 32 GB / 1 TB NVMe + sync standby + async replica | 32 vCPU / 128 GB / 4 TB + replicas; PgBouncer | Split clusters: money · social · messaging; partitioned/sharded social & messaging |
| Redis/Valkey | 1 primary + replica, 8 GB | 3-node, 32 GB | Clustered |
| NATS (core at A; JetStream when triggered) | 3 × 2 vCPU | 3 × 4 vCPU | 5 × 8 vCPU |
| Object storage | 5 TB | 150 TB year-1 | PB-class |
| Search (OpenSearch) | — (Postgres) | 3 × 8 vCPU / 32 GB | 6+ |
| Observability stack | 3 × 4 vCPU | 6 × 8 vCPU | dedicated cluster |
| TURN / SFU (Phase 8) | — | 2–4 × 8 vCPU, high bandwidth | regional pools |

## 5. Cost structure

### 5.1 Relative cost drivers at Tier B (excluding staff) — v3: shares from the doc 17 model

| Driver | Share of infra + third-party spend (estimate) | Levers |
|---|---|---|
| CDN / video egress | **≈ 27 %** in the doc 17 model at Tier B (video delivery via a managed platform + CDN); **≈ 46 %** including all media storage; rises to 50–70 % if video is self-delivered through CDN overage | Default 480p on cellular, per-title ladders, HEVC/AV1 where devices allow, prefetch limits, cache hit ratio > 95% |
| Compute (API, workers, media) | ≈ 28 % at Tier B list prices (doc 17) | Go efficiency, autoscaling, batch transcoding off-peak |
| Storage (object + DB) | ≈ 25–30 % at Tier B list prices (object storage, DB local disk, block storage) | Lifecycle rules, mezzanine deletion, cold tier |
| SMS (OTP + transactional) | 5–15% | Long-lived device sessions, in-app/push for non-OTP notices, SMS only when needed |
| KYC checks (per verification) | variable | Verify only when a limit is hit (progressive KYC) |
| PSP fees | not infra, but material | Negotiated; ledger tracks precisely |
| Live (when enabled) | can rival VOD egress during events | Viewer caps, bitrate caps, schedule-based capacity |

### 5.2 Cost implications of architecture choices

| Choice | Cost effect |
|---|---|
| Modular monolith vs microservices | ~3–5× fewer always-on instances at Tier A/B; one DB cluster instead of many |
| HLS via CDN for live viewers (vs WebRTC to every viewer) | Order-of-magnitude lower per-viewer cost; slightly higher latency |
| Self-hosted observability (vs foreign SaaS) | Infra cost + ops effort, but required for residency/resilience |
| OpenSearch deferred to Phase 4 | Saves a 3-node cluster during MVP |
| No microservice mesh | Avoids sidecar overhead (~10–20% CPU) |

**Staff dominates total cost of ownership** through Tier B. The cheapest architecture is the one a small team can operate without a dedicated platform group — the main reason for ADR-001.

## 6. Scalability plan & bottlenecks

| Bottleneck | Early signal | Mitigation path |
|---|---|---|
| PostgreSQL write throughput (reactions, messages) | WAL rate, lock waits, replica lag | Async counters; partitioning; move messaging to its own cluster; then distributed store if needed |
| Feed fan-out for large accounts | Fan-out job lag | Hybrid push (normal accounts) / pull-at-read (accounts with > 50k followers) |
| Hot products during live or flash sale | Inventory row contention | Per-variant serialisation; pre-split stock buckets for flash sales; queue-based checkout admission |
| Realtime connection storms after outage | Reconnect spike | Jittered reconnect, connection admission control, resume tokens |
| CDN cache misses on new reels | Origin egress spike | Origin shield, pre-warm popular items |
| PSP throughput / outages | Verify latency | Second PSP — **launch-gate item** at P7 exit if the PSP contract cannot show ≥ 99.9% availability; otherwise P10; routing by health |
| SMS throughput | Queue lag | Second provider, priority queue for OTP |

## 7. Monitoring, SLOs and alerting

| SLO | Target | Alert |
|---|---|---|
| Checkout API availability (`/checkout*`), **excluding** time when the PSP itself is unavailable (PSP availability tracked as its own SLI) | 99.9% monthly (from P7 launch) | Burn rate 2% in 1 h → page |
| Payment verification success (excluding user cancels) | ≥ 99% of callbacks verified within 60 s | < 97% over 15 min → page |
| Payments stuck in `unknown` > 30 min | 0 | any → page finance on-call |
| Ledger integrity | 100% | any break → SEV1 |
| API p95 latency | < 300 ms read / < 600 ms write | sustained 10 min breach → ticket; 30 min → page |
| Message delivery p95 | < 1 s | breach 15 min → page |
| Media processing p90 (60 s reel) | < 60 s | queue age > 5 min → page |
| Outbox relay lag | < 10 s | > 60 s → page |
| OTP delivery success | ≥ 95% within 30 s | < 90% → switch provider + page |

Dashboards per module (RED metrics), per runtime (USE metrics), business (orders/min, GMV, verify rate, refund rate), security (doc 08 §11).

## 8. Disaster-recovery objectives

| Service tier | Examples | RPO | RTO |
|---|---|---|---|
| Tier 0 — Money | payments, ledger, settlement | ≈ 0 for committed transactions (synchronous standby); ≤ 5 min worst case | 1 h |
| Tier 1 — Commerce | orders, catalog, checkout, identity | ≤ 5 min | 2 h |
| Tier 2 — Social & messaging | posts, comments, messages | ≤ 15 min | 4 h |
| Tier 3 — Derived | feed cache, search index, analytics | rebuildable | 24 h |
| Media | object storage | ≤ 24 h | 24 h (CDN serves cached content meanwhile) |

DR model (v3, aligned with doc 18 §5.2 and doc 06 §11):

* **Data-centre loss (same provider):** warm standby — PostgreSQL quorum-sync standby and application capacity in a second data centre of the primary provider; RTO ≤ 1 h for Tier 0. Failover rehearsed twice a year.
* **Provider loss:** recovery at a second, independent domestic provider — **pilot light** in Scenarios A/B (RTO ≤ 24 h), **warm standby** in Scenario C (RTO ≤ 4 h); RPO ≤ 1 min for ledger data from the off-provider WAL archive. Full restore at the recovery provider rehearsed twice a year.
* Infrastructure is reproducible from IaC within the RTO.

## 9. Release infrastructure

Container registry (in-country) with signed images; GitOps controller or deployment pipeline; blue/green for `vira-api`, rolling for workers, connection-draining for `vira-realtime`; database migrations run as a separate, gated pipeline step (doc 11 §5).
