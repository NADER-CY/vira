# 13 — Decision Register, External Dependencies and Risk Register

**Status:** Phase 0 **v4 (final for owner acceptance)** · 2026-10-29. v4: Claude-led execution (doc 20) resolves B-3 staffing; P1 blockers updated; risks R-48…R-52 added. Section 0 is the **authoritative final register**. Sections 1–4 are retained unchanged as history (v1/v2 recommendations and evidence) except where marked v3.

## 0. Final Product Owner Decision Register (v3)

### 0.1 Classification

| Class | Meaning |
|---|---|
| **CONFIRMED** | Decided by the product owner in writing (v3 brief) |
| **RETAINED** | Capability kept on the long-term roadmap by owner decision B-2 — **roadmap only; building or enabling it is NOT approved** until its own gate passes |
| **PROVISIONAL** | Architecture decision proposed by the architect, usable for planning; owner may change it at the named gate |
| **ASSUMPTION** | Planning assumption, to be validated |
| **EXTERNAL** | Depends on a provider, vendor or third party |
| **LEGAL/FIN** | Requires legal and/or financial confirmation before implementation |
| **OPEN** | Owner decision still required |

### 0.2 Owner decisions B-1 … B-7

| ID | Topic | Status | What is decided | What remains |
|---|---|---|---|---|
| B-1 | Launch market & residency | **CONFIRMED** | Iran; Persian; RTL; Toman display; integer Rial ledger; domestic hosting for operations and regulated data; no mandatory foreign runtime dependencies; international expansion is future | Iranian regulatory/payment requirements are **not** legally confirmed (LEGAL/FIN) |
| B-2 | Scope strategy | **CONFIRMED** | Full-platform architecture with phased implementation; MVP priorities 1–13 (identity, social, posts/stories/reels, media, essential messaging, marketplace, seller onboarding & storefronts, catalog & inventory, cart & orders, payments **subject to separate financial approval**, Seller Admin, essential Super Admin, notifications/security/moderation/audit) | Long-term capabilities retained in the roadmap as gated tracks (doc 12 §v3) |
| B-3 | Budget, traffic, team | **CONFIRMED (v4) for development** | Claude-led development, no team hired; specialists only at gates L and FG-1b (doc 20). Doc 17 kept as production-infrastructure reference | Production sizing at gate L from measured load |
| B-4 | Mobile distribution | **PROVISIONAL (owner)** | Flutter; Android first; iOS compatibility from the start (ADR-003 amended) | Evaluation delivered (doc 11 §4.1: Bazaar + Myket + direct APK; iOS via PWA interim, native after counsel); store and legal confirmations EXTERNAL/LEGAL |
| B-5 | Hosting | **OPEN** (evaluation delivered) | Strategy recommended: primary full-service domestic cloud (ArvanCloud conditional), recovery at a second domestic provider (doc 18) | Vendor due diligence (doc 18 §6) and contracts (EXTERNAL) |
| B-6 | ADR baseline | **CONDITIONALLY ACCEPTED** | ADR-001…022 baseline; final matrix in doc 05 (15 READY, 4 AMENDED → READY, 3 with BLOCKED components) | Owner’s final acceptance of the matrix |
| B-7 | UI security corrections | **CONFIRMED in principle** | Seven unsafe patterns prohibited; redesign specification doc 19 | Design work in the owning phases |
| — | Shared seller identity | **CONFIRMED** | One account; seller permissions on approval; “Manage My Stall” in app; Seller Admin web on the same backend; seller notifications on both | Design detail in doc 04 §10 (PROVISIONAL architecture) |
| — | Financial model | **OPEN — LEGAL/FIN** | Platform-collects is **not** approved; formal decision required between platform collection and PSP split (doc 06 §6.5) | Counsel + PSP confirmation before any real-money build |

### 0.3 Status of decisions D-01 … D-35

| ID | Topic | v3 status | Needed by |
|---|---|---|---|
| D-01 | Jurisdiction & residency | **CONFIRMED** (B-1); legal requirements LEGAL/FIN | P2 (legal review of terms) |
| D-02 | MVP boundary | **CONFIRMED** (B-2) | — |
| D-03 | Wallet | **RETAINED** as gated track G1; activation LEGAL/FIN | G1 gate |
| D-04 | USDT payments, USDT↔Toman conversion both ways, eligible seller USDT withdrawals | **RETAINED** as gated track G2; default OFF; activation LEGAL/FIN | G2 gate |
| D-05 | PSP & settlement model | **OPEN — LEGAL/FIN + EXTERNAL** | **FG-1a** (before P6 design freeze) |
| D-06 | Hold period & “escrow” wording | OPEN — LEGAL | FG-1a (wording/model), FG-1b (final values) |
| D-07 | Commission & tax | OPEN — owner + LEGAL/FIN | FG-1a (draft), FG-1b (final) |
| D-08 | Returns, withdrawal window, refund rail | OPEN — LEGAL | P7 design / FG-1b |
| D-09 | Seller eligibility | OPEN | P4 |
| D-10 | KYC levels, providers, limits, biometric retention | OPEN — owner + LEGAL + EXTERNAL | P2 (L1), P4 (L2) |
| D-11 | Messaging privacy model | OPEN — LEGAL | P5 |
| D-12 | Retention & erasure | OPEN — LEGAL | P2 exit |
| D-13 | Content policy, minimum age | OPEN | P2 |
| D-14 | Traffic, budget, team | CONFIRMED for development (= B-3, doc 20); production traffic targets OPEN | Gate L (production sizing) |
| D-15 | Seller agreement acceptance | OPEN — LEGAL | P4 |
| D-16 | Mobile distribution | PROVISIONAL (= B-4); EXTERNAL | Before first public release |
| D-17 | Hosting provider | OPEN (= B-5); EXTERNAL | **Phase 1 start (dev/CI/staging account)**; production contract before public beta |
| D-18 | Push providers | OPEN — EXTERNAL | P2 |
| D-19 | Live shopping & calls | **RETAINED** (P8 gate) | P8 |
| D-20 | Ads & affiliate | **RETAINED** (P9 gate); creator payout tax LEGAL | P9 |
| D-21 | AI providers & data sharing | **RETAINED** (P9 gate); data sharing LEGAL | P9 |
| D-22 | Login factors (OTP 6 digits, optional TOTP, mandatory TOTP for seller owner/finance; financial PIN/biometric optional) | PROVISIONAL | P2 |
| D-23 | Seller Admin web | **CONFIRMED** (owner seller requirement) | P4 |
| D-24 | Shipping carriers | OPEN — EXTERNAL | P7 |
| D-25 | Location features | PROVISIONAL (coarse by default) | P4 |
| D-26 | Brand seed & typography | PROVISIONAL (`#ba0038`, Vazirmatn) | P1 design tokens (can change later without rework) |
| D-27 | Credit / instalments / BNPL | **RETAINED** as gated track G3; LEGAL/FIN | G3 gate |
| D-28 | Primary navigation | PROVISIONAL (most frequent designed pattern); validate with users | P2 |
| D-29 | Advanced shipping & logistics | **RETAINED** (P11). How it is delivered — own courier fleet vs 3PL partnership vs carrier integrations — is **OPEN** (P11 gate). MVP: seller-shipped + carriers (A3, PROVISIONAL) | P7 / P11 |
| D-30 | Loyalty, rewards, gift cards; lucky wheel | **RETAINED** (P9); gift cards with G1; lucky wheel LEGAL review | P9 |
| D-31 | Live virtual gifts | **RETAINED** with live (P8); stored value → LEGAL/FIN (G1) | P8 / G1 |
| D-32 | Seller plans & VIRA Plus | **RETAINED** (P9) | P9 |
| D-33 | Live auctions | **RETAINED** as advanced capability (P8/P9 gate); LEGAL review | P8 / P9 |
| D-34 | Saved cards (PSP tokenisation) | OPEN — EXTERNAL (PSP capability) | FG-1b |
| D-35 | Stock holding during KYC | PROVISIONAL (cart kept, stock not held) | P6 |

### 0.4 Assumptions still in force

A1 Iran launch (now CONFIRMED by B-1) · A2 one PSP at launch (ASSUMPTION) · A3 sellers ship in MVP (PROVISIONAL; advanced logistics retained, delivery model OPEN) · A4 pure marketplace (ASSUMPTION) · A5 Persian only at launch (CONFIRMED by B-1) · A6 *v4:* Claude-led development, specialists at gates (CONFIRMED, doc 20) · A7 card data never in VIRA (CONFIRMED by B-7) · A8 archive is the latest design (ASSUMPTION) · A9 *new:* provider prices as of 2026-10-10 list prices (ASSUMPTION, re-price quarterly) · A10 people-cost parameters (doc 17) — no longer needed for development (v4).

### 0.5 Blockers that genuinely prevent starting Phase 1 foundations

| # | Blocker | Type |
|---|---|---|
| P1-B1 | Owner’s explicit written **authorisation of Phase 1** | Owner |
| P1-B2 | Owner’s final acceptance of the ADR matrix (doc 05) | Owner |
| P1-B3 | **v4:** private repository with access for Claude’s sessions, and an implementation environment with package-registry access (doc 20 §5.1 O-3, O-4) | Owner |
| P1-B4 | **v4: no longer blocks the start** — a domestic provider account and one staging VM are needed by the **end** of Phase 1 (doc 20 §3.2) | Owner + EXTERNAL |

**Not Phase 1 blockers:** financial model (FG-1a, before P6) and real-money activation (FG-1b), PSP, wallet/USDT/credit tracks, KYC providers, messaging privacy, retention policy, carriers, iOS feasibility, live/ads/AI gates — each is attached to the later phase that needs it.

---

## 1. Open decisions (product owner) — v2 history

| ID | Decision | Options | Architect’s recommendation | Blocks |
|---|---|---|---|---|
| **D-01** | Launch jurisdiction, legal entity, data residency | Iran-only · Iran + other · other | Iran-only (matches all UI evidence); in-country residency | **P1** |
| **D-02** | MVP boundary | Social only · Social + catalog · Full commerce loop | Full loop P1–P7, no live/calls/ads/affiliate/AI/wallet/crypto (doc 03 §7) | **P1** |
| D-03 | Wallet model | None · Seller/creator earnings balance only · Buyer refund credit · Stored-value top-up | Earnings balance only; no top-up; refunds go to the original payment source by default (see D-08) | P6 |
| D-04 | Crypto/USDT | Out of scope · Future gated track | Out of scope; G2 only after separate review | — (default OFF) |
| **D-05** | PSP & settlement model | Platform collects (licensing?) · PSP split settlement (تسهیم) | Depends on licensing; platform-collects gives return/fraud protection via holds; split simplifies regulation but weakens holds (doc 06 §6.3) | **P6** (design), P1 (vendor shortlist) |
| D-06 | Settlement-hold rules & terminology | Hold until delivery + N days · until return window closes | Hold until return window closes; no “escrow” wording without legal approval | P6 |
| D-07 | Commission & tax | Flat · per category · per seller tier; commission on shipping?; tax base; e-invoicing | Per-category bps with seller-tier overrides; snapshot per order | P6 |
| D-08 | Returns & refunds | Window length; who pays return shipping; refund rail (PSP reversal / Sheba transfer / credit) | Statutory minimum per counsel; **refund to source via PSP** by default; Sheba transfer only where the PSP cannot refund, and only to a Sheba whose owner matches the buyer’s national ID (L1+), with step-up and a finance checker (audit F-08) | P7 |
| D-09 | Seller eligibility & verification | Individuals allowed? · Legal entities only · Trust seal requirement | Individuals at lower limits; entities for higher tiers | P4 |
| D-10 | KYC levels, providers, limits, biometric retention | — | L0/L1/L2 as doc 03; retain decision + minimal data; raw biometrics ≤ the shortest legally allowed period | P2 (L1), P4 (L2) |
| D-11 | Messaging privacy model | Server-readable encrypted at rest · E2EE | Store chats server-readable; personal DMs E2EE later if permissible | P5 |
| D-12 | Retention & erasure | Per data class | Retention matrix to be produced in P1 with counsel | P2 |
| D-13 | Content policy, minimum age, child safety | — | Written policy v1 before P2 exit | P2 |
| **D-14** | Traffic targets, budget envelope, team size | Tier A/B/C | Plan for Tier A, design to Tier B; **re-baseline team**: add a second platform/SRE engineer, a payments-specialist backend engineer and a security lead, plus moderation, finance-ops and support staffing before P2/P7 exits (audit F-11) | **P1** |
| D-15 | Seller agreement acceptance mechanism | Click-through · OTP-signed · qualified digital signature | OTP-signed click-through with versioned terms unless counsel requires more | P4 |
| D-16 | Mobile distribution | Android domestic stores / direct APK / Google Play; iOS feasibility | Android first; iOS decision after legal review | P1 |
| **D-17** | Hosting provider & DR site | Domestic cloud providers / colocation | Domestic provider with two failure domains; quote-based selection | **P1** |
| D-18 | Push providers | FCM/APNs + domestic | Abstraction with ≥ 1 domestic provider | P2 |
| D-19 | Live shopping & calls timing | MVP · post-MVP | Post-MVP (P8) | P8 |
| D-20 | Ads & affiliate monetisation | — | Post-MVP (P9); requires ad-spend billing and creator payout tax treatment | P9 |
| D-21 | AI providers & data sharing | Foreign API · domestic · self-hosted | Self-hosted or domestic for anything touching personal data; no user data to foreign AI APIs without consent and legal basis | P3 (moderation), P9 |
| D-22 | Login factors | OTP-only · OTP + password · OTP + TOTP | OTP + optional TOTP; mandatory TOTP for seller owner/finance | P2 |
| D-23 | Seller web console in MVP | Mobile only · Web MVP | Web MVP (catalog bulk + orders + finance) | P4 |
| D-24 | Shipping carriers | Integrate · manual tracking codes | Manual tracking codes in MVP + one carrier integration | P7 |
| D-25 | Location features | Off · coarse · precise opt-in | Coarse by default, precise opt-in | P9 |
| D-26 | Brand seed colour & typography | `#ba0038` · `#8e0028` | `#ba0038`, Vazirmatn (v2: five primaries found in use) | P1 |
| D-27 | Credit / instalments / BNPL (UI: “VIRA credit wallet”, “4 instalments, no guarantor”) | Out of scope · bank-partner BNPL · own lending | Out of MVP; gated track G3 | — (default OFF) |
| D-28 | Primary mobile navigation | Dominant design (Home·Explore·＋·Reels·Profile, Shop in top bar) · Shop as tab | Most frequent design, provisionally (doc 09 §4); no majority in the designs — validate with users | P1 |
| D-29 | Platform courier fleet (“سفیر ویرا”, live rider tracking) vs seller-shipped | Seller-shipped + carriers · own fleet · 3PL partnership | Seller-shipped + carriers for MVP; courier product later | P7 |
| D-30 | Loyalty: coins, cashback, gift cards, lucky wheel | — | Coins/cashback as non-cash points later; gift cards gated with D-03; **lucky wheel rejected** pending legal review | P9 |
| D-31 | Live virtual gifts (“diamonds”) | Out · gated | Gated (virtual currency = stored value) | — |
| D-32 | Seller subscription plans; “VIRA Plus” buyer membership | — | Later (P9) | P9 |
| D-33 | Live auctions | Out · later | Out of scope | — |
| D-34 | Saved cards | None · PSP-tokenised card-on-file | PSP tokenisation only, if offered | P6 |
| D-35 | Holding stock while KYC is reviewed (UI: cart reserved 24 h) | Hold stock · hold cart only | Hold the cart, not inventory; stock reserved only until the payment deadline | P6 |

**v2 evidence on existing decisions (doc 15):** D-03 — UI assumes stored value (top-up, transfer, refunds to wallet, gift cards). D-04 — UI promotes USDT as the default **and includes an in-app Toman→USDT exchange** (`_163`); D-04 must cover payment, payout **and** exchange. D-10 — UI also defines a third tier (L3 corporate/VIP, unlimited, OTC). D-06/D-08 — four different hold/return values in the UI (immediate, 48 h, 7 days; statutory 48 h vs 7 days). D-10 — L2 limits 50M vs 250M; withdrawal 50M/100M/200M. D-22 — UI adds password, biometric login and a separate “financial PIN”. D-23 — `_228` references a “Seller Web Admin”. D-24 — UI includes “VIRA express” as a carrier.

## 2. Risk register

Scoring: Likelihood (L) and Impact (I) 1–5; Score = L × I. ≥ 15 = HIGH, ≥ 20 = CRITICAL.

| ID | Risk | L | I | Score | Owner | Mitigation | Trigger / indicator |
|---|---|---:|---:|---:|---|---|---|
| R-01 | Phase 0 inventory built without viewing screenshots/HTML; misclassification | ~~4~~ 1 | 4 | ~~16~~ 4 | Architect | **Mitigated in v2:** archive integrity verified, all 42 open records resolved, priority screens inspected (doc 15). Residual: 260 screens verified by HTML structure only | New design exports |
| R-02 | Payment licensing model unclear (collect vs split) | 4 | 5 | **20** | Owner + counsel | Decide D-05 before P6 design freeze. Only the platform-collects model is fully specified; the split model needs its own rule set (doc 06 §6.3, audit F-01) | No decision by P4 exit |
| R-03 | Crypto features enabled or leaked into core flows | 2 | 5 | 10 | Security | Multi-layer gate (doc 08 §7) | GATED flag change requests |
| R-04 | Sanctions / app-store distribution prevent iOS launch | 4 | 4 | 16 | Owner | Android-first; D-16 legal review | — |
| R-05 | International connectivity disruption breaks app or builds | 3 | 5 | 15 | Platform | No foreign runtime deps; in-country mirrors; domestic push/SMS/maps | Disruption events |
| R-06 | Double charge or lost payment from callback/verify edge cases | 3 | 5 | 15 | Payments lead | Idempotency, inquiry jobs, simulator matrix, recon | Unknown-state intents > 0 for 30 min |
| R-07 | Ledger imbalance / incorrect seller payout | 2 | 5 | 10 | Finance + payments | Invariants, nightly integrity, maker–checker, payout freeze on break | Integrity job failure |
| R-08 | Return/refund fraud | 4 | 3 | 12 | Risk | Evidence, thresholds, scoring, holds | Refund rate per seller/buyer |
| R-09 | SIM-swap account takeover → payout theft | 3 | 4 | 12 | Security | Cooling-off, TOTP, notifications | Sheba change + new device |
| R-10 | SMS pumping / OTP cost explosion | 4 | 3 | 12 | Platform | Layered rate limits, prefix anomaly detection, cost alerts | SMS spend / hour |
| R-11 | CDN egress cost exceeds budget | 4 | 4 | 16 | Platform + product | Bitrate policy, per-title encoding, measurement in P3 exit | Cost per DAU |
| R-12 | Illegal / harmful content incl. CSAM | 3 | 5 | 15 | Trust & safety | Hash matching, escalation queue, staffing, policy D-13 | Report volume |
| R-13 | Tenant data leakage between sellers | 2 | 5 | 10 | Backend lead | Membership check + RLS + automated cross-tenant tests | Any finding |
| R-14 | Admin account compromise | 2 | 5 | 10 | Security | VPN, WebAuthn, least privilege, audit alerts | Unusual admin activity |
| R-15 | Scope creep: 53 admin consoles + 277 screens treated as commitments | 5 | 3 | 15 | Product | Classification in doc 03; MVP boundary D-02 | Backlog growth vs plan |
| R-16 | Seller productivity limited by mobile-only tools | 4 | 3 | 12 | Product | Seller web MVP (D-23) | Seller churn / support tickets |
| R-17 | Persian search quality poor (normalisation, morphology) | 3 | 3 | 9 | Search | Normalisation from day one; OpenSearch Persian analyser; curated query tests | Zero-result rate |
| R-18 | Small team cannot operate infra (K8s, NATS, OpenSearch, observability) | 3 | 4 | 12 | Platform | Defer OpenSearch; conditional K8s; managed services where mature | On-call load |
| R-19 | Database becomes bottleneck (reactions, messages) | 2 | 4 | 8 | Backend | Async counters, partitioning, extraction triggers | Lock waits, WAL rate |
| R-20 | KYC provider availability / accuracy | 3 | 3 | 9 | Compliance | Queue + manual review; second provider later | Provider error rate |
| R-21 | Biometric data breach | 2 | 5 | 10 | Security | Separate encrypted bucket + key, minimal retention, access audit | — |
| R-22 | Design inconsistency (Latin fonts, theme gaps, duplicates) ships to users | 4 | 2 | 8 | Design | Token pipeline, golden tests, single seed | Visual review |
| R-23 | Legal exposure from “escrow/guarantee” wording | 3 | 4 | 12 | Owner + counsel | Copy review (D-06) | — |
| R-24 | Key personnel dependency (one person knows payments) | 3 | 4 | 12 | Eng manager | Pairing, runbooks, two reviewers on money code | — |
| R-25 | Live shopping costs/moderation overwhelm | 3 | 3 | 9 | Product | Post-MVP, eligibility, caps | — |
| R-26 | Third-party trademark in design artefacts (`tiktok_shop_hub`) | 2 | 2 | 4 | Design | Rename | — |
| R-27 | Messaging lawful-access/retention obligations conflict with privacy promises | 3 | 4 | 12 | Owner + counsel | D-11 before P5 | — |
| R-28 | Late PSP success after stock released | 3 | 3 | 9 | Payments | Re-reserve or auto-refund path tested | Count per day |
| R-29 | Designs that capture or search card data (`_41`, `_142`, `_66`, admin card-number search) are implemented — card-data exposure, phishing pattern, likely PSP/Shaparak non-compliance | 3 | 5 | 15 | Design + security | Designs rejected (doc 15 V-01); security review gate on every payment UI | Any card field in app code |
| R-30 | UI-promoted crypto leaks into MVP builds or copy | 3 | 5 | 15 | Product + security | Multi-layer gate (doc 08 §7); remove from designs | Any USDT string in MVP |
| R-31 | UI-implied regulated products (credit, wallet top-up, gift cards, virtual gifts) are built without licences | 3 | 5 | 15 | Owner + counsel | D-27, D-03, D-30, D-31 default OFF | Backlog items |
| R-32 | Contradictory policy values shown to users (limits, return windows, hold times) become binding promises | 4 | 4 | 16 | Product + counsel | All values from configuration; copy references config | Copy review |
| R-33 | Admin one-click money actions and unmasked PII are implemented as drawn | 3 | 5 | 15 | Security | Admin finance redesign before P6/P7 (doc 15 V-06) | Admin UI review |
| R-34 | Navigation inconsistency (14 tab bars, 46 sidebars) carried into build | 4 | 2 | 8 | Design | D-28; single admin sidebar | — |
| R-35 | Gambling-like mechanics (lucky wheel) create legal/reputational exposure | 2 | 4 | 8 | Owner + counsel | Rejected pending review (D-30) | — |
| R-36 | Platform courier expectations (live rider tracking) set by design but not planned | 3 | 3 | 9 | Product | D-29 | — |
| R-37 | Government domain (`vira.gov.ir`) used in admin designs → impersonation concerns | 2 | 3 | 6 | Design | Replace with the real corporate domain | — |
| R-38 | Pre-ticked consent and attestation boxes are implemented as drawn — invalid consent | 3 | 4 | 12 | Product + counsel | Untick by default; versioned consent log (doc 08 S-UI-08) | UI review |
| R-39 | In-app fiat→USDT exchange (`_163`) treated as part of “payments” rather than a separate regulated activity | 2 | 5 | 10 | Owner + counsel | Named explicitly in D-04; out of scope | — |
| R-40 | Provider prices revised quarterly with high inflation → budget drift | 5 | 3 | 15 | Owner + finance | Quarterly re-pricing; 30–40 % contingency; committed-use contracts (doc 17 §1) | Price-version change |
| R-41 | Provider concentration in Tehran; city- or provider-level outage | 2 | 5 | 10 | Platform | Second domestic provider (pilot light → warm), facility outside Tehran preferred (doc 18 §5.2) | DR drill results |
| R-42 | Vendor SLA, certifications and support quality unverified | 4 | 4 | 16 | Platform | Vendor questionnaire (doc 18 §6) before contract; exit plan | Missing answers |
| R-43 | People-cost and hiring uncertainty (salary data unverified) | 4 | 4 | 16 | Owner | HR data for C_eng/C_ops; phased hiring (doc 17 §8) | Open roles > 60 days |
| R-44 | Video-platform lock-in | 3 | 3 | 9 | Platform | `VideoProvider` adapter; originals kept in VIRA storage (ADR-009 amended) | Price/terms change |
| R-45 | Shared identity: staff gain access to owner’s personal data, or removed staff keep store access | 2 | 4 | 8 | Security | Store-scoped permissions, immediate session scope revocation, tests (doc 04 §10, doc 08 §4.2) | Any finding |
| R-46 | iOS distribution not feasible at launch | 4 | 3 | 12 | Owner | Android-first; platform-neutral code (ADR-003 amended) | D-16 outcome |
| R-47 | Legal/sanctions status of a chosen provider changes | 2 | 5 | 10 | Owner + counsel | Due diligence; recovery provider; IaC portability | News / notices |
| R-48 | Single AI implementer: defects missed by self-review | 3 | 4 | 12 | Claude + owner | Test-first definition of done; ChatGPT review on request; independent human security audit at gate L and before FG-1b (doc 20 §1.4) | Escaped defects in completion reports |
| R-49 | Session continuity: context lost between sessions | 3 | 3 | 9 | Claude | Repository as single source of truth; `docs/progress/Pn.md` per phase | Rework caused by lost context |
| R-50 | No 24/7 operator once users exist | 4 | 4 | 16 | Owner | Alerting to owner; named on-call human or contracted operator before launch (gate L) | Gate L |
| R-51 | Implementation environment lacks registry access or repository access is restricted | 3 | 4 | 12 | Owner | Enable registry access (O-4); CI builds; self-hosted Gitea fallback | Build failures |
| R-52 | Owner review and account-opening turnaround becomes the bottleneck | 3 | 3 | 9 | Owner | Approval required only for sensitive changes (doc 20 §1.2); provider accounts opened one phase ahead | Phase blocked on owner input |

## 3. Explicit assumptions (to be confirmed or corrected)

1. A1 — Launch market is Iran; currency IRR/Toman only.
2. A2 — One PSP at launch; a second PSP is a post-MVP resilience item.
3. A3 — Sellers handle their own shipping in MVP (no platform warehouse). **v2: contradicted by UI** (platform courier `_106`, `_44`; “VIRA express” carrier `_15`) — kept for MVP pending D-29.
4. A4 — No platform-owned inventory (pure marketplace).
5. A5 — Persian is the only UI language at launch; English used only for technical strings.
6. A6 — Team size and composition as in doc 12.
7. A7 — Card data never touches VIRA systems (PSP-hosted payment page).
8. A8 — The two manifest copies supplied represent the latest UI pack; no newer design export exists.

## 4. External dependency register (added after audit F-25)

Each must have a named provider candidate, contract owner and fallback before the phase it blocks. Behaviour (cycles, thresholds, reversal windows, response codes) is **to be confirmed from the contract and sandbox**, not assumed.

| Dependency | Needed for | Blocks | Fallback | Residency |
|---|---|---|---|---|
| SMS gateway ×2 | OTP, transactional SMS | P2 | Second provider; voice OTP | In-country |
| Shahkar-class phone↔national-ID check | KYC L1, recycled-number protection, recovery | P2 | Manual review | In-country |
| Civil-registry identity check | KYC L1/L2 | P2/P4 | Manual review | In-country |
| Liveness / document-verification vendor | KYC L2 | P4 | Manual video review | In-country preferred |
| Sheba (IBAN) owner inquiry | Seller payout accounts, Sheba refunds | P4 | Manual document check + small test deposit | In-country |
| Bank-card owner inquiry | `bank_card_ownership_verification` flow | P4 | Skip (lower limits) | In-country |
| PSP (IPG) incl. inquiry, settle, refund APIs | Checkout | P6 | Second PSP | In-country |
| Paya/Satna submission or PSP payout API, return files | Payouts | P7 | Manual bank portal with maker–checker | In-country |
| Domestic push provider | Notifications without Play services / during disruptions | P2 | In-app WebSocket | In-country |
| Domestic maps / geocoding | Address pin, nearby stores | P4 | Text address only | In-country |
| CDN + object storage | Media | P3 | Second provider | In-country |
| Antivirus engine (self-hosted) | Document uploads | P2 | Reject PDFs | Self-hosted |
| NSFW / CSAM hash-matching | Moderation | P3 | Human review only (capacity-limited) | Self-hosted or in-country |
| Shipping carrier API(s) | Tracking | P7 | Manual tracking codes | In-country |
| Tax e-invoicing integration | Seller/platform invoices (D-07) | P7 (if required) | Export files | In-country |
| Self-hosted challenge (PoW/CAPTCHA) | OTP abuse | P2 | Stricter limits | Self-hosted |
| Primary cloud provider (doc 18) | All runtimes | **P1** (dev/CI/staging), public beta (production) | Recovery provider | In-country |
| Recovery cloud provider (doc 18) | DR copy, pilot light | Before public beta | Second region of primary (weaker) | In-country |
| Managed video platform (doc 18) | VOD transcoding & delivery, later live | P3 | Self-hosted FFmpeg/HLS (ADR-009) | In-country |
| Counsel (legal opinions L-01…L-13) | Financial gate, policies | P2 (terms, privacy), FG-1a, FG-1b | — | — |
| HR / recruiter salary data | Budget scenarios (doc 17) | P1 staffing | — | — |
