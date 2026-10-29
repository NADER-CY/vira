# 03 — Product Requirements and Gap Analysis

**Status:** Phase 0 v3 (final for owner acceptance; v2/v3 additions marked) · **v2 (2026-10-28): updated after verification against the original archive — see §8 and doc 15** · **Basis:** doc 02 inventory, Master Requirements v0.1, Phase 0 brief · **Rule:** a screen is evidence of an *idea*, not of an approved requirement.

## 1. Requirement classification

Each candidate requirement is tagged:

| Tag | Meaning |
|---|---|
| **CORE** | Needed for any credible social-commerce launch; recommended for MVP subject to D-02 |
| **LATER** | Valuable, depends on CORE, recommended after MVP |
| **GATED** | Must not be built or enabled until a named owner decision and legal/provider approval exist |
| **CONCEPT** | Pictured in UI but speculative or poorly specified; needs product definition before it becomes a requirement |
| **REJECT?** | Recommended for removal; owner to confirm |

`D-xx` references point to the open-decision register in doc 13.

## 2. Inferred launch context

The UI consistently assumes **Iran**: prices in Toman, Paya/Satna interbank settlement, Sheba (IBAN) binding, Shaparak card switching, national smart ID card, “31 provinces”, Tehran neighbourhoods (Saadat Abad), Jalali dates (“Aban 1403”). The architecture is designed **Iran-first** because the evidence points there, but **jurisdiction is an owner decision (D-01)** and it changes payments, KYC, hosting, app distribution, push delivery and the crypto question. Where Iran-specific choices are made they are isolated behind adapters.

> Not legal advice. Regulatory statements in these documents identify questions for qualified counsel; they are not conclusions.

## 3. Functional requirements by domain

### 3.1 Identity & accounts (IDN, SET)

| ID | Requirement | Tag | Evidence |
|---|---|---|---|
| R-IDN-01 | Sign-up / sign-in with Iranian mobile number + SMS OTP; phone normalisation (`+98`, `09…`, Persian/Arabic digits) | CORE | `otp`, `mobile_otp_verification_light` |
| R-IDN-02 | One person = one platform identity; buyer, creator, seller are **capabilities** on the identity, not separate accounts | CORE | `_224` seller mode |
| R-IDN-03 | Session & device management: list, revoke, “sign out other devices” | CORE | `_77`, `_205` |
| R-IDN-04 | Optional second factor (TOTP) for high-risk actions (payout, Sheba change, seller-staff changes) | CORE | `_59`, `_121`, `_178` |
| R-IDN-05 | Passwords | CONCEPT (D-22) | `_59`, `_121` show “password”; OTP-only is simpler and was implied by the brief |
| R-IDN-06 | Account recovery when the SIM is lost/recycled | CORE — **missing** | none |
| R-IDN-07 | Deactivate and delete account with grace period and financial-obligation checks | CORE | `_206`, `_53` |
| R-IDN-08 | Verified (“blue badge”) programme | LATER | `blue_badge` |
| R-IDN-09 | Appearance (light/dark/system) | CORE | `_61`, `account_settings_light` |

### 3.2 KYC & verification levels (KYC)

| ID | Requirement | Tag | Evidence |
|---|---|---|---|
| R-KYC-01 | Level 0 (phone verified), Level 1 (national ID + phone-owner match), Level 2 (ID card image + liveness) | CORE (levels and limits D-10) | `_184`, `_182`, `level_2_*`, `_181` |
| R-KYC-02 | Limits bound to level (UI: purchases > 10M Toman need L2) — limits are **configuration**, not code | CORE | `buyer_level_1_approved_light` |
| R-KYC-03 | Manual review queue with evidence viewer, reason codes, four-eyes for overrides | CORE | admin `vira_kyc_financial_compliance_dark` |
| R-KYC-04 | AI-assisted document/face analysis | CONCEPT | `_108` — automated decisions on biometrics need policy (D-10) |
| R-KYC-05 | Seller KYC: individual vs legal entity, Sheba ownership match, bank card ownership, tax identity | CORE (D-09) | `_185`, `bank_card_ownership_verification_light`, `_183` |
| R-KYC-06 | AML monitoring & thresholds | LATER (D-01) | admin KYC & AML console |

### 3.3 Social (SOC, STY, RLS, CRT, DISC, MOD)

| ID | Requirement | Tag |
|---|---|---|
| R-SOC-01 | Profiles (personal, creator, store), follow/unfollow, followers lists, private accounts with follow requests | CORE |
| R-SOC-02 | Posts (image carousel, video), captions, hashtags, mentions, **product tags** | CORE |
| R-SOC-03 | Home feed: following-first, chronological + light ranking in MVP | CORE |
| R-SOC-04 | Likes, comments (threaded 1 level), saves & collections | CORE |
| R-SOC-05 | Stories (24 h), viewers list, close friends, highlights, archive, stickers (poll) | CORE (stickers LATER) |
| R-SOC-06 | Reels: vertical short video, share sheet, product tags | CORE |
| R-SOC-07 | Explore & category directory, filters/sort, shop hub | CORE |
| R-SOC-08 | Nearby stores (location-based) | LATER (D-25) |
| R-SOC-09 | Blocks, mutes, restricted DMs, mention/tag controls | CORE |
| R-SOC-10 | Report content/user/store; appeal a restriction | CORE |
| R-SOC-11 | Creator studio with drafts, media crop, multi-select, offline drafts | CORE |
| R-SOC-12 | Recommendation engine with tunable weights | LATER |
| R-SOC-13 | Notifications: in-app inbox + push; per-category preferences | CORE |

### 3.4 Messaging & calls (MSG, CALL)

| ID | Requirement | Tag |
|---|---|---|
| R-MSG-01 | 1:1 DMs with text, images, product cards, read receipts, typing | CORE |
| R-MSG-02 | Message requests from non-followers; buyer→store chat always permitted | CORE |
| R-MSG-03 | Group chats with roles & invite links | LATER |
| R-MSG-04 | Broadcast channels (store discount channels) | LATER |
| R-MSG-05 | Voice/video 1:1 calls; call history | LATER (D-19) |
| R-MSG-06 | In-call product sharing / purchase | CONCEPT |
| R-MSG-07 | Privacy model: E2EE vs server-readable for dispute evidence | **Decision D-11** |

### 3.5 Live shopping (LIVE)

| ID | Requirement | Tag |
|---|---|---|
| R-LIVE-01 | Seller goes live with pinned products; viewers chat and buy without leaving the stream | LATER (D-19) |
| R-LIVE-02 | Host desk: product queue, stock, chat moderation | LATER |
| R-LIVE-03 | Admin monitoring and **emergency stop** | LATER — mandatory before LIVE launch |
| R-LIVE-04 | Network fallback / reconnect | LATER |
| R-LIVE-05 | Scheduling, reminders, replay | LATER — **missing in UI** |

### 3.6 Marketplace (CAT, CART, CHK, ORD, RET, REV, SEL)

| ID | Requirement | Tag |
|---|---|---|
| R-MKT-01 | Store (storefront) per seller, with policies, shipping methods/rates, coverage | CORE |
| R-MKT-02 | Products with variants (colour × size matrix), SKU, stock, low-stock alerts | CORE |
| R-MKT-03 | Global category tree governed by admin | CORE |
| R-MKT-04 | Multi-seller cart; one checkout; **one payment**; split into per-seller orders and parcels | CORE |
| R-MKT-05 | Address book with map pin; delivery estimate per seller | CORE |
| R-MKT-06 | Order lifecycle per seller order with SLA timers | CORE |
| R-MKT-07 | Buyer delivery confirmation; automatic confirmation after N days | CORE |
| R-MKT-08 | **Partial returns** per item, with evidence (photos/video), seller decision, admin escalation | CORE |
| R-MKT-09 | Ratings and reviews (text, photo, **video**); verified-purchase badge | CORE (video LATER) |
| R-MKT-10 | Seller staff with roles (owner, manager, inventory, fulfilment, support, finance) | CORE |
| R-MKT-11 | Seller order command centre, pick/pack lists | CORE |
| R-MKT-12 | Coupons, gift cards, loyalty tiers | LATER |
| R-MKT-13 | Shipping-carrier integration and tracking | CORE (carriers D-24) |
| R-MKT-14 | Instant/same-day delivery from nearby stores | CONCEPT |

### 3.7 Payments, ledger, settlement (PAY, LEDG, STL, WAL, CRY)

| ID | Requirement | Tag |
|---|---|---|
| R-FIN-01 | Card payment through a licensed Iranian PSP via redirect + server-side verify | CORE (D-05) |
| R-FIN-02 | **Payment-unknown** handling: never invite a second payment while the first is unresolved; automatic inquiry | CORE (`_219`, `_223`) |
| R-FIN-03 | Double-entry ledger as financial source of truth | CORE |
| R-FIN-04 | Commission per category/seller tier; fee snapshot on order | CORE (D-07) |
| R-FIN-05 | Settlement hold until delivery + return window, then seller balance becomes withdrawable | CORE **under Model 1 only** — subject to FG-1a (doc 06 §6.5); under Model 2 replaced by PSP split + receivables (D-05, D-06, D-08) |
| R-FIN-06 | Seller/creator withdrawal to verified Sheba; Paya batch cycles; 2FA | CORE |
| R-FIN-07 | Refunds: full/partial; rail per D-08 | CORE |
| R-FIN-08 | Daily three-way reconciliation (ledger ↔ PSP report ↔ bank statement) | CORE |
| R-FIN-09 | Fraud rules (velocity, device, mismatched card owner) and investigation console | CORE (basic) / LATER (advanced) |
| R-FIN-10 | Buyer wallet top-up, stored value | **GATED (D-03)** |
| R-FIN-11 | USDT payment, conversion, TRC20 payout, exchange providers | **GATED (D-04) — default OFF** |
| R-FIN-12 | Affiliate commissions to creators | LATER (D-20) |
| R-FIN-13 | Ad-spend billing for promote campaigns | LATER — **missing in UI** |

### 3.8 Administration

| ID | Requirement | Tag |
|---|---|---|
| R-ADM-01 | Separate admin identity realm, mandatory 2FA, RBAC with granular permission matrix, admin session list | CORE |
| R-ADM-02 | Immutable audit log of every admin read of sensitive data and every write | CORE |
| R-ADM-03 | Maker–checker on money movement, KYC overrides, config affecting fees | CORE |
| R-ADM-04 | Users, sellers, stores, products, orders, refunds, payouts, KYC, moderation queues | CORE |
| R-ADM-05 | Feature flags & remote config; app-version gate | CORE |
| R-ADM-06 | Banners/placements, static pages | CORE (simple) |
| R-ADM-07 | BI, custom report builder | LATER (use read replica / warehouse) |
| R-ADM-08 | NOC/SOC/command-centre consoles | REJECT? — use standard observability tools (Grafana etc.) rather than bespoke screens |
| R-ADM-09 | AI content hub, AI product studio, AI ops cost | LATER / CONCEPT (D-21) |
| R-ADM-10 | “Platform control centre” kill switches | CORE (as flags + runbooks, not a bespoke console) |
| R-ADM-11 | Data archive & backup console | REJECT? — backups are an infrastructure function, not an admin-UI action |
| R-SADM-01 | **Seller Admin web console** (catalog bulk edit, orders, finance, staff) | CORE-candidate — **no UI exists** (D-23) |

## 4. Non-functional requirements (proposed targets — owner to confirm, D-14)

| ID | Area | Target (MVP) |
|---|---|---|
| NFR-01 | API latency | p95 < 300 ms read, < 600 ms write (in-country) |
| NFR-02 | Availability | 99.5% MVP; 99.9% for checkout/payments after Phase 6 |
| NFR-03 | Payment correctness | 0 unreconciled differences older than T+1 business day |
| NFR-04 | Durability | Ledger RPO ≈ 0 for committed transactions (quorum sync replication), ≤ 1 min from WAL archive; RTO ≤ 1 h (doc 06 §11) |
| NFR-05 | Media | Image upload to visible < 5 s; reel upload → playable < 60 s (p90, 60 s clip) |
| NFR-06 | Realtime | Message delivery p95 < 1 s when both online |
| NFR-07 | Localisation | 100% RTL layouts, Persian digits, Jalali dates, ZWNJ-correct text |
| NFR-08 | Accessibility | Screen-reader labels, 4.5:1 contrast, dynamic type up to 200% |
| NFR-09 | Security | OWASP ASVS L2 for APIs; MASVS L1 (+ R for payments) for mobile |
| NFR-10 | Degraded connectivity | Core read paths usable on 3G; no hard runtime dependency on foreign services (see doc 10) |

## 5. Gap analysis

### 5.1 Missing screens / flows

| Gap ID | Missing | Impact | Severity |
|---|---|---|---|
| G-IDN-01 | Account recovery (lost SIM, number recycled) | Account takeover or permanent lockout | HIGH |
| G-IDN-02 | Terms/privacy consent, age gate | Legal. v2: consent is only *implicit* on `otp`; an explicit biometric-consent checkbox exists on `level_2_*`; no age gate anywhere | HIGH |
| G-IDN-03 | OTP errors: wrong code, expired, rate-limited, resend countdown | Abuse + UX | MEDIUM |
| G-SEL-01 | Seller Admin **web** console | Seller productivity, bulk ops. v2: `_228` explicitly references a “Seller Web Admin”, so the design intent exists but no screens do | HIGH |
| G-SEL-02 | Store creation wizard, seller agreement & fee acceptance | Contractual basis for commission | HIGH |
| G-SEL-03 | Seller financial statement (per-order fee breakdown, settlement report, invoices) | Disputes, tax | HIGH |
| G-SOC-01 | Single post view, hashtag page, search results (users/products/posts) | Navigation dead-ends | MEDIUM |
| G-SOC-02 | Activity notifications inbox (vs. notification settings) | **Resolved in v2:** `_12` is the activity centre | — |
| G-SOC-03 | Blocked-user experience; “restricted” state | Safety | MEDIUM |
| G-MKT-01 | Coupon entry at checkout; out-of-stock during checkout; price-changed during checkout | Checkout failure modes | HIGH |
| G-MKT-02 | Seller cancellation / partial fulfilment by seller | Order integrity | HIGH |
| G-MKT-03 | Wishlist / back-in-stock | Engagement | LOW |
| G-FIN-01 | Ad-spend payment & invoice | Cannot charge for promote | MEDIUM |
| G-FIN-02 | Admin: tax/VAT configuration, e-invoice export | Compliance | HIGH (D-07) |
| G-LIVE-01 | Live scheduling, replay, post-live sales report | LIVE completeness | MEDIUM |
| G-ADM-01 | Admin login, 2FA enrolment, password/WebAuthn reset | Security | CRITICAL for Phase 1 |
| G-ADM-02 | Data-subject requests (export/erase) queue | Privacy | MEDIUM |
| G-ADM-03 | Child-safety / urgent-escalation queue distinct from general reports | Safety | HIGH (D-13) |
| G-SYS-01 | Force-update, maintenance mode, permission-rationale (camera/mic/location/notifications), empty states, skeleton loaders | Mobile hygiene | MEDIUM |
| G-SYS-02 | Error state catalogue (each API error code → UI message) | UX consistency | MEDIUM |

### 5.2 Contradictions requiring a product decision

| C-ID | Contradiction | Recommendation |
|---|---|---|
| C-01 | OTP-only login (brief) vs password screens (`_59`, `_121`) | OTP + optional TOTP; no passwords in MVP (D-22) |
| C-02 | Buyer “escrow lock” / smart contract vs no escrow licence | Internal settlement hold; rename copy (D-06) |
| C-03 | Wallet as balance view vs top-up stored value (`wallet_light` “demo top-up”) | Earnings balance only for sellers/creators; buyer refund credit optional; no top-up (D-03) |
| C-04 | Crypto embedded in approved admin consoles | Strip crypto from core consoles; keep only behind GATED module |
| C-05 | “Microservices health grid” vs MVP simplicity | Modular monolith (ADR-001); NOC via standard tooling |
| C-06 | Seller admin on mobile only vs “Seller Admin dashboard” in brief | Mobile seller workspace for daily ops + web console for bulk (D-23) |
| C-07 | Light/dark naming vs content | Theme from tokens; do not trust per-screen theme |
| C-08 | Mobile design docs specify Latin-only fonts | Vazirmatn as primary family (ADR-014) |
| C-09 | `tiktok_shop_hub` uses a third-party trademark in naming | Rename; keep design idea only |

### 5.3 Concept-only features (not to be estimated until specified)

In-call purchase, AI shopping assistant & visual search, AI product studio, AI content hub, smart promote/“auto explore”, regional discovery algorithm tuning console, loyalty tiers, instant delivery from nearby stores, digital signature of seller agreement, “treasury verification” stage.

## 6. Sensitive workflow register

| Workflow | Security-sensitive | Financially sensitive | Personal data |
|---|---|---|---|
| OTP login, session, device revoke | ● | | ● |
| KYC capture, liveness, review | ● | | ●● (biometric, national ID) |
| Sheba binding, bank card ownership | ● | ● | ● |
| Checkout, PSP redirect, verify | ● | ●● | ● |
| Refund, partial return | | ●● | ● |
| Seller payout, creator payout | ●● | ●● | ● |
| Commission/fee configuration | ● | ●● | |
| Admin RBAC, admin impersonation/support view | ●● | ● | ●● |
| Seller staff permissions | ● | ● | |
| Messaging & calls | ● | | ●● |
| Location (nearby, address pin) | | | ● |
| Crypto (all) | ●● | ●● | ● — GATED |

## 7. Recommended MVP boundary (for D-02)

**MVP = Phases P1–P7 in doc 12:** identity + KYC L0/L1, social core (posts, stories, reels, explore, follow, comments, notifications, report), seller onboarding + catalog, multi-seller cart + checkout + one PSP, ledger + settlement hold + payouts with maker–checker, orders/shipping/returns/refunds, buyer↔store chat (1:1 only), Super Admin essentials, Seller workspace (mobile) + minimal Seller web console.

**Not in MVP:** live shopping, calls, groups/channels, ads, affiliate, loyalty, AI features, wallet top-up, crypto, recommendation tuning, BI report builder.

Rationale: every excluded item either multiplies infrastructure cost (live, calls), adds a regulated money flow (wallet, crypto, affiliate payouts, ad billing), or depends on data the platform will not have until it has users (recommendations, AI, BI).

## 8. v2 additions from UI verification (doc 15)

### 8.1 New candidate requirements

| ID | Requirement | Source screens | Tag |
|---|---|---|---|
| R-IDN-10 | Account-type choice at sign-up (social user vs official seller) — sets initial capability, not a separate account | `otp` | CORE |
| R-IDN-11 | Explicit, versioned consent capture (terms, privacy, biometric processing); no implicit “by logging in you accept” | `otp`, `level_2_*` | CORE |
| R-KYC-07 | KYC step embeddable in checkout, resuming checkout on approval (cart kept, stock not held — D-35) | `_89`, `_83` | CORE |
| R-MKT-16 | Seller packing tools: checklist, waybill/barcode scan, label print, tamper seal | `_15`, `_78`, `_128` | CORE |
| R-MSG-08 | Seller-issued pro-forma invoice in chat, paid via the standard checkout | `_110` | LATER |
| R-SOC-14 | Notes (status), shoppable story stickers (poll + product) | `_110`, `_8` | LATER |
| R-SUP-05 | Anti-phishing education (“VIRA never asks for CVV2 / second password”) | `_66`, `_83` | CORE |
| R-FIN-14 | Saved cards only through PSP tokenisation | `_129` | DECISION (D-34) |
| R-FIN-15 | Credit / instalments (BNPL) | `_129`, `_69`, `_96`, `level_2_*` | **GATED** (D-27) |
| R-LIVE-06 | Live virtual gifts | `_101` | **GATED** (D-31) |
| R-SEL-12 | Seller subscription plans (staff seats, features); “VIRA Plus” buyer membership | `_227`, `_231`, `_105` | LATER (D-32) |
| R-MKT-15 | Platform courier dispatch and live tracking | `_106`, `_44`, `_12` | DECISION (D-29) |
| R-PROMO-02 | Coins / cashback rewards | `_12`, `_27`, `_35`, `_76`, `_81` | LATER (D-30) |
| R-PROMO-03 | Gift cards and corporate vouchers | `_152` | **GATED** with wallet (D-03, D-30) |

### 8.2 New contradictions

| C-ID | Contradiction | Recommendation |
|---|---|---|
| C-10 | In-app card entry (`_41`, `_142`; full PAN in `_66`, `bank_card_ownership_verification_light`) vs PSP-hosted payment (A7) | **Reject** the designs; PSP redirect; masked/tokenised card for ownership checks |
| C-11 | USDT pre-selected / recommended (`_164`, `_172`) vs crypto OFF | Remove from MVP designs |
| C-12 | Refund to in-app wallet (`_90`, `_120`, `_124`) or registered IBAN (`_115`, `_197`, `_199`); never to the source card | Refund to source by default; IBAN only with owner match (D-03/D-08, audit F-08) |
| C-13 | Hold release “immediately” (`_63`) / “48 h” (admin) / “after 7-day window” (`_189`) and statutory window 48 h (`_198`) vs 7 days (checkout) | One configurable policy set by counsel (D-06/D-08) |
| C-14 | L2 limits 50M vs 250M; withdrawal 50M/100M/200M | Limits are configuration (D-10) |
| C-15 | OTP 4 digits (`otp`) / 5 digits (`mobile_otp_verification_light`) vs 6 | 6 digits |
| C-16 | Platform courier (`_106`) vs sellers ship (A3) | D-29 |
| C-17 | Cart reserved 24 h during KYC vs reservation until payment deadline | D-35 |
| C-18 | Admin “instant settlement” in 17 console headers, single-password payout batch approval, single-action early release vs maker–checker | Redesign payout, hold-release and refund consoles reusing the dual-control pattern already drawn in the wallet-monitoring console (doc 08 §4.3) |
| C-19 | Admin shows unmasked phone/e-mail (users) and national IDs (orders, stalls, withdrawals); masking inconsistent between list and detail | One masking rule for all views (doc 08 §10) |
| C-21 | Pre-ticked consent/attestation boxes (biometric consent, save card, withdrawal attestation, crypto terms) vs valid explicit consent | Unticked by default (R-IDN-11) |
| C-22 | Admin search by full card number vs no PAN storage (A7) | Remove; search by reference/RRN/masked PAN |
| C-20 | Password + biometric login + “financial PIN” (`_59`, `_178`) vs OTP + TOTP | D-22 |

### 8.3 Concept-only features added

Credit/instalments, live auctions, virtual gifts, lucky wheel (gambling-like — recommended **reject** pending legal review), “VIRA Plus”, platform courier fleet.

### 8.4 Effect on the MVP boundary (§7)

The recommended MVP boundary is **unchanged**. D-02 must now also **explicitly exclude or schedule** these UI-implied items: crypto, credit/BNPL, wallet top-up and gift cards, virtual gifts, seller plans/“VIRA Plus”, coins/cashback, the lucky wheel, the platform courier fleet and live auctions (doc 15 §7, blocker B-2).

## 9. v3 — owner decisions on scope (B-2)

The owner confirmed **full-platform architecture with phased implementation**. MVP priorities: shared identity and authentication; Instagram-like social networking; posts, stories, reels, profiles and interactions; media upload and processing; essential messaging; multi-vendor marketplace; seller onboarding and storefronts; catalogs and inventory; cart and order management; payment infrastructure **subject to separate financial approval (FG-1a model decision before P6; FG-1b real-money activation)**; Seller Admin (web + “Manage My Stall”, shared identity); essential Super Admin; notifications, security, moderation and audit logging.

Long-term capabilities retained (not deleted, not MVP, each behind its own gate in doc 12 §v3): live shopping and streaming; voice/video calling; advanced messaging and channels; advertising and affiliate commissions; AI features; seller subscription plans and VIRA Plus; loyalty and rewards; advanced shipping and logistics; wallet; USDT payments; USDT↔Toman conversion in both directions; eligible seller USDT withdrawals; credit/BNPL, virtual gifts and live auctions as identified advanced capabilities. This supersedes the “Not in MVP” list of §7 where wording differs; the recommended MVP content itself is unchanged.
