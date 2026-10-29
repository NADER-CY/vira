# 17 — Budget and Team Scenarios

> **v4 — superseded for development staffing.** The owner has decided that Claude builds VIRA with no development team hired (doc 20). The team sizes, people costs and durations below no longer apply to development. The **infrastructure resource models and unit-price analysis remain valid** as the reference for sizing production at gate L and later scaling (P10). Development infrastructure is in doc 20 §3.

**Status:** Phase 0 v3 · 2026-10-29 · Responds to owner decision **B-3 (scenario analysis required)**. No budget, traffic target or team size has been fixed by the owner; nothing here assumes the earlier proposed team is available.

## 1. How to read the numbers

* **Infrastructure costs** are computed bottom-up: a resource list per scenario × ArvanCloud Toman list prices taken from a **third-party machine-readable snapshot of ArvanCloud’s official price page dated 2026-10-10** (`github.com/saqi-sn/arvan-pricing-all`, commit `c854cbc`; copy in `evidence/`). The snapshot is unofficial; its figures were spot-checked for internal consistency and are **list-price planning estimates, not quotes** — confirm with a vendor quote. The calculation is reproducible: `python3 evidence/cost_model.py`.
* ArvanCloud revises prices roughly **quarterly**; budgets should carry a **30–40 % contingency** and be re-priced each quarter.
* **People costs could not be verified** (Iranian salary surveys for 1405 exist but could not be retrieved). They are therefore expressed as **headcount × C**, where **C = average fully-loaded monthly cost per person in that role group**, to be filled from HR or recruiter data. A sensitivity example is given with clearly hypothetical C values.
* SMS, KYC checks, PSP fees, push and map APIs are **per-unit third-party costs** whose prices were not verified; volumes are given so the owner can price them.
* Delivery ranges assume work starts after Phase 1 authorisation and exclude waiting time on external approvals (PSP contract, licences, FG-1a/FG-1b).

## 2. Scenario summary

| | **A — Lean MVP** | **B — Balanced Production** | **C — Growth-Ready Platform** |
|---|---|---|---|
| Purpose | Launch social + marketplace with minimum spend and operations | Production-grade launch with sustainable operations | Fast parallel delivery, high concurrency, fault isolation |
| Engineering headcount | **8** | **17** | **35** |
| Non-engineering operations (FTE, excl. owner) | ≈ 2–2.5 | ≈ 7.5–8.5 | 20–28 |
| Design traffic tier (doc 10 §3) | Tier A (≈10k DAU) | Tier B (≈100k DAU), launch at A | Tier C (≈1M DAU) |
| Infra at design load (Toman / month, list price; add 30–40 % contingency) | **≈ 0.26 billion** (≈ 0.26–0.32 B with managed DB) | **≈ 1.57 billion** at Tier B load; **≈ 0.94 B** at launch traffic | **≈ 8.75 billion** (enterprise discounts likely) |
| MVP build P1→P7 incl. FG-1b activation (excl. external waits) | 14.5–19 months | 10.5–13.5 months | 9–10 months |
| Operating model | Managed services, single DevOps | Self-managed DB HA, Kubernetes, 2 SRE | Platform team, split clusters, warm DR |
| Main risk | Key-person dependency, slow, limited 24/7 coverage | Cost discipline, hiring time | Coordination overhead, burn rate before product-market fit |

## 3. Infrastructure cost detail (monthly recurring, Toman, list price)

Resource lists (production + staging allowance) and unit prices are in `evidence/cost_model.py`; its output is reproduced below.

| Line item | A | B (Tier B load) | B (launch traffic) | C |
|---|---:|---:|---:|---:|
| Compute vCores (incl. staging) | 84 | 410 | 410 | 1,507 |
| Compute RAM GB (incl. staging) | 254 | 1,210 | 1,210 | 4,589 |
| Compute — vCore | 27.5 M | 134.3 M | 134.3 M | 493.8 M |
| Compute — RAM | 64.9 M | 308.6 M | 308.6 M | 1,170.4 M |
| Database local disk | 37.8 M | 151.2 M | 151.2 M | 756.0 M |
| Block storage (SSD) | 36.0 M | 180.0 M | 180.0 M | 720.0 M |
| Public IPv4 (first 234k, additional 468k each) | 1.6 M | 3.5 M | 3.5 M | 7.3 M |
| Object storage (media) incl. package fee | 12.6 M | 200.7 M | 28.8 M | 1,622.7 M |
| Video delivery (watch hours) incl. package fee | 47.3 M | 313.2 M | 66.8 M | 1,865.7 M |
| Video storage (stored hours) | 13.5 M | 111.6 M | 13.5 M | 777.6 M |
| CDN (images, API, static) | 10.3 M | 103.5 M | 36.0 M | 913.5 M |
| Off-provider backup copy (2nd provider, assumed same unit price) | 7.2 M | 63.9 M | 21.0 M | 425.7 M |
| **Total (list price)** | **≈ 259 M** | **≈ 1,570 M** | **≈ 944 M** | **≈ 8,753 M** |
| Budget with 30–40 % contingency | ≈ 340–360 M | ≈ 2.0–2.2 B | ≈ 1.2–1.3 B | ≈ 11.4–12.3 B |

**Variants and comparisons (same model):**

* **Scenario A with managed PostgreSQL** (recommended operating model in doc 18 §7, 3 × 8 vCore / 32 GB, 500 GB SSD each): Starter tier ≈ 262 M, Basic ≈ 293 M, Standard ≈ 317 M per month (DB line 80 / 111 / 135 M vs 77 M self-managed). Managed-DB backup is free **only up to 7 days** — the 35-day PITR in doc 06 §11 needs an additional backup copy.
* **Video: managed platform vs self-delivery via CDN** (≈ 450 MB per watch-hour at 1 Mbps): A 45 M vs 51 M; B 292 M vs 506 M; C 1,844 M vs 5,062 M per month (before transcoding compute for self-delivery). This is the evidence for ADR-009’s build-or-buy amendment.

**Excluded or simplified (inside the contingency):** CDN request fees above 1 billion requests/month (relevant at C), object-storage request fees and origin egress, L4-proxy/realtime traffic, CDN Enterprise negotiation, managed-DB extra backup retention, the unconfirmed billing period of video storage “per hour”, SMS (≈ 60–120k OTP/month at A, 0.5–1 M at B, 5–10 M at C), KYC checks, PSP fees, push/maps APIs, pentests, support tools. CDN fair use is quoted per day (50 / 500 GB/day) and converted to 30 days.

Key assumptions: 15 min video per DAU per day; 20 MB/DAU/day non-video CDN traffic; object storage = year-1 average; Standard compute tier; self-managed PostgreSQL unless stated. Media (video delivery, video storage, object storage, CDN) is ≈ 46 % of the B total and compute (vCore + RAM) ≈ 28 %; video + CDN alone ≈ 27 % (doc 10 §5.1 updated accordingly).

### 3.1 One-time setup costs (not recurring)

| Item | A | B | C | Price basis |
|---|---|---|---|---|
| Legal & regulatory set-up (entity, e-commerce trust seal, PSP onboarding, data-protection review, terms) | required | required | required | counsel quote |
| PSP integration certification / sandbox | required | required | required (2 PSPs) | PSP contract |
| External penetration tests before P6 and P7 | 2 tests | 2 tests | 2 tests + red team | vendor quotes |
| Design-system rebuild & UI security corrections (doc 19) | ~2–3 designer-months | ~3–4 | ~4–6 | people (C) |
| Phase 1 foundations engineering (subset of recurring people cost, not additional) | ≈ 20–24 person-months (8 × 2.5–3 mo) | ≈ 26–34 (17 × 1.5–2) | ≈ 53 (35 × 1.5) | people (C) |
| Domestic mirrors / CI runners / registry set-up | small | small | moderate | infra list price |
| App-store accounts, code-signing, devices lab | small | small | moderate | vendor prices |
| Recruitment | — | moderate | high | recruiter fees |

## 4. Team composition

### 4.1 Engineering roles and headcount

| Role | A | B | C |
|---|---:|---:|---:|
| Engineering lead / architect | 1 (hands-on) | 1 | 1 + 1 EM |
| Backend (Go) | 2 | 4 | 9 (social, commerce, money, media/realtime squads) |
| — of which payments/ledger specialist | (shared) | 1 | 2 |
| Mobile (Flutter) | 2 | 3 | 6 |
| Web (React: Seller Admin + Super Admin) | 1 | 2 | 4 |
| Platform / SRE / DevOps | 1 | 2 | 5 |
| QA / test automation | 1 | 2 | 3 |
| Security engineer | part-time advisor | 1 | 2 |
| Data / analytics engineer | — | 1 (from P7) | 2 |
| Product designer | (contract, part-time) | 1 | 2 |
| **Engineering total** | **8** (+ part-time design/security) | **17** | **35** |

### 4.2 Non-engineering operational roles

| Role | A | B | C |
|---|---|---|---|
| Product owner / product manager | owner | 1 PM | 2–3 PMs |
| Customer support | 1 (+ owner backup) | 2–3 | 6–10 (shifts) |
| Content moderation / trust & safety | 1 (part-time, grows at launch) | 2 | 6–8 incl. lead |
| Seller onboarding & KYC review | combined with support | 1 | 2–3 |
| Finance operations (reconciliation, payouts) — from P6 | 0.5 (accountant) | 1 | 2 |
| Legal / compliance | external counsel | external counsel + compliance officer (part-time) | in-house compliance officer |
| Security / incident coordinator | engineering lead | security engineer | security lead |

## 5. Development operating expense model

Monthly people cost = Σ (headcount_role × C_role). With a single blended C for engineering:

| | A | B | C |
|---|---|---|---|
| Engineering person-months to MVP (P1–P7) | ≈ 8 × 17 ≈ **136** | ≈ 17 × 11.5 ≈ **196** | ≈ 35 × 9 ≈ **315** |
| Monthly engineering cost | 8 × C_eng | 17 × C_eng | 35 × C_eng |
| Monthly operations cost | ≈ 2.5 × C_ops | ≈ 8 × C_ops | ≈ 24 × C_ops |

**Sensitivity example only — C values are hypothetical placeholders, not market data.** If C_eng = 100 M and C_ops = 40 M Toman/month: A ≈ 0.90 B, B ≈ 2.02 B, C ≈ 4.46 B Toman/month in people cost. Under these placeholders, **people cost exceeds infrastructure in A and B and is about half of infrastructure in C at full Tier C load**, so team size and hiring pace are the largest controllable budget levers until traffic is high.

## 6. Delivery time ranges (after Phase 1 authorisation)

| Milestone | A | B | C |
|---|---|---|---|
| P1 Foundations | 2.5–3 mo | 1.5–2 mo | 1.5 mo |
| Social MVP usable internally (P2–P3) | +4–5 mo | +3–3.5 mo | +2.5 mo |
| Catalog + Seller Admin + messaging (P4–P5) | +4–5 mo | +3–4 mo | +2.5–3 mo (parallel) |
| Commerce **without** real money (cart, orders in sandbox) | included in P6 build | included | included |
| P6–P7 built on PSP sandbox, then FG-1b real-money activation | +4–6 mo | +3–4 mo | +2.5–3 mo |
| **MVP total (sum of rows; external waits excluded)** | **14.5–19 mo** | **10.5–13.5 mo** | **9–10 mo** |
| First long-term track (live, advanced messaging) | after MVP | after MVP | in parallel from month ~6 |

## 7. Capacity, bottlenecks, risks, upgrade triggers, migration

| | A — Lean MVP | B — Balanced | C — Growth-Ready |
|---|---|---|---|
| **Expected capacity** | ≈10k DAU, 1k concurrent sessions, 300 RPS peak, ~300 orders/day, ~75k video watch-hours/month | ≈100k DAU, 10k concurrent, 3k RPS, ~3k orders/day, ~750k watch-hours/month | ≈1M DAU, 100k concurrent, 30k RPS, ~30k orders/day, ~7.5M watch-hours/month |
| **Major bottlenecks** | People (one DevOps, one payments-capable backend); PostgreSQL single cluster; moderation capacity at launch | Hiring and onboarding; feed fan-out for large accounts; video cost | Coordination across squads; data-store splits; vendor capacity (CDN/video at PB scale) |
| **Risks & trade-offs** | Slow delivery; key-person risk; limited 24/7 on-call; heavier reliance on provider-managed services (lock-in, SLA unknown) | Higher fixed burn before traction; needs disciplined scope (D-02) | High burn rate; risk of building scale before product-market fit; more partial-failure modes |
| **Upgrade triggers** | Sustained > 5k DAU growth/month; p95 latency SLO breaches; on-call fatigue (> 2 pages/week); real-money launch approved | DAU > 150k; DB CPU > 60 % sustained; outbox lag SLO; second PSP needed; live shopping approved | Measured triggers in doc 04 §9 (messaging cluster, feed service, money-module extraction) |
| **Recommended migration path** | A → B: add SRE #2, payments specialist, security engineer; move from managed DB to self-managed HA if vendor HA is insufficient; introduce Kubernetes; establish pilot-light DR | B → C: split DB clusters (money / social / messaging), OpenSearch scale-out, warm standby at second provider, squad structure | Extract services per doc 04 §9 only when triggers fire |

## 8. Recommendation

Start in **Scenario A for P1–P3 (foundations and social core)** — the work is mostly non-regulated and benefits from a small, senior team — and **move to Scenario B before P6** (real money), because B is the smallest configuration that provides segregation of duties for finance, a dedicated payments/ledger engineer, a security engineer and two-person on-call for money incidents. Scenario C is justified only by measured traction (§7 triggers). The owner selects the scenario; this recommendation is not a decision.
