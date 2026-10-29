# 18 — Iranian Hosting Provider Comparison

**Status:** Phase 0 v3 · 2026-10-29 · **Evaluation only — nothing was purchased, deployed or configured.**
**Decision owner:** product owner (B-5 / D-17). This document recommends a strategy and the evidence still required; it does not select a vendor contractually.

## 1. Evidence rules

| Code | Meaning |
|---|---|
| **V1** | Verified from the vendor’s own documentation or price page, read directly during this review (sources in `evidence/sources.md`) |
| **S+** | Third-party machine-readable snapshot of the vendor’s official price page (dated, commit-pinned, copy in `evidence/`), internally consistent with the vendor’s directly-read EUR price pages — to be confirmed by vendor quote |
| **V2** | Product *existence* verified from the vendor’s own domain (page title / URL), but specifications not readable (most Iranian vendor sites block automated reading) |
| **S** | Secondary source (independent article, aggregator) — indicative only |
| **U** | Unknown — must be obtained from the vendor (see §6 questions) |

**Not claimed:** SLA percentages, certifications, data-centre Tier levels, support quality, or real-world incident history for any vendor — none could be verified. An independent 2026 review found that **no Iranian data centre could be confirmed in the Uptime Institute’s public certification list**; any “Tier III” claim must be treated as self-declared until a certificate (issuer, facility, date, stage) is produced.

Toman prices come from a **third-party snapshot (S+)** of ArvanCloud’s official “all products” price page dated **2026-10-10** (`github.com/saqi-sn/arvan-pricing-all`, commit `c854cbc`, unofficial; the vendor site blocks automated clients). ArvanCloud’s **English price pages (EUR) for cloud servers and the video platform were read directly (V1)** and show the same price structure. The snapshot lists dated price versions (e.g. 1 Tir 1405, 1 Mehr 1405), i.e. **prices are revised roughly quarterly**. Treat every number as a list price for planning, not a quote.

## 2. Providers evaluated

| Provider | Type | Evidence level |
|---|---|---|
| **ArvanCloud (ابر آروان)** | Full public cloud: compute, Kubernetes-based containers, managed DB, object storage, CDN/WAF, video VOD/live, logs | V1 (EUR price pages, container multi-zone docs) + S+ (Toman prices) + V2 (DB engine docs) + S |
| **Sotoon (ستون)** | Public cloud (compute, Kubernetes, database, object storage, DevOps services, AI studio) | V2 + S |
| **Hamravesh (هم‌روش) / Darkube** | Kubernetes PaaS (Darkube) + managed database (DBaaS); public status page | V2 |
| **Liara (لیارا)** | Developer PaaS: app hosting, PostgreSQL / MySQL / Redis / Elasticsearch “hourly cloud hosting”, object storage | V1 (object storage) + V2 |
| **ParsPack (پارس‌پک)** | Cloud servers, hosting, CDN, SSL, PaaS, S3 object storage (Iran + Europe) | V1 (object storage docs, product nav) + V2 |
| **Abramad (ابرآمد, Systemgroup)** | Enterprise managed cloud: cloud servers, object storage, Kubernetes services | V2 |
| **Derak Cloud (ابر دراک)** | CDN, cloud storage, cloud servers | V2 |
| **Kubit** | Managed Kubernetes (KaaS) — not shown in the matrix: only an aggregator listing was available | S |
| **Colocation operators** (Asiatech — Milad Tower, Afranet, MCI, Irancell, Shatel, Respina) | Racks / dedicated servers in Tehran | S |

## 3. Comparison matrix

Legend: ● available (verified V1) · ◐ exists (V2/S, specs unknown) · ○ not found · ? unknown.

| Capability VIRA needs | ArvanCloud | Sotoon | Hamravesh/Darkube | Liara | ParsPack | Abramad | Derak | Colocation |
|---|---|---|---|---|---|---|---|---|
| Go app hosting (VM) | ● cloud server, per-vCore/RAM pricing | ◐ compute | ◐ (containers) | ◐ PaaS | ◐ servers | ◐ cloud server | ◐ cloud server | ● own hardware |
| Managed Kubernetes / containers | ● Cloud Container on Kubernetes across **3 data centres (Bamdad, Simin, Forough)**, multi-zone pod spreading, automatic load shift if a DC fails | ◐ Kubernetes product | ◐ Darkube (Kubernetes PaaS) | ◐ PaaS (not raw K8s) | ○ not in product nav | ◐ (articles) | ? | self-run |
| Managed PostgreSQL | ◐ DBaaS PostgreSQL (docs + news, V2); price list per vCore/RAM/disk (S+); **backup free only up to 7 days**; HA/PITR **U** | ◐ database product | ◐ DBaaS + Darkube PostgreSQL docs | ◐ PostgreSQL hourly | ? | ? | ? | self-run |
| Redis | ? (not confirmed) | ? | ? | ◐ Redis hourly | ? | ? | ? | self-run |
| Object storage (S3 API) | ● graduated price per GB; read/write request pricing; antivirus scanning option | ◐ | ? | ● S3-compatible, SSD, multi-disk, “all services in Iran” | ● S3 protocol, multi-tenant, ≤100 buckets, public/private, custom domain | ◐ | ◐ | self-run (MinIO) |
| CDN / WAF / DDoS | ● CDN plans, per-GB over fair use, WAF, L4 proxy, logs, metrics | ? | ? | ? | ◐ CDN | ? | ◐ CDN | — |
| Video VOD / live | ● Video Platform: storage per hour, **watch-time per hour**, live per hour, ABR outputs | ? | ? | ? | ? | ? | ? | — |
| Background workers | ● (VMs/containers) | ◐ | ◐ | ◐ | ◐ | ◐ | ◐ | ● |
| Realtime (WebSocket, long-lived) | ● on VMs; L4 proxy offered | ◐ | ? (PaaS ingress limits **U**) | ? | ◐ | ◐ | ? | ● |
| Private networking | ? | ? | ? | ? | ? | ? | ? | ● |
| Backups / DR | ● backup & snapshot pricing; multi-zone | ? | ◐ backup guide | ? | ? | ◐ (managed) | ? | self-run |
| Monitoring / logging | ● Cloud Log, CDN metrics | ? | ◐ status page | ? | ? | ? | ? | self-run |
| Security controls | ◐ WAF, DDoS, rate-limit (S) | ? | ? | ? | ? | ? | ◐ | physical control |
| Bandwidth cost transparency | ● published per-GB tiers | ? | ? | ? | ? | ? | ? | negotiated |
| Infrastructure-as-code | ◐ API, CLI, Terraform (S) | ? | ? | ? | ? | ? | ? | — |
| Second region / city | Europe region for servers (**not usable for regulated data**); Iranian DCs all Tehran-area (**U**) | ? | ? | ? | Iran + Europe storage | ? | ? | Asiatech also lists Tabriz (S) |
| SLA, support quality, certifications | **U** | **U** | **U** | **U** | **U** | **U** | **U** | **U** |

### 3.1 ArvanCloud unit prices used in doc 17 (Toman, S+ snapshot of 2026-10-10)

| Item | Price |
|---|---|
| Cloud server vCore / month (Iran: Basic · Standard · Premium · Premium Plus) | 252,000 · 327,600 · 478,800 · 592,200 |
| RAM per GB / month (Iran: Basic · Standard · Premium) | 196,200 · 255,060 · 372,780 |
| Block storage per GB / month (Hot-SSD · Local · Warm) | 18,000 · 25,200 · 8,640 |
| Server egress per GB (Iran) | first 250 GB free · 1,800 (to 10 TB) · 1,200 (10–100 TB) · 1,050 (>100 TB); inbound free |
| Managed DB per vCore / month (Starter · Basic · Standard · Premium) | 264,600 · 579,600 · 819,000 · 1,292,760 |
| Managed DB RAM per GB / month (same tiers) | 206,010 · 451,260 · 637,650 · 1,006,510; SSD disk 36,000/GB; backup free **up to 7 days**; traffic listed free |
| Cloud Container per vCore · per GB RAM / month | 655,200 · 510,120 |
| Object storage per GB / month | 3,600 (≤5 TB) · 3,060 (5–50 TB) · 2,700 (50–500 TB) · 2,340 (>500 TB) |
| Object storage download per GB | 965 (≤20 TB) · 820 (20–200 TB) · 675 (0.2–2 PB) |
| CDN plans / month | Basic free · Growth 3.6 M · Professional 36 M · Enterprise 144 M; traffic above fair use 1,500/GB (fair use 8 / 50 / 500 GB per day) |
| Video platform watch time per hour | 900 (≤20k h) · 495 (20k–100k) · 360 (100k–1M) · 225 (>1M); storage 18,000 → 10,800 per stored hour; live 108,000 → 54,000 per streamed hour |

Units for compute are per month as laid out on the page (“per vCore”, “per GB”); confirm the billing period and the Iran/abroad region of each tier with the vendor.

## 4. Assessment

| Criterion | Finding |
|---|---|
| Breadth for VIRA’s stack | **ArvanCloud** is the only provider for which every building block could be evidenced: compute, Kubernetes-based containers across 3 DCs and video VOD/live (V1), object storage, CDN and logging (S+ prices), managed PostgreSQL (V2 existence, S+ prices; HA/PITR unknown). Sotoon and Hamravesh appear to cover compute/Kubernetes/DB/object storage but could not be verified. Liara is a developer PaaS (good for dev/staging and Scenario A side services). ParsPack, Derak and Abramad lack verified Kubernetes/DB depth. |
| Video economics | Arvan’s **video platform** watch-time pricing is cheaper than self-delivery via CDN overage at VIRA’s volumes — e.g. Scenario B ≈ 292 M vs ≈ 506 M Toman/month (doc 17 §3) and removes the most operations-heavy part of the media pipeline — evidence for amending ADR-009 to a build-or-buy decision (doc 05 §ADR matrix). |
| Concentration risk | ArvanCloud documents three named DCs (Bamdad, Simin, Forough) but not their cities (**U**); market reporting places most Iranian data centres in Tehran (S). Assume a **Tehran-area concentration** until vendors confirm locations; a city-level or provider-level outage needs a second provider, preferably with a facility outside Tehran. |
| Lock-in | Avoid provider-proprietary APIs in application code: use VMs/Kubernetes, S3 API, standard PostgreSQL, standard HLS. The video platform is the only proprietary dependency proposed, and it sits behind a `VideoProvider` adapter with the self-hosted FFmpeg pipeline as fallback. |
| Regulatory | Providers’ own legal status, data-handling terms, lawful-access obligations and any sanctions exposure require counsel’s due diligence (L-11). Europe regions must **not** hold regulated user data (B-1). |

## 5. Recommended strategy

### 5.1 Primary: **ArvanCloud** (conditional on vendor due diligence, §6)

* Compute on cloud servers or the Kubernetes-based Cloud Container, spread across **at least two of the three Tehran DCs** (Bamdad, Simin, Forough).
* PostgreSQL: **self-managed on cloud servers with local disks** (Patroni-class, quorum sync across two DCs; ADR-004, doc 06 §11) **unless** the managed DB proves — in writing — quorum-sync HA across zones, PITR, superuser-equivalent control for RLS/roles, and audit access. Managed DB is attractive in Scenario A for the lower operating load.
* Object storage (S3 API) for media; CDN for images/static/API edge; **video platform** for VOD transcoding and delivery (and live in P8), behind the `VideoProvider` adapter.
* WAF/DDoS at the CDN edge; admin network isolated from public ingress.

### 5.2 Recovery: **second, independent domestic provider** (Sotoon or Hamravesh, whichever passes §6; Abramad as an enterprise alternative)

| Element | Design |
|---|---|
| Data | Continuous encrypted WAL archive + weekly base backups shipped to the recovery provider’s object storage (ADR-004 amended, doc 06 §11); daily object-storage sync of media originals |
| Mode | **Pilot light** in Scenarios A/B (IaC ready, minimal always-on footprint); **warm standby** in Scenario C |
| Targets | Provider-loss RTO ≤ 24 h (A/B), ≤ 4 h (C); RPO ≤ 1 min for ledger data from the WAL archive |
| Location | Prefer a facility outside Tehran if offered |
| Drill | Full restore at the recovery provider twice a year |

### 5.3 Development / staging

Staging mirrors production on the primary provider. Developer sandboxes may use a PaaS (e.g. Liara) where convenient — **synthetic data only**.

## 6. Missing evidence and vendor questions (send before contracting)

1. Written **SLA** per service (compute, K8s, managed DB, object storage, CDN, video), credits, and exclusions.
2. Data-centre locations (city, facility), **independent certifications** (Uptime, ISO 27001, others) with certificate copies.
3. Managed PostgreSQL: versions; HA topology (sync/quorum, cross-DC); PITR window; backup location; failover time; access to `pg_hba`, roles, extensions (`pg_trgm`, `pgcrypto`), RLS, logical replication; maintenance windows; encryption at rest; customer-managed keys.
4. Redis/Valkey availability (managed or not).
5. Private networking between VMs/containers across DCs; bandwidth and latency between DCs; egress pricing between zones.
6. Kubernetes: version policy, control-plane SLA, node pools, storage classes (local NVMe), load balancers, network policies, ingress for long-lived WebSockets (timeouts, connection limits).
7. Object storage: durability claim, replication across DCs, versioning, object lock/WORM (for audit sealing, doc 06 §8), lifecycle rules, server-side encryption.
8. Video platform: API for upload/transcode/signed playback, DRM/token auth, webhooks, ABR ladder control, latency for live (LL-HLS), export of originals (exit plan).
9. Security: DDoS capacity, WAF rules, audit logs of console access, MFA for console, IAM granularity, incident notification commitments, lawful-access process and notification policy.
10. Support: 24/7 channels, response times by severity, named account manager, escalation path.
11. Price stability: revision frequency, notice period, fixed-price or committed-use contracts.
12. Exit: data export formats and fees, retention after termination.
13. Billing: Toman invoicing, official invoices for tax (e-invoicing), payment terms.

## 7. Operability by budget scenario (see doc 17)

| Scenario | Operating model on the primary provider | Can the team run it reliably? |
|---|---|---|
| A — Lean MVP | Managed DB (if §6.3 passes), containers or a few VMs, video platform, CDN, object storage; minimal self-hosted observability | **Yes, with one DevOps/SRE**, provided managed services are used and on-call is shared |
| B — Balanced | Self-managed PostgreSQL HA, Kubernetes (Cloud Container or self-run on VMs), self-hosted observability, pilot-light recovery | **Yes, with two platform/SRE engineers** and a security engineer |
| C — Growth | Split DB clusters, OpenSearch, larger K8s, warm standby at second provider, enterprise contracts | **Yes, with a platform team (4–6)** and 24/7 on-call |
