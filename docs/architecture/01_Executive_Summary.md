# 01 — Executive Summary

**Status:** Phase 0 **v4 — final, ready for owner acceptance** · **Phase 1 not authorised** · 2026-10-29 (v2: UI verified against the original archive, doc 15; v4: Claude-led execution strategy, doc 20)

> **v4 (2026-10-29):** development is **Claude-led** with no development team hired (owner instruction). Doc 20 supersedes all staffing, team-size and duration assumptions below; architecture, scope, gates and security requirements are unchanged.


## What VIRA is

A Persian-first, RTL social-commerce platform for Android and iOS: an Instagram/TikTok-style social network (feed, stories, reels, explore, creators) fused with a multi-vendor marketplace (storefronts, variants and inventory, multi-seller cart, one payment split into per-seller orders, returns, settlements), plus messaging, calls and live shopping, operated through a Super Admin console and a seller workspace.

## What the evidence shows

* **330 unique UI screens** (277 mobile, 53 Super Admin; the second admin tree is a byte-identical copy). One mobile screen (`_42`) duplicates another exactly and is the one without a screenshot. Six design documents reduce to four, i.e. two crimson brand seeds with light/dark pairs.
* **v2 verification:** the original archive matches the manifest exactly (383/383 HTML). All 17 previously unverified and 25 contradictory records are resolved — the contradictions were stale top-bar labels, not conflicting content. Every screen was parsed; 63 were inspected visually; 5 screenshots are blank and 1 is truncated.
* The design pack assumes **Iran**: Toman, Shaparak, Paya/Satna, Sheba, the national smart card, Jalali dates.
* **Crypto (USDT/TRC20) is woven into otherwise-core admin consoles**, and the UI uses “escrow” and “smart contract” language that implies regulated or on-chain mechanisms that do not exist.
* **There is no Seller Admin web UI**; all seller tooling is on the phone.
* **v2:** the designs **capture card data inside the app** (`_41`, `_142`: card number, CVV2, expiry, dynamic PIN; one imitates a bank gateway and offers “save card”). These designs are rejected; the architecture (PSP-hosted payment page) stands.
* **v2:** the UI implies products the architecture deliberately excludes or has not scoped: USDT as the *default* payment and payout method plus an in-app Toman→USDT exchange, credit/instalments (BNPL), a stored-value wallet with gift cards and refunds paid into it, live virtual gifts, seller subscription plans, coins/cashback, a lucky wheel, a platform courier fleet and live auctions.
* **v2:** the money-out paths in admin (payout batches, early release of held funds, refunds) are single-actor actions, an “instant settlement” button sits in 17 console headers, and personal data is shown unmasked — although the same pack already draws a correct dual-control flow for wallet adjustments. Several consent and attestation boxes are **pre-ticked**, and one admin console searches by full card number. Policy values also contradict each other (hold release immediate / 48 h / 7 days; KYC Level 2 limit 50M vs 250M Toman).
* 162 of 277 mobile screens never reference a **Persian font**; every screen depends on Google Fonts, the Tailwind CDN and hot-linked Google-hosted images, none of which is acceptable for an in-country production app.
* Several features are **concepts, not requirements**: in-call purchasing, AI shopping assistant and visual search, AI studios, loyalty tiers, instant delivery, “treasury verification”, bespoke NOC/command-centre consoles.

## What we recommend

| Area | Recommendation |
|---|---|
| Architecture style | **Modular monolith in Go** with strict module boundaries, schema-per-module in PostgreSQL, a single transactional outbox, and five runtimes from one codebase (`api`, `admin-api`, `worker`, `realtime`, `media`). Microservices rejected for MVP on cost, correctness and team-size grounds; extraction triggers defined. |
| Proposed stack | Flutter ✅ · Go ✅ · PostgreSQL ✅ · Redis-compatible cache ✅ (Valkey option for licensing) · NATS ✅ for realtime, with **JetStream deferred** until a measured trigger (outbox + PostgreSQL job queue cover Tier A) · React + TypeScript ✅ · Object storage + in-country CDN ✅ · WebRTC for calls/hosts, **HLS for live viewers** (refinement). Additions: PostgreSQL-backed job queue, OpenSearch deferred to the catalog phase, self-hosted observability. |
| Money | **Double-entry, append-only ledger** in integer Rial is the financial source of truth; settlement hold until the return window closes (**Model 1 only — subject to the financial-model gate FG-1a**, doc 06 §6.5); maker–checker on payouts and large refunds; daily three-way reconciliation; payment “unknown” state handled by inquiry jobs so buyers are never asked to pay twice. |
| Security | Separate Super Admin realm behind VPN with WebAuthn/TOTP; RBAC for admins, store-scoped roles plus PostgreSQL row-level security for sellers; field-level encryption for national ID, biometrics and Sheba; append-only, hash-chained audit log; layered OTP abuse controls; crypto disabled at build, API, ledger and UI layers. |
| Hosting | **In-country**, two failure domains, no foreign runtime dependencies, in-country mirrors for build dependencies; primary + independent recovery provider (doc 18). |
| Design | One brand seed, generated light/dark tokens, **Vazirmatn** everywhere, self-hosted assets, five-tab navigation following the most frequent designed pattern (provisional, D-28) with a seller “Manage My Stall” mode. |
| MVP | Phases **P1–P7**: foundations → social core → media/reels/stories → stores & catalog (+ Seller web MVP) → messaging → checkout/payments/ledger (sandbox until FG-1b) → fulfilment/returns/settlement. Live, calls, ads, affiliate, AI, wallet, USDT and credit are retained on the roadmap behind their own gates. Duration depends on the budget scenario (doc 17 §6). |

## Development strategy (v4, doc 20)

Claude is architect and sole implementation owner; ChatGPT reviews on request (mainly React/admin); the owner decides, owns accounts and contracts, and authorises each phase; human specialists are brought in only at named gates (security audit before launch, financial verification before real money). Development runs on local Docker Compose, CI and one small domestic staging VM (≈ 5.5 M Toman/month list price, ≈ 11.2 M from P3). No calendar durations are given until Phase 1 velocity is measured.

## Production infrastructure reference (v3, doc 17 — team columns superseded by doc 20)

Bottom-up estimates from ArvanCloud Toman list prices in a dated third-party snapshot of its official price page (2026-10-10; prices are revised roughly quarterly — confirm with a vendor quote and budget a 30–40 % contingency):

| Scenario | Engineering team | Infra at design load (list price, per month) | MVP delivery |
|---|---|---|---|
| A — Lean MVP (≈10k DAU) | 8 | ≈ 0.26 billion Toman (≈ 0.26–0.32 B with managed DB) | 14.5–19 months |
| B — Balanced (≈100k DAU) | 17 | ≈ 1.57 billion Toman (≈ 0.94 B at launch traffic) | 10.5–13.5 months |
| C — Growth-ready (≈1M DAU) | 35 | ≈ 8.75 billion Toman | 9–10 months |

Video delivery and media storage dominate infrastructure from Scenario B upward. People costs could not be verified and are modelled as headcount × a cost parameter the owner supplies; under illustrative parameters they exceed infrastructure in A and B. Recommendation: **A for P1–P3, B before real money**.

## What blocks Phase 1 (v4)

Only three things: the owner’s written **authorisation**, final **acceptance of Phase 0 / the ADR matrix**, and a **repository plus an implementation environment with package-registry access** (doc 20 §5). The domestic staging account is needed by the end of Phase 1, not to start it. The financial model, PSP, wallet/USDT/credit tracks and later policy decisions are attached to later gates and do not block foundations (doc 16).

## Final verdict (v3)

| Question | Answer |
|---|---|
| Phase 0 ready for final owner acceptance? | **YES** |
| Could Phase 1 technically begin after authorisation? | **YES**, after the prerequisites above (doc 20 §5) |
| Phase 1 authorised? | **NO** |
| Real money, wallet, USDT, credit? | **NO-GO** until FG-1a/FG-1b, G1, G2, G3 |

Owner decisions B-1…B-7 are recorded in the final register (doc 13 §0); hosting options in doc 18; UI corrections in doc 19.

## History — v1/v2 audit and verdict


A separate reviewer audited the package cold and raised **30 findings (2 CRITICAL, 9 HIGH, 14 MEDIUM, 5 LOW)**. Twenty-six are corrected in this revision; the rest are open with named gates (doc 14). The most important corrections: the checkout timer no longer races the payment, PSP callbacks are treated as untrusted browser hints, the ledger has a complete rule set (including refunds after payout and returned transfers), the database role and row-level-security model was made workable, and JetStream was deferred to reduce operational load.

| Question | Recommendation (v2, after UI verification) |
|---|---|
| Ready for product-owner review? | **GO** |
| Phase 0 complete? | **NO-GO** — seven owner-side blockers remained at v2 (doc 16, v2 appendix); the UI evidence gap is closed |
| Start Phase 1? | **Conditional GO** once D-01, D-02 (including rulings on the UI-implied products), D-14 (larger team), D-16 and D-17 are decided, the ADRs are accepted, and the rejected designs are acknowledged |
| Real-money beta (P6)? | **NO-GO** until counsel and the PSP confirm in writing who may collect buyer funds (F-01), finance signs off the posting rules, the PSP adapter is specified from the contract, and the payment and admin money-out screens are redesigned |
| Wallet top-up / crypto / credit / in-app exchange? | **NO-GO** — out of scope |

**Round 2 (UI verification):** 20 findings (1 CRITICAL, 8 HIGH, 9 MEDIUM, 2 LOW), independently spot-checked. None required a change to the ADRs, modules, ledger or payment design; all require UI changes or owner decisions (doc 15).

**Phase 0 is not declared complete or approved.** Approval and authorisation of Phase 1 rest with the product owner.
