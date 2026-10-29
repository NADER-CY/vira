# 15 — UI Verification Report (Phase 0, round 2)

**Status:** Phase 0 · 2026-10-28 · Documentation only — no code, the reference ZIP was not modified, Phase 1 not started.
**Input:** `ویرا.zip` (300,989,389 bytes; SHA-256 prefix `85f73a6f8d90a17e`), extracted read-only into an isolated folder.

## 1. What was verified and how

| Step | Method | Result |
|---|---|---|
| Integrity | SHA-256 of every `code.html` and `DESIGN.md` against the v0.1 manifest | **383 / 383 HTML and 6 / 6 DESIGN.md match exactly** — the manifest-based inventory was built on the true archive |
| Extra content | Files not in the manifest | 2 nested ZIPs (**byte-identical** copies of the extracted trees: 108 and 555 files) and 3 Vazirmatn v33.003 font packages (148 WOFF2 files each) |
| HTML structure (all 330 unique screens) | Parsed every page: theme class, colour tokens, fonts, headers, inputs (type/placeholder/label), buttons, links, navigation bars, inline JS behaviour, sensitive-data keywords | 708 inputs on 194 screens; 8 `<form>` elements; 254 screens with click handlers |
| Rendered theme (all screenshots) | Mean luminance of every `screen.png` (dark < 100 < mixed < 150 < light) | Mobile **158 dark / 116 light / 2 mixed**; admin **48 dark / 5 light** |
| Visual inspection | Contact sheets viewed for the 42 open records + priority financial, identity, checkout, seller and admin screens | **63 screens verified visually + HTML** |
| Independent check | Separate reviewer re-derived key facts from the archive | Core facts reproduced; corrections and 3 additional findings incorporated (§3.2) |
| Screenshot quality | Luminance/variance check | **5 blank (all-black) screenshots** (fade-in animation captured at opacity 0) (`call_history_light_1`, `checkout_light`, `direct_messages_inbox_light_1`, `my_orders_light`, `shopping_cart_light_2`), 1 missing (`_42`), 1 truncated (`_53`) — resolved from HTML |

Verification levels used in the updated CSV (`status_v2`, `verification`):

* **VERIFIED** — purpose confirmed from HTML **and** screenshot (or HTML alone where the screenshot is unusable, flagged).
* **CONSISTENT-HTML** — v1 classification checked against the extracted HTML structure and found consistent; screenshot not individually viewed.
* **NEAR-DUPLICATE / DUPLICATE / GATED** — unchanged meaning.

## 2. Resolution of the 42 open records (+3 reclassifications)

All **17 UNVERIFIED** and **25 CONTRADICTORY** records are resolved; three further records (`_44`, `_89`, `_168`) were found misclassified during the priority sweep. No `UNK` domain remains.

Root cause of the 25 contradictions: the design generator re-used a stale English top-bar label from another screen (e.g. “Product Detail” on security, returns and logout screens; “Shopping Cart” on comments and OTP; “Instant Checkout” on product editing). In every case the **content** was coherent; only the label was wrong. Two folder-name/theme contradictions (`level_2_*_light`) are resolved: both **render light**; the word “dark” appears only in body copy.

| ID | Folder | v1 status / domain | **Verified domain** | Verified purpose | Rendered theme | Screenshot | Notes |
|---|---|---|---|---|---|---|---|
| M-008 | `_8` | UNVERIFIED / UNK | **STY** | Story viewer: poll sticker + shoppable product + reply | dark | ok | Shoppable story |
| M-011 | `_11` | UNVERIFIED / UNK | **CRT** | In-app camera: story/reel/live-shop/post modes, product tag with commission | dark | ok |  |
| M-012 | `_12` | UNVERIFIED / UNK | **NOTIF** | Activity centre: follow requests, live alerts, order status, comments | dark | ok | Resolves G-SOC-02 (notification inbox exists) |
| M-014 | `_14` | UNVERIFIED / UNK | **LIVE** | Live host broadcast view: pinned product, flash offer, guest invite, live sales total | dark | ok |  |
| M-015 | `_15` | CONTRADICTORY / ORD | **SEL** | Seller pack & ship: carrier choice (Tipax/Post/VIRA express), waybill scan, label print, packing checklist, SLA timer | dark | ok | Was ORD |
| M-018 | `_18` | UNVERIFIED / UNK | **AFF** | Creator earnings studio: withdrawable commission, reel sales analytics | dark | ok | Creator payouts |
| M-026 | `_26` | UNVERIFIED / UNK | **DISC** | Explore grid: search (visual/voice), trend chips, hashtags | dark | ok |  |
| M-033 | `_33` | UNVERIFIED / UNK | **SOC** | Home feed: For you / Following / Live shop tabs, stories tray, shoppable posts | dark | ok |  |
| M-037 | `_37` | CONTRADICTORY / SUP | **SUP** | Help centre: topic search, FAQ, my tickets, new-ticket form | dark | ok | Label "account settings" wrong |
| M-044 | `_44` | INFERRED / CHK | **ORD** | My orders list with live courier distance/ETA and rider call | dark | ok | Was CHK; label "Checkout Order" wrong |
| M-047 | `_47` | CONTRADICTORY / MSG | **MSG** | Create broadcast channel: name, handle, description, public/VIP access, auto-publish toggles | dark | ok |  |
| M-050 | `_50` | UNVERIFIED / UNK | **RLS** | Shoppable reel player with buy button | dark | ok |  |
| M-053 | `_53` | CONTRADICTORY / SET | **SET** | Logout confirmation + deactivate/delete entry; shows wallet balance & active orders | dark | truncated | Screenshot truncated (only top third rendered) |
| M-055 | `_55` | UNVERIFIED / UNK | **WAL** | Wallet (DEMO balance 1bn Toman): top-up, transfer, instant settle, cashback, lucky wheel | dark | ok | Stored value + gamification — D-03, D-30 |
| M-059 | `_59` | CONTRADICTORY / SET | **SET** | Security centre: SMS 2FA, authenticator, biometric login, change password, last security event (IP, city), sessions | dark | ok | Password + biometric login — D-22 |
| M-068 | `_68` | CONTRADICTORY / MSG | **MSG** | Channel directory: categories, join, create-store-channel CTA | dark | ok |  |
| M-083 | `_83` | CONTRADICTORY / RET | **KYC** | Level-2 submitted (during checkout): case code, inventory + 35% code held ~3 h, anti-phishing note (VIRA never asks CVV2) | dark | ok | Was RET; reservation during KYC — D-35 |
| M-089 | `_89` | INFERRED / CHK | **KYC** | Inline Level-2 national-ID scan inside checkout (OCR) | dark | ok | Was CHK |
| M-090 | `_90` | CONTRADICTORY / RET | **RET** | Seller return decision: refund to WALLET / exchange / reject→arbitration, evidence photos | dark | ok | Refund-to-wallet conflicts with D-03/D-08 |
| M-101 | `_101` | UNVERIFIED / UNK | **LIVE** | Live viewer: chat, flash sale, buy now, virtual gifts ("diamonds") | dark | ok | Virtual gifting — D-31 |
| M-106 | `_106` | CONTRADICTORY / ORD | **ORD** | Live courier tracking: VIRA rider ("سفیر") on map, speed, ETA, call/chat rider, delivery code | dark | ok | Platform courier fleet — D-29 |
| M-110 | `_110` | UNVERIFIED / UNK | **MSG** | DM inbox: notes, requests, seller pro-forma invoice with "direct payment" in chat, voice msg | dark | ok | Chat invoice payment — new payment initiation path |
| M-120 | `_120` | CONTRADICTORY / RET | **RET** | Seller return decision: refund instantly to customer WALLET / resend / arbitration | light | ok | Refund-to-wallet; duplicate of _90 pattern |
| M-121 | `_121` | CONTRADICTORY / SET | **SET** | Security: SMS OTP, authenticator, active sessions (3), sign out others | light | ok |  |
| M-122 | `_122` | CONTRADICTORY / RLS | **RLS** | Share reel sheet: DM recipients, story, channel, copy link, external apps | light | ok |  |
| M-140 | `_140` | CONTRADICTORY / SEL | **SEL** | Add product step 2/3: media, basics, category, SKU, brand, variants, bulk price | light | ok |  |
| M-141 | `_141` | UNVERIFIED / UNK | **REV** | Product reviews & Q&A: filters, seller pinned reply, verified-buyer badges | light | ok |  |
| M-142 | `_142` | UNVERIFIED / UNK | **PAY** | IMITATION of a bank IPG page inside app: card no., CVV2, expiry, dynamic PIN, "save card in VIRA" | light | ok | CRITICAL: in-app card capture — REJECT design (V-01) |
| M-146 | `_146` | CONTRADICTORY / SET | **SET** | Business profile editor: name, handle, title, bio, links, category, contact, email, gender | light | ok | Gender field — minimise (privacy) |
| M-152 | `_152` | CONTRADICTORY / PROMO | **PROMO** | Gift cards: buy, redeem voucher, send gift card to friend, coupon history | dark | ok | Gift cards = stored value — D-03/D-30 |
| M-168 | `_168` | INFERRED / STL | **PAY** | Checkout payment-method step (Rial or USDT up to 250M/day at L2) | light | ok | Was STL; contains GATED option |
| M-169 | `_169` | UNVERIFIED / UNK | **CRY** | USDT checkout: network choice TRC20/Polygon/ERC20, QR + address, locked rate, "escrow", sandbox | light | ok | GATED |
| M-170 | `_170` | UNVERIFIED / UNK | **CRY** | USDT payment confirmation tracker: block confirmations, TXID, under/over-payment rules | light | ok | GATED |
| M-188 | `_188` | UNVERIFIED / UNK | **AFF** | Creator affiliate overview with own bottom nav (dashboard/products/campaigns/earnings) | dark | ok | Separate creator workspace |
| M-189 | `_189` | UNVERIFIED / UNK | **AFF** | Affiliate catalog: commission %, links/tags, creator sample orders | dark | ok |  |
| M-221 | `_221` | CONTRADICTORY / LIVE | **SYS** | Upload interrupted (504): resume at 78%, local draft protection, save to drafts | light | ok | Was LIVE; label wrong |
| M-228 | `_228` | CONTRADICTORY / SEL | **SEL** | Seller mode hub: switch buyer/seller, seller tier (gold, 2% fee), permission matrix incl. USDT settlement, links to "Seller Web Admin" | light | ok | Evidence for a Seller web console (D-23) |
| M-230 | `_230` | CONTRADICTORY / SEL | **SEL** | Seller alert inbox: SLA alerts, new paid orders, return evidence, chats, payout receipt | light | ok |  |
| M-231 | `_231` | CONTRADICTORY / SEL | **SEL** | Store staff & security: owner/legal rep, staff roles, suspend, invite; plan cap "silver package: 5 users" | light | ok | Seller subscription plans — D-32 |
| M-241 | `checkout_light` | CONTRADICTORY / ORD | **CHK** | Checkout step 2 (address/shipping) with KYC-L1 limit notice; "connect to DEMO gateway" | dark | blank | Screenshot BLANK; resolved from HTML; folder name was right |
| M-242 | `comments` | CONTRADICTORY / SOC | **SOC** | Comments sheet with seller pinned comment + product card, reactions | dark | ok | Top-bar label wrong |
| M-253 | `level_2_identity_verification_introduction_light` | CONTRADICTORY / KYC | **KYC** | Level-2 intro: limits, required docs, biometric consent checkbox; promises credit purchase, instalments, live auctions | light | ok | Rendered LIGHT (heading text mentions dark); credit/auction concepts — D-27/D-33 |
| M-255 | `level_2_verification_pending_light` | CONTRADICTORY / KYC | **KYC** | Level-2 pending: 4-step status, cart reserved 24 h | light | ok | Rendered LIGHT; 24 h reservation conflicts with reservation design — D-35 |
| M-263 | `otp` | CONTRADICTORY / IDN | **IDN** | Welcome: phone + 4-digit OTP + account type (social user / official seller) + implicit terms acceptance | dark | ok | OTP length 4 vs 6; no age gate; implicit consent |
| M-272 | `shopping_cart_light_2` | CONTRADICTORY / ORD | **CART** | Cart step 1 with coupon field | dark | blank | Screenshot BLANK; resolved from HTML |

## 3. Findings against the current architecture

Severity uses the doc 14 scale. “Arch. impact” states whether the **architecture** must change (it mostly must not — the UI must).

| ID | Sev | Finding | Evidence | Arch. impact | Disposition |
|---|---|---|---|---|---|
| **V-01** | **CRITICAL** | **Card data is captured inside the app.** `_41` is a VIRA-branded “payment gateway” (card number, CVV2, expiry, dynamic PIN, captcha; labelled sandbox/test card); `_142` imitates a bank IPG page (Bank Saman and Shaparak named in text) with card fields and a **“save card in VIRA account” box that is ticked by default**. `_66` (which also offers a **camera scan of the card**) and `bank_card_ownership_verification_light` take the full 16-digit card number. `_92` tells the user to enter a dynamic PIN within the VIRA flow. Only these screens contain card-entry fields (independently re-checked over all 330). | Parsed inputs (`maxlength=19` PAN, `password` CVV2, month/year, dynamic PIN); screenshots | **None** — doc 08 §1 / A7 already forbid it. The designs must be **rejected** | `_41`, `_142` → REJECTED; replaced by redirect to the PSP-hosted page (doc 07 §4.3). Ownership checks: card entered only into a PSP/inquiry-provider flow, or masked + tokenised immediately; VIRA keeps the inquiry result and first-6/last-4 only. Saved cards only via PSP tokenisation (D-34), never pre-ticked |
| **V-02** | HIGH | **Crypto is promoted, not gated.** USDT is **pre-selected** in checkout (`_164`: JS `selectedMethod = 'tether'`, 1 % discount badge) and **pre-selected and “recommended • instant”** for withdrawals (`_172`, `_177`, with USDT as “default currency”); USDT checkout with TRC20/Polygon/ERC20 (`_169`, `_170`, `_168`); refunds console signs **hot-wallet** outflows; KYC console grants “unlimited USDT exchange”; one admin sidebar (`vira_usdt_networks_exchange_providers_dark`) lists cold storage, hardware wallets, multi-signature keys and cross-border OTC providers. 78 screens mention USDT/Tether. | Parsed JS/HTML, screenshots | None — ADR-019 and doc 08 §7 hold | Remove every USDT element from MVP designs; D-04 stays OFF; default payment method is the card IPG |
| **V-03** | HIGH | **A regulated credit product is designed in.** “4 instalments, no guarantor” (`_129`), instalment wallet (`_96`), instalment access promised at KYC (`_40`, `level_2_*`, `buyer_level_1_approved_light`, `account_settings_light`), admin “BNPL bank facilities” and an orders filter for “VIRA Pay organisational credit (BNPL)”. (`_69` “credit wallet” may mean stored credit only.) | Text | **Gap** — no ledger, risk or licensing design | New decision **D-27** (default OFF, gated track G3) |
| **V-04** | HIGH | **Refund destinations are mostly an in-app wallet, and the wallet is stored value.** Seller decisions refund “immediately to the customer’s wallet” (`_120`, `_124`) or to the wallet after physical inspection (`_90`); `_5` shows a (demo) wallet credit; `_197` lets the buyer choose **wallet or registered IBAN (Paya)**; `_115`/`_199` pay to a registered IBAN — none refunds to the **source card**. The wallet supports top-up, transfer, cashback (`_55`, `wallet_light`); gift cards are bought and sent (`_152`); `_92` offers wallet payment “without dynamic PIN”. | Text, buttons | Conflicts with the D-03/D-08 recommendations (refund to source, no top-up) | Owner rules on D-03 + D-30 before P6; architecture default unchanged (refund to source; IBAN only with owner match — audit F-08) |
| **V-05** | HIGH | **Hold and return-window policy contradicts itself.** Seller funds released *immediately after delivery confirmation* (`_63`), *48 h after delivery* (admin settlement console), or *after the 7-day return window* (`_188`, `_189`, `_193`, `_194`). Buyer side: “statutory test & withdrawal right **48 h**”, extendable to **72 h** (`_198`), vs a “7-day money-back guarantee” on 25+ screens, which `_188`/`_194` also call the *statutory* test period. | Quoted text | Confirms D-06/D-08 are open and urgent | Values become configuration; legal value set by counsel (L-03) |
| **V-06** | HIGH | **The money-out paths in the admin UI bypass the control model, although a correct pattern already exists in the same pack.** An “instant settlement” (تسویه فوری) button sits in the header of **17 of 53 consoles**, including non-finance ones (AI studio, stories, comments, BI); the payout batch is approved by **one password** and sent “directly to the Paya web service”; held funds can be **released early** by a single privileged action; the payout, hold-release and refund consoles have no second approver. By contrast, `vira_wallet_monitoring_risk_dark` models **dual control** (reason field + senior approval + CFO inbox), several consoles carry reason fields and 2FA on critical switches — that pattern should be reused. PII: `vira_users_management` shows full phones and e-mails; **national IDs are unmasked** in orders, stalls and withdrawals consoles, and masking is inconsistent between list and detail views. | Buttons/inputs in admin consoles; screenshots; independent re-check | None — doc 08 §4.3 maker–checker and §10 masking stand | Redesign payout, hold-release and refund consoles on the existing dual-control pattern before P6/P7; one masking rule for all views |
| V-07 | MEDIUM | **KYC and withdrawal limits contradict each other**: L1 10M Toman (consistent); L2 daily purchase 50M **and** 250M; daily withdrawal 50M / 100M / 200M; a **third tier (L3 corporate/VIP-B2B, “unlimited”, OTC)** in `_182`, `_184` and the KYC console. | Text on 22+ screens | None (limits are config, R-KYC-02) | Owner sets tiers and values (D-10); L3 crypto/OTC privileges out of scope |
| V-08 | MEDIUM | **OTP length**: 4 digits (`otp`), 5 digits (`mobile_otp_verification_light`); architecture specifies 6. | Text, 4 `maxlength=1` inputs | None | Keep 6 digits; UI to change |
| V-09 | MEDIUM | **Navigation is not specified anywhere.** No link points to another screen (only `#`, in-page fragments, `tel:` and one external explorer link); **14–15 bottom-nav variants** on ~50 tabbed screens (count depends on the extraction rule); **46 distinct sidebars** across 53 admin consoles. The most frequent tab bar (19–20 screens, **under 40 %**) is Home · Explore · ＋ · Reels · Profile with Shop in the top bar; a competing light-theme variant (4 screens) uses Home · Explore · ＋ · **Shop** · Profile. | Nav extraction, independently reproduced | Doc 09 IA updated to the most frequent pattern, explicitly provisional | D-28 decides; no majority exists in the designs |
| V-10 | MEDIUM | **Platform courier fleet.** VIRA riders (“سفیر”) with live map, speed and ETA (`_106`, `_44`, `_12`); “VIRA express” as a carrier option (`_15`). | Text, screenshots | Contradicts assumption **A3** (sellers ship) | D-29; A3 kept for MVP, flagged |
| V-11 | MEDIUM | **Money flows absent from the architecture:** live **virtual gifts** (`_101`); **seller subscription plans** (“silver package: 5 users”, `_227`/`_231`); **“VIRA Plus”** membership; **seller-issued invoice paid in chat** (`_110`); **coins** and cashback rewards (`_12`, `_27`, `_35`, `_76`, `_81`). | Text | Gap — each needs ledger rules | D-30, D-31, D-32; chat invoice → same checkout path (`source=chat_invoice`) |
| V-12 | MEDIUM | **Design-system evidence:** themes are not paired screen-by-screen (each domain mixes light and dark designs of different screens; ADS, AI and PROMO exist only dark); **162 / 277** mobile screens never reference Vazirmatn (how many use it as the *primary* family depends on the measure: 30–82); primaries in use: `#ba0038`, `#ffb2ba`, `#8e0028`, `#ff1753`, `#ff2d65`; admin pages declare a light token set (`#faf9ff`) but render dark through body classes. | Token and luminance extraction | Confirms ADR-014, DS-01/02 | Doc 09 updated |
| V-13 | MEDIUM | **Gambling-like mechanics**: daily “lucky wheel” (`_55`) awarding cash coupons. | `_55` | Gap | D-30 (legal review) |
| V-14 | MEDIUM | **Additional authentication factors**: biometric login and **password** (`_59`), a separate **“financial PIN”** for withdrawals (`_178`). | Text, inputs | Touches D-22 | D-22 extended |
| V-15 | MEDIUM | **Long reservations during KYC**: cart held **24 h** while L2 is reviewed (`level_2_verification_pending_light`), ~3 h with a discount code (`_83`). | Text | Conflicts with reservation design | D-35 |
| V-16 | LOW | Admin invite form and admin list use e-mails at **`vira.gov.ir`** — a government domain. | `vira_administrators_rbac_dark` | None | Replace; never use |
| V-17 | LOW | Evidence quality: 5 blank screenshots — root cause `<html style="opacity:0">` fade-in captured at frame 0 (fixable by re-export with animations disabled) — plus 1 missing and 1 truncated (`_53`). | §1, independent review | None | Re-export if kept |
| **V-18** | HIGH | **Consent and attestations are pre-ticked**: biometric-processing consent (`level_2_identity_verification_introduction_light`), “save card in VIRA” (`_142`), withdrawal-address ownership attestation (`_178`), acceptance of crypto rate/irrevocability (`_163`). Pre-ticked boxes are not valid consent. | `<input type=checkbox checked>`; screenshots | None — reinforces R-IDN-11 | All consent and attestation controls unticked by default; consent versioned and logged |
| **V-19** | HIGH | **In-app fiat→USDT exchange**, beyond accepting USDT: `_163` buys Tether from the Toman wallet (250M/day, 0.25 % fee, irrevocable), “buy Tether” on `_158`/`_160`, run from the admin conversions console. This is exchange/custody activity — a different regulatory class. | `_163`, `_158`, `_160`, `vira_wallets_conversion_dark` | None — gated under ADR-019 | Out of scope; explicitly named in D-04 |
| **V-20** | HIGH | **Admin search by full card number** (`vira_payments_management_dark`) implies server-side storage of card numbers (other consoles show first-6/last-4). | Search placeholder | None — contradicts A7 / doc 08 §1 | Remove; payment search by reference, RRN, masked PAN only |

### 3.1 Confirmations (UI agrees with the architecture)

* Payment-unknown handling (“do not pay again”, live inquiry, automatic return within 72 h) — `_16`, `_219`, `_223` match doc 07 §4.3.
* Multi-seller cart → per-store parcels and partial returns with photo/video evidence — matches doc 06 order model.
* Account deletion with a **30-day** cancellation window and a financial-obligation check (`_206`) — matches doc 08 §10.
* Sessions/devices list and “sign out other devices” — matches R-IDN-03.
* Wallet-adjustment console (`vira_wallet_monitoring_risk_dark`) already models dual control with a reason field and CFO approval — the pattern to reuse for payouts, hold releases and refunds.
* Seller staff roles (owner, order/packing manager, inventory, CRM/support) with suspend and audit history (`_227`, `_231`) — matches the store-scoped role model (doc 08 §4.2).
* Seller mode as a switch inside one account (`_224`, `_228`) and an explicit reference to a **“Seller Web Admin”** (`_228`) — supports D-23.
* Admin users console shows a **Shahkar** verification rate — supports the Shahkar-class dependency (doc 13 §4, audit F-09).
* Offline drafts, resumable uploads and a pending-action queue (`_216`, `_217`, `_221`) — matches doc 04 §7 client outbox.
* Platform settings console uses versioned publish with preview and rollback — matches the flags/config design.
* Withdrawal confirmation uses OTP/TOTP and an ownership attestation — consistent with step-up (doc 08 §3).

## 3.2 Independent check of this report

A separate reviewer re-derived the key facts directly from the archive without reusing the architect’s scripts: integrity (all hashes), theme counts, blank screenshots, Vazirmatn count, admin sidebars, absence of cross-screen links, input counts, V-01, V-02, V-03, and **10 / 10 randomly sampled resolutions** were reproduced. The review corrected V-04, V-05, V-06 and V-09 (wording and citations — dispositions unchanged) and added V-18, V-19 and V-20, all incorporated above.

## 4. Missing requirements identified

| ID | Requirement | Source | Tag |
|---|---|---|---|
| R-MSG-08 | Seller-issued pro-forma invoice in chat, paid through the standard checkout | `_110` | LATER |
| R-LIVE-06 | Live virtual gifts | `_101` | GATED (D-31) |
| R-SEL-12 | Seller subscription plans limiting staff seats and features | `_227`, `_231` | LATER (D-32) |
| R-MKT-15 | Platform courier (“سفیر”) dispatch, live tracking, delivery code | `_106`, `_44` | DECISION (D-29) |
| R-MKT-16 | Packing checklist, waybill scan, label printing, tamper seal | `_15`, `_78`, `_128` | CORE (seller tooling) |
| R-PROMO-02 | Coins / cashback rewards for reviews and purchases | `_12`, `_27`, `_76` | LATER (D-30) |
| R-PROMO-03 | Gift cards and corporate vouchers | `_152` | GATED with wallet (D-03/D-30) |
| R-FIN-14 | Card-on-file via PSP tokenisation | `_129` | DECISION (D-34) |
| R-FIN-15 | Credit / instalments (BNPL) | `_129`, `_69`, `_96` | GATED (D-27) |
| R-IDN-10 | Account-type choice at sign-up (social user vs official seller) | `otp` | CORE |
| R-IDN-11 | Explicit consent capture (terms, privacy, **biometric processing**) with versioning | `otp` (implicit), `level_2_*` (checkbox) | CORE |
| R-KYC-07 | KYC step embeddable inside checkout, resuming checkout on approval | `_89`, `_83` | CORE (with D-35 limits) |
| R-SOC-14 | Notes (status), shoppable story stickers (poll + product) | `_110`, `_8` | LATER |
| R-SUP-05 | Anti-phishing education (“VIRA never asks for CVV2 / second password”) | `_66`, `_83` | CORE |

## 5. New open decisions (added to doc 13)

| ID | Decision | Recommendation |
|---|---|---|
| D-27 | Credit / instalments / BNPL | Out of MVP; gated track **G3**; requires lending licence or bank partnership, credit-risk design, ledger rules |
| D-28 | Primary mobile navigation | Provisionally the most frequent designed pattern (Home · Explore · ＋ · Reels · Profile; Shop in top bar) — no majority exists; validate with users |
| D-29 | Platform courier fleet vs seller-shipped | MVP: seller-shipped + carrier integration (A3); platform express courier only as a later, separately-designed logistics product |
| D-30 | Loyalty: coins, cashback, gift cards, lucky wheel | Coins/cashback as non-cash points (later); gift cards gated with D-03; **lucky wheel rejected** pending legal review |
| D-31 | Live virtual gifts | Gated; virtual currency is stored value |
| D-32 | Seller plans and “VIRA Plus” membership | Later (P9); recurring billing needs ledger + invoicing |
| D-33 | Live auctions | Concept; out of scope |
| D-34 | Saved cards | Only PSP-tokenised card-on-file if the PSP offers it; never stored by VIRA |
| D-35 | Holding stock during KYC review | No stock reservation beyond the payment deadline; KYC pending keeps the **cart**, not the inventory |

Existing decisions with new evidence: D-03 (wallet: UI assumes stored value), D-04 (crypto: UI promotes it), D-06/D-08 (four different hold/return values), D-10 (contradictory limits), D-22 (password, biometric, financial PIN), D-23 (UI references a Seller Web Admin), D-24 (VIRA express carrier).

## 6. Effect on existing architecture decisions

| Decision | Status after verification |
|---|---|
| ADR-001…022 | **Unchanged.** No evidence requires a technology or structural change |
| Card data never touches VIRA (A7, doc 08 §1) | **Reaffirmed**; four screens rejected/redesigned |
| Crypto/wallet gate (ADR-019) | **Reaffirmed**; scope of UI removal enlarged |
| Ledger posting rules (doc 06 §6.2a) | Unchanged for MVP; V-03, V-11 add future rule sets (gated) |
| Navigation IA (doc 09 §4) | **Changed provisionally** to the most frequent designed pattern (V-09), pending D-28 |
| Assumption A3 (sellers ship) | Kept for MVP; flagged *contradicted by UI* pending D-29 |
| Reservation semantics (doc 06 T1/T3) | Unchanged; D-35 records the conflict |
| OTP 6 digits (doc 07 §4.1) | Unchanged; UI to change |

## 7. Remaining Phase 0 blockers (v2 — superseded by doc 13 §0 and doc 16 v3)

| # | Blocker | Owner | Status |
|---|---|---|---|
| B-1 | Launch jurisdiction and residency (D-01) | Product owner | Open |
| B-2 | MVP boundary (D-02) — must now also rule explicitly on the UI-implied features: crypto incl. in-app exchange (V-02, V-19), credit (V-03), wallet/gift cards (V-04), courier (V-10), gifts/plans/coins (V-11) | Product owner | Open |
| B-3 | Traffic, budget and **team re-baseline** (D-14, audit F-11) | Product owner | Open |
| B-4 | Mobile distribution incl. iOS feasibility (D-16) | Product owner + counsel | Open |
| B-5 | Hosting provider (D-17) | Product owner | Open |
| B-6 | Acceptance or amendment of ADR-001…022 | Product owner | Open |
| B-7 | Acknowledge rejection of in-app card capture and admin card-number search (V-01, V-20), pre-ticked consents (V-18), and single-actor money-out actions in admin (V-06) | Product owner + design | Open |
| — | UI verification (audit F-10) | Architect | **Closed** — all 42 records resolved; integrity verified; priority screens inspected |

Not Phase 0 blockers but gates for later phases: F-01 (collect vs split) and D-06/D-08/D-03 before **P6 design freeze**; V-06 admin redesign before **P6/P7**; D-29 before **P7**.

## 8. GO / NO-GO (v2 — superseded by doc 16 v3)

| Question | Recommendation | Basis |
|---|---|---|
| Phase 0 package ready for product-owner review | **GO** | Evidence gap closed; all records resolved; architecture unchanged where not contradicted |
| Phase 0 complete | **NO-GO** until B-1…B-7 are resolved | §7 |
| Phase 1 (foundations) | **Conditional GO** once B-1…B-7 are resolved. None of the round-2 findings changes a Phase 1 foundation | §6 |
| P6 real-money beta | **NO-GO** (unchanged) + V-01 redesign, V-05 policy values and V-06 admin redesign required | doc 14 |
| Crypto, credit, wallet top-up, virtual gifts | **NO-GO** — out of scope pending D-04, D-27, D-03, D-31 | ADR-019 |

Approval rests solely with the product owner. No implementation has started.
