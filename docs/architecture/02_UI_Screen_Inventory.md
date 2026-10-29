# 02 — Complete UI Screen Inventory (v2 — verified against the original archive)

**Document status:** Phase 0 · **v2, 2026-10-28** (v1 was built from the manifest only) · **Machine-readable companion:** `02a_VIRA_Screen_Inventory.csv` (330 rows; v1 columns kept, v2 columns added) · **Verification report:** doc 15

## 1. Evidence base

**v2:** the original archive `ویرا.zip` was supplied and inspected read-only. Every one of the 383 `code.html` and 6 `DESIGN.md` files matches the v0.1 manifest hash exactly; the archive additionally contains two nested ZIPs (byte-identical copies of the extracted trees) and three Vazirmatn v33.003 font packages.

| Evidence now used | Coverage |
|---|---|
| Titles and headings (v1) | 330 / 330 |
| Parsed HTML: inputs, buttons, links, nav bars, colour tokens, fonts, inline JS behaviour, sensitive-data keywords | 330 / 330 |
| Rendered theme measured from screenshot luminance | 329 / 330 (`_42` has no screenshot) |
| Screenshot inspected visually (with HTML) | 63 screens: all 42 previously open records + priority financial, identity, checkout, seller and admin screens |
| Unusable screenshots | 5 blank (all black), 1 missing, 1 truncated — resolved from HTML |

`status_v2` meanings: **VERIFIED** (HTML + screenshot, or HTML where the screenshot is unusable, flagged), **CONSISTENT-HTML** (v1 classification checked against parsed HTML), **NEAR-DUPLICATE / DUPLICATE / GATED** as before.

## 2. Inventory verification against the brief

| Claim in the brief | Verified value | Result |
|---|---|---|
| 277 mobile HTML screens | 277 `code.html` under `stitch_vira_social_commerce_app` | ✅ Match |
| 106 admin HTML files = 53 duplicated pairs | 106 files; 53 folder names × 2 trees (`stitch_vira_admin_light_theme/` and `super-admin/`); **all 53 pairs byte-identical** | ✅ Match — the second tree adds nothing |
| 382 screenshots | 382 `screen.png`; 383 HTML → 1 missing | ✅ Match |
| 6 design documents = 4 distinct versions | 6 `DESIGN.md`; 2 admin docs duplicated across trees → 4 distinct hashes | ✅ Match |
| Missing screenshot (unidentified in v0.1) | `stitch_vira_social_commerce_app/_42/code.html` | ✅ Identified — and `_42` is **byte-identical to `_17`** (Edit Profile), so nothing is lost |
| 232 numbered folders | 231 numbered (`_1`…`_231`) + 46 named folders = 277 | ⚠️ Correction: v0.1 said 232 |

**Unique screen count:** 277 mobile + 53 admin = **330 unique HTML screens**, of which 1 mobile screen is an exact duplicate (`_42`) → **329 distinct designs**.

## 3. Summary by status (mobile, 277)

| v1 status | Count | → v2 status | Count |
|---|---:|---|---:|
| CONFIRMED | 37 | VERIFIED | 59 |
| INFERRED | 143 | CONSISTENT-HTML | 162 |
| NEAR-DUPLICATE | 40 | NEAR-DUPLICATE | 39 |
| DUPLICATE | 1 | DUPLICATE | 1 |
| CONTRADICTORY | 25 | — (all resolved → VERIFIED) | 0 |
| UNVERIFIED | 17 | — (all resolved) | 0 |
| GATED | 14 | GATED | 16 |

GATED rose from 14 to 16 because `_169` and `_170` (previously unverified) are USDT checkout screens. NEAR-DUPLICATE fell by one because `_66` (KYC card registration) carries a distinct security finding (V-01) and is kept as its own VERIFIED record. Super Admin: 8 VERIFIED, 42 CONSISTENT-HTML, 2 GATED, 1 NEAR-DUPLICATE.

The 25 contradictions were caused by one defect: **stale English top-bar labels** copied from other screens. Content was coherent in every case. The resolution of each record is in doc 15 §2 and in the CSV (`domain_v2`, `feature_v2`, `notes_v2`, `label_defect`).

## 4. Summary by domain

| Code | Domain | Mobile (v2) | Admin | Notes |
|---|---|---:|---:|---|
| IDN | Identity, OTP, account | 3 | 1 | Thin: no registration, recovery, or SIM-change flows |
| KYC | Verification levels, ID card, liveness | 19 | 1 | Biometric + national-ID data; heavy |
| SET | Settings, privacy, security, sessions | 16 | – | Several imply passwords (contradicts OTP-only) |
| SOC | Feed, profile, followers, comments, saved | 9 | – | Feed/profile core screens have no headings (image-heavy) |
| STY | Stories viewer/editor/highlights | 8 | 1 (MOD) | |
| RLS | Reels player/share | 3 | – | |
| CRT | Creator studio, publish, media picker | 15 | – | |
| DISC | Explore, categories, search filters, nearby, shop hub | 10 | 3 | Location features |
| MSG | DMs, groups, channels, requests | 14 | – | |
| CALL | Voice/video calls, history | 7 | – | Includes in-call purchase concept |
| LIVE | Live shopping viewer/host | 6 | 1 | |
| CAT | Storefront, product detail, shipping rates | 15 | 3 | |
| REV | Ratings & video reviews | 5 | – | |
| CART | Multi-seller cart | 7 | – | |
| CHK | Checkout, address, payment result | 12 | – | 4 demo/test-payment states |
| PAY | Payment method, failure, recovery | 7 | 2 | |
| ORD | Orders, shipments, tracking | 13 | 2 | |
| RET | Returns & refunds (buyer & seller) | 9 | 1 | Partial returns; settlement lock |
| WAL | Wallet | 5 | 1 | Scope unapproved |
| STL | Seller/creator withdrawals, Sheba binding | 9 | 1 | Financial |
| CRY | USDT conversion/payment/payout | 16 | 2 | **GATED** |
| LEDG | Commission split, reconciliation | – | 2 | “Escrow” terminology to be replaced |
| SEL | Seller workspace (mobile), staff, orders | 24 | 2 | **No Seller Admin web UI exists** (`_228` links to one) |
| ADS | Promote campaigns, ad transparency | 8 | 1 | |
| AFF | Creator affiliate & commission | 11 | – | Creator payouts |
| PROMO | Gift cards, coupons, discounts | 1 | 1 | |
| AI | Shopping assistant, visual search, studios | 4 | 3 | |
| SUP | Help, tickets, disputes | 9 | 2 | |
| MOD | Reports, appeals, moderation consoles | 4 | 4 | |
| NOTIF | Notifications | 2 | 1 | Admin references FCM |
| SYS | Offline, network error, drafts | 6 | – | Offline write queue implied |
| ADM/SEC/CFG/OPS/ANL/CMS/GROW/RISK | Platform administration | – | 19 | Super Admin only |
| UNK | Unverified | 0 | – | All resolved in v2 |

## 5. Material findings from the inventory

1. **Seller Admin has no web UI.** All seller operations (catalog, variants, orders, staff permissions, payouts, alerts — 23 screens) are designed as **mobile** screens inside the consumer app (`_224` “seller mode”). The admin archive contains only Super Admin. Bulk catalog and fulfilment work on a phone is a known usability ceiling; a Seller Admin web console is listed as a gap (doc 03, G-SEL-01).
2. **Theme coverage is uneven (measured in v2).** Rendered from screenshots: mobile **158 dark / 116 light / 2 mixed**. Folder names and headings are unreliable theme signals (76 mobile screens without `class="dark"` render dark via body classes; the two `level_2_*_light` screens render light). There are no systematic light/dark pairs; dark mode must be produced from tokens (doc 09).
3. **Admin renders 48 dark / 5 light** (light: users, sellers, stalls, both dashboards) although its tree is named `stitch_vira_admin_light_theme`; 25 admin pages declare a *light* token set yet render dark through body classes, and **46 different sidebars** exist across 53 consoles.
4. **Font evidence contradicts a Persian-first product.** Parsed HTML: **162 of 277** mobile screens never reference Vazirmatn (Persian falls back to the device font); how many use it as the *primary* family depends on the measure (30–82). All admin pages load Vazirmatn. The archive itself ships Vazirmatn v33.003, which supports ADR-014.
5. **Demo states are designed in.** `_76`, `_92`, `_100`, `_153`, `wallet_light`, `_113` show “test/demo payment”, “sandbox troubleshooting”, “demo top-up”. These must not ship; they indicate the PSP-sandbox experience and are mapped to the staging environment only.
6. **Crypto is pervasive, not isolated.** Besides 14 mobile GATED screens and 2 admin consoles, USDT appears inside otherwise-approved admin consoles: payments, refunds, reconciliation, platform settings and static content. Gating must be enforced at the API and ledger level, not by hiding screens (doc 08 §7).
7. **“Escrow”, “smart contract”, “Escrow Lock Telemetry”, “Protocol v4.2/v4.8”** appear in admin and buyer screens. These imply a blockchain escrow and a regulated escrow service. The architecture uses an internal **settlement hold** in a double-entry ledger; user-facing copy must change once legal terminology is approved (D-06).
8. **External runtime dependencies in every screen:** Google Fonts (1,305 refs), images hot-linked from `lh3.googleusercontent.com` (1,003), Tailwind Play CDN (383), jsDelivr (4). None is acceptable in production; for an Iran-hosted service they are also availability risks (doc 10 §2).
10. **In-app card capture (v2, CRITICAL):** `_41` and `_142` collect card number, CVV2, expiry and dynamic PIN inside the app (one imitates a bank IPG page and pre-ticks “save card in VIRA”); `_66` and `bank_card_ownership_verification_light` take the full card number; an admin console searches by full card number. These designs are rejected (doc 15 V-01, V-20).
13. **Pre-ticked consents (v2):** biometric consent, save-card, withdrawal attestation and crypto terms are ticked by default (doc 15 V-18).
11. **UI-implied products outside the architecture (v2):** credit/instalments (BNPL), in-app wallet top-up and gift cards, live virtual gifts, seller subscription plans, “VIRA Plus”, coins/cashback, a lucky wheel, a platform courier fleet, and live auctions (doc 15 V-03, V-04, V-10, V-11, V-13).
12. **Navigation is undefined (v2):** no link points to another screen, 14–15 bottom-nav variants with no majority, 46 admin sidebars (doc 15 V-09).
9. **Generation artifacts** leak into content: admin titles contain “Phase 10 • Page 34”, “page 42”, “phase 14”; NOC screen titled “Core Microservices Health Grid”. These are design-tool artifacts, not requirements, and do not commit the architecture to microservices.

## 6. Journeys reconstructed from the inventory

| # | Journey | Screens (representative) | Completeness |
|---|---|---|---|
| J1 | Onboarding & OTP login | `otp`, `mobile_otp_verification_light` | **Partial** — account-type choice exists; OTP shown as 4 and 5 digits; consent only implicit; no recovery, no age gate |
| J2 | Feed → post → comments → profile → follow | `home_feed_light`, `_33`, `comments`, `user_profile_light`, `_109`, `_12` (activity inbox) | Partial — no single-post view, no hashtag/user search results |
| J3 | Create post / reel / story | `create_post_light`, `create_reel_light`, `_91`, `_130`, `_137`, `_93`, `_52`, `_23`, `_151` | Good |
| J4 | Watch reel/story → tagged product → buy | `reels_light`, `_50`, `_8`, `_147`, `product_details_light`, `_28` | Good (buy button on reel and story) |
| J5 | Browse → storefront → product → cart (multi-seller) | `shop_hub_light`, `_2`, `_60`, `_54`, `seller_storefront_light`, `_49`, `multi_vendor_cart_split_shipping_dark` | Good |
| J6 | Checkout → PSP → result | `checkout_light`, `_69`, `_74`, `_80`, `_164`, `_132`, `_16`, `_129`, `_219`, `_223` | Good **except** the payment step: `_41`/`_142` capture cards in-app (rejected); USDT is the default method in `_164` |
| J7 | Order tracking → delivery confirmation | `my_orders_light`, `_138`, `_25`, `_154`, `_196`, `_198`, `_200` | Good |
| J8 | Return/refund (buyer) | `_148`, `_197`, `_124`, `_5`, `_115`, `_199` | Good — but refunds are paid to an in-app wallet (D-03/D-08) |
| J9 | Return decision (seller) | `_90`, `_120` | Partial |
| J10 | KYC level 1 → level 2 (incl. inside checkout) | `_184`, `_180`, `level_2_*`, `_181`, `_108`, `_73`, `_89`, `_83`, `buyer_level_1_approved_light` | Good — L1 limit 10M Toman consistent; L2 limits contradictory (50M vs 250M) |
| J11 | Seller onboarding | `_224`, `_183`, `_185`, `bank_card_ownership_verification_light`, `_187` | **Partial** — no store-creation wizard, no seller agreement acceptance |
| J12 | Seller catalog & inventory | `_34`, `_140`, `_31`, `_126`, `_139`, `product_management_light` | Good (mobile only) |
| J13 | Seller fulfilment | `_63`, `_78`, `_128`, `_226`, `_229`, `_85` | Good (mobile only) |
| J14 | Seller/creator payout | `_7`, `_168`, `_172`, `_177`, `_178`, `_174`, `_176`, `_190`, `_195` | Good, with 2FA |
| J15 | Messaging (DM, groups, channels) | `direct_messages_inbox_light_*`, `private_chat_light`, `_58`, `group_chat_light`, `_19`, `_47`, `_68`, `channel_feed_light` | Good |
| J16 | Calls | `incoming_*_call_light`, `active_video_call_light`, `_30`, `call_history_*` | Good |
| J17 | Live shopping (host & viewer) | `_103`, `_136`, `_14`, `live_shopping_light`, `_87`, `_101` | Partial — no pre-live scheduling, no post-live replay; virtual gifts in viewer |
| J18 | Promote & ads | `_10`, `_29`, `_45`, `_113`, `_56` | Partial — no billing/top-up for ad spend |
| J19 | Affiliate | `_191`, `_193`, `_194`, `_107`, `_190` | Partial — no seller-side affiliate program setup |
| J20 | Report/appeal | `_210`, `_213`, `_212`, `_208` | Good |
| J21 | Support & disputes | `_209`, `_127`, `_98`, `_211`, `_112` | Good |
| J22 | Account security & deletion | `_77`, `_205`, `_206`, `_204` | Good |
| J23 | Offline/resilience | `_216`, `_217`, `_218`, `_220`, `_221` | Good — implies client-side outbox |
| J24 | Creator affiliate workspace | `_188`, `_189`, `_18`, `_191`, `_193`, `_194`, `_190` | Good — separate creator bottom nav |
| J25 | Delivery with platform courier | `_106`, `_44`, `_12` | Contradicts A3 (D-29) |

## 7. Full screen tables (v2)

Columns: **ID**, **Folder**, **Domain (v2)**, **Feature (v2)**, **Actor**, **Rendered theme** (measured), **Status v2**, **Verification**, **Dup. of**, **Flags** (L = stale top-bar label, C = in-app card capture, G = gated content, S = unusable screenshot), **Notes**. v1 columns, hashes, Persian labels and parsed input/button counts are in the CSV.

### 7.1 Mobile app (277)

| ID | Folder | Domain | Feature (v2) | Actor | Theme | Status v2 | Verification | Dup. of | Flags | Notes |
|---|---|---|---|---|---|---|---|---|---|---|
| M-001 | _1 | SET | Privacy, close friends, blocked/muted users | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-002 | _2 | DISC | Category directory | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-003 | _3 | MSG | Direct messages inbox | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-004 | _4 | WAL | Wallet overview | User | dark | CONSISTENT-HTML | HTML |  |  | Wallet scope unapproved (D-03) |
| M-005 | _5 | RET | Buyer refund & return status timeline | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-006 | _6 | CRT | Creator studio | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-007 | _7 | STL | Withdrawal request to bank account (Paya cycle) | Seller | dark | VERIFIED | VISUAL+HTML |  |  |  |
| M-008 | _8 | STY | Story viewer: poll sticker + shoppable product + reply | - | dark | VERIFIED | VISUAL+HTML |  |  | Shoppable story; Needs screenshot review |
| M-009 | _9 | CAT | Storefront detail: shipping methods & rates | Buyer | light | NEAR-DUPLICATE | HTML | storefront_shipping_coverage_rates_dark |  |  |
| M-010 | _10 | ADS | Reels promote campaign setup | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-011 | _11 | CRT | In-app camera: story/reel/live-shop/post modes, product tag with commission | - | dark | VERIFIED | VISUAL+HTML |  |  | Needs screenshot review |
| M-012 | _12 | NOTIF | Activity centre: follow requests, live alerts, order status, comments | - | dark | VERIFIED | VISUAL+HTML |  |  | Resolves G-SOC-02 (notification inbox exists); Needs screenshot review |
| M-013 | _13 | AI | AI shopping assistant | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-014 | _14 | LIVE | Live host broadcast view: pinned product, flash offer, guest invite, live sales total | - | dark | VERIFIED | VISUAL+HTML |  |  | Needs screenshot review |
| M-015 | _15 | SEL | Seller pack & ship: carrier choice (Tipax/Post/VIRA express), waybill scan, label print, packing checklist, SLA timer | Buyer | dark | VERIFIED | VISUAL+HTML |  | L | Was ORD; Top-bar label 'Product Details' contradicts invoice content |
| M-016 | _16 | CHK | Checkout: payment failed | Buyer | dark | VERIFIED | VISUAL+HTML |  |  |  |
| M-017 | _17 | SOC | Edit profile | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-018 | _18 | AFF | Creator earnings studio: withdrawable commission, reel sales analytics | - | dark | VERIFIED | VISUAL+HTML |  |  | Creator payouts; Needs screenshot review |
| M-019 | _19 | MSG | Group & channel management: invite link, member permissions | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-020 | _20 | ORD | Order tracking | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-021 | _21 | SOC | Edit profile: interest categories | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-022 | _22 | CAT | Seller storefront (product grid) | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-023 | _23 | STY | Story editor (creator) | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-024 | _24 | AI | AI visual search assistant | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-025 | _25 | ORD | Multi-vendor order tracking | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-026 | _26 | DISC | Explore grid: search (visual/voice), trend chips, hashtags | - | dark | VERIFIED | VISUAL+HTML |  |  | Needs screenshot review |
| M-027 | _27 | REV | Write review, rating and video review | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-028 | _28 | CART | Multi-seller cart with itemised bill | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-029 | _29 | ADS | Promote campaign: objective selection | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-030 | _30 | CALL | Voice/video call with in-call purchase | User | mixed | CONSISTENT-HTML | HTML |  |  | In-call commerce is conceptual |
| M-031 | _31 | SEL | Variant, size & stock editor; low-stock automation | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-032 | _32 | STY | Story archive & highlights | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-033 | _33 | SOC | Home feed: For you / Following / Live shop tabs, stories tray, shoppable posts | - | dark | VERIFIED | VISUAL+HTML |  |  | Needs screenshot review |
| M-034 | _34 | SEL | Add product (media, basics, pricing, stock) | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-035 | _35 | ORD | Order detail: delivered, parcel items, final invoice | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-036 | _36 | STY | Story archive & highlights | Creator | dark | NEAR-DUPLICATE | HTML | _32 |  |  |
| M-037 | _37 | SUP | Help centre: topic search, FAQ, my tickets, new-ticket form | User | dark | VERIFIED | VISUAL+HTML |  | L | Label "account settings" wrong; Top-bar label 'Account settings' vs support content |
| M-038 | _38 | STL | Withdrawal request | Seller | dark | NEAR-DUPLICATE | HTML | _7 |  |  |
| M-039 | _39 | SEL | Product editor (dark) | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-040 | _40 | KYC | Identity verification & liveness | User | light | CONSISTENT-HTML | HTML |  |  | Biometric data |
| M-041 | _41 | PAY | VIRA-branded in-app "payment gateway" (sandbox): card no., CVV2, expiry, dynamic PIN, captcha | Buyer | dark | VERIFIED | VISUAL+HTML |  | C | CRITICAL: in-app card capture — REJECT design (V-01); Real PSP page is external; must not be replicated in-app |
| M-042 | _42 | SOC | Edit profile | User | no-screenshot | DUPLICATE | HTML (no usable screenshot) | _17 | S | Byte-identical to _17; only screen without screenshot |
| M-043 | _43 | SEL | Variant & inventory editor (colour, size) | Seller | dark | NEAR-DUPLICATE | HTML | _31 |  |  |
| M-044 | _44 | ORD | My orders list with live courier distance/ETA and rider call | Buyer | dark | VERIFIED | HTML |  |  | Was CHK; label "Checkout Order" wrong |
| M-045 | _45 | ADS | Promote campaign: objective & audience | Seller | dark | NEAR-DUPLICATE | HTML | _29 |  |  |
| M-046 | _46 | SET | Privacy: account, stories, mentions, tags, products | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-047 | _47 | MSG | Create broadcast channel: name, handle, description, public/VIP access, auto-publish toggles | User | dark | VERIFIED | VISUAL+HTML |  | L | Top-bar label 'Checkout Flow' vs channel creation content |
| M-048 | _48 | CRT | Creator studio | Creator | dark | NEAR-DUPLICATE | HTML | _6 |  |  |
| M-049 | _49 | CAT | Product detail | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-050 | _50 | RLS | Shoppable reel player with buy button | - | dark | VERIFIED | VISUAL+HTML |  |  | Needs screenshot review |
| M-051 | _51 | CART | Shopping cart | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-052 | _52 | CRT | Publish success (post/reel) | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-053 | _53 | SET | Logout confirmation + deactivate/delete entry; shows wallet balance & active orders | User | dark | VERIFIED | HTML (no usable screenshot) |  | LS | Screenshot truncated (only top third rendered); Top-bar label 'Product Detail' vs account content |
| M-054 | _54 | DISC | Filter & sort | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-055 | _55 | WAL | Wallet (DEMO balance 1bn Toman): top-up, transfer, instant settle, cashback, lucky wheel | - | dark | VERIFIED | VISUAL+HTML |  |  | Stored value + gamification — D-03, D-30; Needs screenshot review |
| M-056 | _56 | ADS | 'Why am I seeing this ad' + buyer protection | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Ad transparency |
| M-057 | _57 | SEL | Seller dashboard (پیشخان) | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-058 | _58 | MSG | 1:1 chat with product context | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-059 | _59 | SET | Security centre: SMS 2FA, authenticator, biometric login, change password, last security event (IP, city), sessions | User | dark | VERIFIED | VISUAL+HTML |  | L | Password + biometric login — D-22; Top-bar 'Product Detail'; also implies passwords exist (OTP-only?) |
| M-060 | _60 | DISC | Category product listing | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-061 | _61 | SET | Appearance / theme settings | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-062 | _62 | AI | AI shopping assistant | Buyer | dark | NEAR-DUPLICATE | HTML | _13 |  |  |
| M-063 | _63 | SEL | Seller order processing | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-064 | _64 | SET | Account settings | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-065 | _65 | CALL | Voice/video call | User | light | NEAR-DUPLICATE | HTML | _30 |  |  |
| M-066 | _66 | KYC | KYC L1: register bank card number (+ optional Sheba) for ownership match | User | light | VERIFIED | VISUAL+HTML | _40 | C | Full PAN entered in-app — tokenise/mask (V-01); Biometric data |
| M-067 | _67 | SEL | Seller management (mobile) | Seller | dark | CONSISTENT-HTML | HTML |  |  | Label only; content unknown |
| M-068 | _68 | MSG | Channel directory: categories, join, create-store-channel CTA | User | dark | VERIFIED | VISUAL+HTML |  | L | Top-bar 'Store Profile' vs channel directory |
| M-069 | _69 | CHK | Checkout: recipient & 2 parcels from 2 stores | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-070 | _70 | DISC | Nearby stores (Saadat Abad, west Tehran) | Buyer | light | NEAR-DUPLICATE | HTML | nearby_stores_location_sellers_dark |  | Location data |
| M-071 | _71 | REV | Video review & rating | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-072 | _72 | CRT | Studio tools | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-073 | _73 | KYC | Level-1 verification success | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-074 | _74 | CHK | Address selector & instant-delivery stores | Buyer | light | NEAR-DUPLICATE | HTML | buyer_delivery_address_location_selector_dark |  | Location data |
| M-075 | _75 | DISC | Category landing / seasonal campaign | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-076 | _76 | CHK | Test payment success | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Demo-only state ('پرداخت تستی') |
| M-077 | _77 | SET | Account security: 2FA, active sessions, event log | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-078 | _78 | SEL | Packing / pick list | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-079 | _79 | CRT | Reel publish success + upsell to promote (dark) | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-080 | _80 | CHK | Checkout payment steps | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-081 | _81 | REV | Video review & rating | Buyer | dark | NEAR-DUPLICATE | HTML | _71 |  |  |
| M-082 | _82 | ADS | 'Why am I seeing this ad' | Buyer | dark | NEAR-DUPLICATE | HTML | _56 |  |  |
| M-083 | _83 | KYC | Level-2 submitted (during checkout): case code, inventory + 35% code held ~3 h, anti-phishing note (VIRA never asks CVV2) | Buyer | dark | VERIFIED | VISUAL+HTML |  | L | Was RET; reservation during KYC — D-35; Top-bar 'Checkout Flow'; could also be KYC evidence |
| M-084 | _84 | MSG | Channel feed (store discounts channel) | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-085 | _85 | SEL | Store order list (dark) | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-086 | _86 | CAT | Storefront | Buyer | dark | NEAR-DUPLICATE | HTML | _22 |  |  |
| M-087 | _87 | LIVE | Live commerce session (viewer) | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-088 | _88 | CART | Cart with pro-forma invoice | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-089 | _89 | KYC | Inline Level-2 national-ID scan inside checkout (OCR) | Buyer | dark | VERIFIED | HTML |  |  | Was CHK |
| M-090 | _90 | RET | Seller return decision: refund to WALLET / exchange / reject→arbitration, evidence photos | Seller | dark | VERIFIED | VISUAL+HTML |  | L | Refund-to-wallet conflicts with D-03/D-08; Top-bar 'Product Detail' |
| M-091 | _91 | CRT | Media picker | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-092 | _92 | CHK | Test payment failed + sandbox troubleshooting | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Instructs dynamic-PIN entry within the VIRA flow (V-01); offers wallet payment without dynamic PIN (V-04); Demo-only state |
| M-093 | _93 | CRT | Publish content | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-094 | _94 | MSG | Group management | User | light | NEAR-DUPLICATE | HTML | _19 |  |  |
| M-095 | _95 | ADS | Promote campaign | Seller | dark | NEAR-DUPLICATE | HTML | _29 |  |  |
| M-096 | _96 | KYC | Upgrade to level 2 | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-097 | _97 | CRT | Publish content | Creator | dark | NEAR-DUPLICATE | HTML | _93 |  |  |
| M-098 | _98 | SUP | Support ticket with document upload | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-099 | _99 | CAT | Product detail with buyer video reviews | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-100 | _100 | CHK | Demo payment receipt (dark) | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Demo-only state |
| M-101 | _101 | LIVE | Live viewer: chat, flash sale, buy now, virtual gifts ("diamonds") | - | dark | VERIFIED | VISUAL+HTML |  |  | Virtual gifting — D-31; Needs screenshot review |
| M-102 | _102 | CRT | Studio tools | Creator | dark | NEAR-DUPLICATE | HTML | _72 |  |  |
| M-103 | _103 | LIVE | Live streaming desk (host) | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-104 | _104 | CAT | Product detail | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-105 | _105 | CART | Cart: addresses, shipping from 3 stores, shipping method | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-106 | _106 | ORD | Live courier tracking: VIRA rider ("سفیر") on map, speed, ETA, call/chat rider, delivery code | Buyer | dark | VERIFIED | VISUAL+HTML |  | L | Platform courier fleet — D-29; Top-bar 'Shopping Cart' vs shipment content |
| M-107 | _107 | AFF | Creator affiliate dashboard (dark) | Creator | dark | CONSISTENT-HTML | HTML |  |  | Commission payouts |
| M-108 | _108 | KYC | AI document/face analysis | User | light | CONSISTENT-HTML | HTML |  |  | Automated decision-making on biometrics |
| M-109 | _109 | SOC | Followers / following | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-110 | _110 | MSG | DM inbox: notes, requests, seller pro-forma invoice with "direct payment" in chat, voice msg | - | dark | VERIFIED | VISUAL+HTML |  |  | Chat invoice payment — new payment initiation path; Needs screenshot review |
| M-111 | _111 | WAL | Wallet overview | User | dark | NEAR-DUPLICATE | HTML | _4 |  |  |
| M-112 | _112 | SUP | Official ticketing & dispute (dark) | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-113 | _113 | ADS | Campaign KPIs & funnel (demo) | Seller | dark | CONSISTENT-HTML | HTML |  |  | Contains demo label |
| M-114 | _114 | STY | Story archive & highlight creation | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-115 | _115 | RET | Refund to source account | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-116 | _116 | AI | Visual search (image/text) | Buyer | dark | NEAR-DUPLICATE | HTML | _24 |  |  |
| M-117 | _117 | STY | Active highlights / create highlight | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-118 | _118 | REV | Video review & rating | Buyer | dark | NEAR-DUPLICATE | HTML | _71 |  |  |
| M-119 | _119 | ADS | Promote campaign objective | Seller | dark | NEAR-DUPLICATE | HTML | _29 |  |  |
| M-120 | _120 | RET | Seller return decision: refund instantly to customer WALLET / resend / arbitration | Seller | light | VERIFIED | VISUAL+HTML | _90 | L | Refund-to-wallet; duplicate of _90 pattern; Top-bar 'Product Detail' |
| M-121 | _121 | SET | Security: SMS OTP, authenticator, active sessions (3), sign out others | User | light | VERIFIED | VISUAL+HTML |  | L | Top-bar 'Product Detail' |
| M-122 | _122 | RLS | Share reel sheet: DM recipients, story, channel, copy link, external apps | User | light | VERIFIED | VISUAL+HTML |  | L | Top-bar 'Product Detail' |
| M-123 | _123 | CAT | Product detail | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-124 | _124 | RET | Return tracking: awaiting courier pickup | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-125 | _125 | SET | Privacy: private account, story/live, product tags | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-126 | _126 | SEL | Variant & inventory matrix | Seller | light | NEAR-DUPLICATE | HTML | _31 |  |  |
| M-127 | _127 | SUP | Create support ticket | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-128 | _128 | SEL | Order fulfilment | Seller | light | CONSISTENT-HTML | HTML |  |  |  |
| M-129 | _129 | PAY | Payment failed: alternatives — saved Shetab cards, VIRA wallet, VIRA credit/4 instalments | Buyer | light | VERIFIED | VISUAL+HTML |  |  | Card-on-file (D-34), BNPL (D-27) |
| M-130 | _130 | CRT | Media crop & aspect ratio | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-131 | _131 | CHK | Order review step | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-132 | _132 | CHK | Payment success & e-receipt | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-133 | _133 | AFF | Creator profile: affiliate earnings | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-134 | _134 | CRT | Reel published + smart promote | Creator | light | NEAR-DUPLICATE | HTML | _79 |  |  |
| M-135 | _135 | CAT | Product detail | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-136 | _136 | LIVE | Host desk: showcase tools & live chat | Seller | light | CONSISTENT-HTML | HTML |  |  |  |
| M-137 | _137 | CRT | Multi-media gallery picker | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-138 | _138 | ORD | My orders list & delivery tracking | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-139 | _139 | SEL | Catalog & inventory list | Seller | light | CONSISTENT-HTML | HTML |  |  |  |
| M-140 | _140 | SEL | Add product step 2/3: media, basics, category, SKU, brand, variants, bulk price | Seller | light | VERIFIED | VISUAL+HTML | _34 | L | Top-bar 'Instant Checkout' |
| M-141 | _141 | REV | Product reviews & Q&A: filters, seller pinned reply, verified-buyer badges | - | light | VERIFIED | VISUAL+HTML |  |  | Top-bar 'Instant Checkout'; only names visible |
| M-142 | _142 | PAY | IMITATION of a bank IPG page inside app: card no., CVV2, expiry, dynamic PIN, "save card in VIRA" | - | light | VERIFIED | VISUAL+HTML |  | C | CRITICAL: in-app card capture, "save card" pre-ticked — REJECT design (V-01, V-18); Top-bar 'Instant Checkout' |
| M-143 | _143 | SOC | Saved items & collections | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-144 | _144 | CAT | Product details | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-145 | _145 | CAT | Product details | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-146 | _146 | SET | Business profile editor: name, handle, title, bio, links, category, contact, email, gender | User | light | VERIFIED | VISUAL+HTML |  | L | Gender field — minimise (privacy); Top-bar 'Product Details' |
| M-147 | _147 | CAT | Product linked to video (product tagging) | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-148 | _148 | ORD | Order detail, invoice & return entry | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-149 | _149 | CAT | Seller storefront | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-150 | _150 | MSG | Groups, message requests, live chat room | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-151 | _151 | STY | Story editor & stickers (poll) | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-152 | _152 | PROMO | Gift cards: buy, redeem voucher, send gift card to friend, coupon history | Buyer | dark | VERIFIED | VISUAL+HTML |  | L | Gift cards = stored value — D-03/D-30; Top-bar 'Settings & Security' |
| M-153 | _153 | CHK | Demo transfer success | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Demo-only state |
| M-154 | _154 | ORD | Shipment route & contents | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-155 | _155 | SET | Notification preferences | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-156 | _156 | CRY | USDT → Toman conversion | User | dark | GATED | HTML |  | G | Crypto: disabled pending approval |
| M-157 | _157 | CRY | Toman → USDT conversion | User | dark | GATED | HTML |  | G | Crypto: disabled pending approval |
| M-158 | _158 | CRY | Dual-currency wallet | User | dark | GATED | HTML |  | G | Crypto: disabled pending approval |
| M-159 | _159 | CRY | Conversion history & receipts | User | dark | GATED | HTML |  | G | Crypto |
| M-160 | _160 | WAL | Wallet overview & recent transactions | User | light | NEAR-DUPLICATE | HTML | _4 | G |  |
| M-161 | _161 | CRY | Conversion receipt & history | User | light | GATED | HTML | _159 | G | Crypto |
| M-162 | _162 | CRY | Instant sell USDT to Toman wallet | User | light | GATED | HTML | _156 | G | Crypto |
| M-163 | _163 | CRY | Toman → USDT | User | light | GATED | HTML | _157 | G | In-app Toman→USDT exchange (V-19); crypto terms pre-ticked (V-18); Crypto |
| M-164 | _164 | PAY | Payment method selection — USDT pre-selected with discount; card IPG; wallet | Buyer | dark | VERIFIED | VISUAL+HTML |  | G | GATED option promoted as default (V-02); May include USDT option (gated) |
| M-165 | _165 | CRY | USDT payment details | Buyer | dark | GATED | HTML |  | G | Crypto |
| M-166 | _166 | CRY | Payment verification: transfer details | Buyer | dark | GATED | HTML |  | G | Crypto (likely TX hash entry) |
| M-167 | _167 | CRY | USDT payment confirmed + 'Escrow Lock' | Buyer | dark | GATED | HTML |  | G | Crypto; escrow terminology |
| M-168 | _168 | PAY | Checkout payment-method step (Rial or USDT up to 250M/day at L2) | Seller | light | VERIFIED | HTML |  | G | Was STL; contains GATED option |
| M-169 | _169 | CRY | USDT checkout: network choice TRC20/Polygon/ERC20, QR + address, locked rate, "escrow", sandbox | - | light | GATED | VISUAL+HTML |  | G | GATED; Needs screenshot review |
| M-170 | _170 | CRY | USDT payment confirmation tracker: block confirmations, TXID, under/over-payment rules | - | light | GATED | VISUAL+HTML |  | G | GATED; Needs screenshot review |
| M-171 | _171 | CRY | USDT payment confirmed | Buyer | light | GATED | HTML | _167 | G | Crypto |
| M-172 | _172 | STL | Withdrawal method: USDT recommended ("instant"), Toman to Sheba | Seller | dark | VERIFIED | VISUAL+HTML |  | G | GATED option promoted; May include USDT (gated) |
| M-173 | _173 | CRY | USDT payout breakdown | Seller | dark | GATED | HTML |  | G | Crypto |
| M-174 | _174 | STL | Withdrawal confirmation | Seller | dark | CONSISTENT-HTML | HTML |  | G |  |
| M-175 | _175 | CRY | USDT payout broadcasting on TRON (TRC20) | Seller | dark | GATED | HTML |  | G | Crypto |
| M-176 | _176 | STL | Funds withdrawal | Seller | light | CONSISTENT-HTML | HTML |  | G |  |
| M-177 | _177 | STL | Settlement method & security terms | Seller | light | CONSISTENT-HTML | HTML |  | G |  |
| M-178 | _178 | STL | Withdrawal confirmation: USDT address, TOTP 6-digit + "financial PIN", ownership attestation | Seller | light | VERIFIED | VISUAL+HTML |  | G | Financial PIN concept — D-22 |
| M-179 | _179 | CRY | TRC20 transfer (treasury cycle) | Seller | light | GATED | HTML | _175 | G | Top-bar 'Checkout Order' |
| M-180 | _180 | KYC | National ID card capture | User | dark | VERIFIED | VISUAL+HTML |  |  | National ID data |
| M-181 | _181 | KYC | Video liveness | User | dark | CONSISTENT-HTML | HTML |  |  | Biometric data |
| M-182 | _182 | KYC | Identity info & level comparison matrix | User | dark | CONSISTENT-HTML | HTML |  | G |  |
| M-183 | _183 | KYC | Final confirmation, digital signature, treasury review | Seller | dark | VERIFIED | VISUAL+HTML |  | G | Digital signature legal validity (D-15) |
| M-184 | _184 | KYC | National identity entry & level status | User | light | CONSISTENT-HTML | HTML |  | G |  |
| M-185 | _185 | STL | Sheba (IBAN) binding + national card | Seller | light | VERIFIED | VISUAL+HTML |  |  | Bank account data |
| M-186 | _186 | KYC | Biometric liveness check | User | light | NEAR-DUPLICATE | HTML | _181 |  |  |
| M-187 | _187 | STL | Treasury verification status | Seller | light | CONSISTENT-HTML | HTML |  | G |  |
| M-188 | _188 | AFF | Creator affiliate overview with own bottom nav (dashboard/products/campaigns/earnings) | - | dark | VERIFIED | VISUAL+HTML |  |  | Separate creator workspace; Needs screenshot review |
| M-189 | _189 | AFF | Affiliate catalog: commission %, links/tags, creator sample orders | - | dark | VERIFIED | VISUAL+HTML |  |  | Needs screenshot review |
| M-190 | _190 | AFF | Commission settlement & withdrawal | Creator | dark | CONSISTENT-HTML | HTML |  | G |  |
| M-191 | _191 | AFF | Affiliate product link generator | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-192 | _192 | AFF | Campaign collaboration | Creator | light | NEAR-DUPLICATE | HTML | _191 |  |  |
| M-193 | _193 | AFF | Affiliate marketplace for creators | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-194 | _194 | AFF | Creator earnings & commission approval cycle | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-195 | _195 | AFF | Earnings withdrawal | Creator | light | NEAR-DUPLICATE | HTML | _190 | G |  |
| M-196 | _196 | ORD | Multi-store order detail & parcel split (dark) | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-197 | _197 | RET | Partial return request (dark) | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-198 | _198 | ORD | Parcel tracking & delivery confirmation; statutory withdrawal window (dark) | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-199 | _199 | RET | Refund status & seller settlement lock (dark) | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-200 | _200 | ORD | Parcel tracking: delivered (dark) | Buyer | dark | NEAR-DUPLICATE | HTML | _198 |  | State variant |
| M-201 | _201 | RET | Partial return request (dark) | Buyer | dark | NEAR-DUPLICATE | HTML | _197 |  |  |
| M-202 | _202 | ORD | Multi-store order detail (dark) | Buyer | dark | NEAR-DUPLICATE | HTML | _196 |  |  |
| M-203 | _203 | RET | Partial return case: financial split (dark) | Buyer | dark | NEAR-DUPLICATE | HTML | _199 |  |  |
| M-204 | _204 | SET | Security & privacy status | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-205 | _205 | SET | 2FA & active sessions | User | light | CONSISTENT-HTML | HTML | _77 |  |  |
| M-206 | _206 | SET | Logout, deactivate, delete account | User | light | CONSISTENT-HTML | HTML |  |  | Data erasure policy (D-12) |
| M-207 | _207 | SET | Privacy: stories, reels, close friends, DMs | User | light | CONSISTENT-HTML | HTML | _1 |  |  |
| M-208 | _208 | MOD | Moderation appeal status | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-209 | _209 | SUP | Support hub | User | dark | CONSISTENT-HTML | HTML |  | G |  |
| M-210 | _210 | MOD | Report violation (dark) | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-211 | _211 | SUP | Ticket timeline (USDT deposit case) | User | dark | CONSISTENT-HTML | HTML |  | G | Crypto-related content |
| M-212 | _212 | MOD | Appeal a restriction | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-213 | _213 | MOD | Report violation | User | light | NEAR-DUPLICATE | HTML | _210 |  |  |
| M-214 | _214 | SUP | Knowledge base | User | light | NEAR-DUPLICATE | HTML | _209 | G |  |
| M-215 | _215 | SUP | Ticket tracker | User | light | NEAR-DUPLICATE | HTML | _211 | G |  |
| M-216 | _216 | SYS | Offline mode: cached content & pending action queue | User | dark | CONSISTENT-HTML | HTML |  |  | Offline write queue |
| M-217 | _217 | SYS | Offline: local drafts | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-218 | _218 | SYS | Upload interrupted: local data protection | Creator | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-219 | _219 | PAY | Payment failed: do NOT pay again; transaction inquiry | Buyer | dark | VERIFIED | VISUAL+HTML |  |  | Payment-unknown state |
| M-220 | _220 | SYS | Network error / offline mode | User | light | NEAR-DUPLICATE | HTML | _216 |  |  |
| M-221 | _221 | SYS | Upload interrupted (504): resume at 78%, local draft protection, save to drafts | Creator | light | VERIFIED | VISUAL+HTML |  | L | Was LIVE; label wrong; Label 'Live Stream Fallback' vs reel-draft content |
| M-222 | _222 | SYS | Offline drafts | Creator | light | NEAR-DUPLICATE | HTML | _217 |  |  |
| M-223 | _223 | PAY | Transaction recovery: stock reserved, do not pay again | Buyer | light | VERIFIED | VISUAL+HTML |  |  |  |
| M-224 | _224 | SEL | Seller mode & permission level (Gold, level 3) | Seller | dark | VERIFIED | VISUAL+HTML |  | G |  |
| M-225 | _225 | SEL | Seller alert & notification centre | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-226 | _226 | SEL | Order command centre | Seller | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-227 | _227 | SEL | Store staff & access permissions | Seller | dark | VERIFIED | VISUAL+HTML |  | G | Seller sub-accounts |
| M-228 | _228 | SEL | Seller mode hub: switch buyer/seller, seller tier (gold, 2% fee), permission matrix incl. USDT settlement, links to "Seller Web Admin" | Seller | light | VERIFIED | VISUAL+HTML |  | LG | Evidence for a Seller web console (D-23); Label vs permissions content |
| M-229 | _229 | SEL | Order detail with warehouse SLA alert | Seller | light | CONSISTENT-HTML | HTML |  |  |  |
| M-230 | _230 | SEL | Seller alert inbox: SLA alerts, new paid orders, return evidence, chats, payout receipt | Seller | light | VERIFIED | VISUAL+HTML |  | L | Label 'Payout Settings' vs alert inbox content |
| M-231 | _231 | SEL | Store staff & security: owner/legal rep, staff roles, suspend, invite; plan cap "silver package: 5 users" | Seller | light | VERIFIED | VISUAL+HTML |  | LG | Seller subscription plans — D-32; Label 'Live Session Setup' vs store legal content |
| M-232 | account_settings_light | SET | Settings & appearance | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-233 | active_video_call_light | CALL | Active video call | User | mixed | CONSISTENT-HTML | HTML |  |  |  |
| M-234 | bank_card_ownership_verification_light | KYC | Card ownership & tax/Shaparak identity match: full card number entry | Seller | light | VERIFIED | VISUAL+HTML |  | C | Full PAN entered in-app (V-01); KYC L1 card-ownership step, same function as _66; Top-bar 'Mobile Verification' |
| M-235 | blue_badge | IDN | Blue badge (verified account) application (dark) | User | dark | CONSISTENT-HTML | HTML |  |  | Theme not in name |
| M-236 | buyer_delivery_address_location_selector_dark | CHK | Delivery address & location selector | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Location data |
| M-237 | buyer_level_1_approved_light | KYC | Buyer level-1 approved; >10M purchases need level 2 | Buyer | light | CONSISTENT-HTML | HTML |  |  | Purchase limits by KYC level |
| M-238 | call_history_light_1 | CALL | Call & consultation history | User | dark | CONSISTENT-HTML | HTML (no usable screenshot) |  | S |  |
| M-239 | call_history_light_2 | CALL | Call history | User | light | NEAR-DUPLICATE | HTML | call_history_light_1 |  |  |
| M-240 | channel_feed_light | MSG | Channel feed | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-241 | checkout_light | CHK | Checkout step 2 (address/shipping) with KYC-L1 limit notice; "connect to DEMO gateway" | Buyer | dark | VERIFIED | HTML (no usable screenshot) |  | LS | Screenshot BLANK; resolved from HTML; folder name was right; Folder 'checkout' but content is order tracking |
| M-242 | comments | SOC | Comments sheet with seller pinned comment + product card, reactions | User | dark | VERIFIED | VISUAL+HTML |  | L | Top-bar label wrong; Top-bar 'Shopping Cart' |
| M-243 | create_post_light | CRT | Create post | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-244 | create_reel_light | CRT | Create reel | Creator | light | CONSISTENT-HTML | HTML |  |  |  |
| M-245 | direct_messages_inbox_light_1 | MSG | DM inbox | User | dark | CONSISTENT-HTML | HTML (no usable screenshot) |  | S |  |
| M-246 | direct_messages_inbox_light_2 | MSG | DM inbox | User | light | NEAR-DUPLICATE | HTML | direct_messages_inbox_light_1 |  |  |
| M-247 | explore_light | DISC | Explore | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-248 | group_chat_light | MSG | Group chat | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-249 | help_support_light | SUP | Help & support | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-250 | home_feed_light | SOC | Home feed | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-251 | incoming_video_call_light | CALL | Incoming video call (seller consultation, auto-replies) | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-252 | incoming_voice_call_light | CALL | Incoming voice call | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-253 | level_2_identity_verification_introduction_light | KYC | Level-2 intro: limits, required docs, biometric consent checkbox; promises credit purchase, instalments, live auctions | User | light | VERIFIED | VISUAL+HTML |  | L | Rendered LIGHT (heading text mentions dark); credit/auction concepts — D-27/D-33; Folder says light; heading says dark (دارک) |
| M-254 | level_2_national_id_capture_light | KYC | National ID card scan | User | light | CONSISTENT-HTML | HTML |  |  | National ID data |
| M-255 | level_2_verification_pending_light | KYC | Level-2 pending: 4-step status, cart reserved 24 h | User | light | VERIFIED | VISUAL+HTML |  | L | Rendered LIGHT; 24 h reservation conflicts with reservation design — D-35; Folder says light; heading says dark (دارک) |
| M-256 | level_2_video_liveness_verification_light | KYC | Video liveness | User | light | CONSISTENT-HTML | HTML |  |  | Biometric data |
| M-257 | live_shopping_light | LIVE | Live commerce session | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-258 | mobile_otp_verification_light | IDN | Mobile OTP verification | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-259 | multi_vendor_cart_split_shipping_dark | CART | Multi-vendor cart, split shipping by store | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-260 | my_orders_light | ORD | My orders | Buyer | dark | CONSISTENT-HTML | HTML (no usable screenshot) |  | S |  |
| M-261 | nearby_stores_location_sellers_dark | DISC | Nearby stores | Buyer | dark | CONSISTENT-HTML | HTML |  |  | Location data |
| M-262 | notifications_light | NOTIF | Notifications | User | light | CONSISTENT-HTML | HTML |  |  | Top-bar 'Settings' (inbox vs preferences unclear) |
| M-263 | otp | IDN | Welcome: phone + 4-digit OTP + account type (social user / official seller) + implicit terms acceptance | User | dark | VERIFIED | VISUAL+HTML |  | L | OTP length 4 vs 6; no age gate; implicit consent; Top-bar 'Shopping Cart' |
| M-264 | private_chat_light | MSG | Private chat | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-265 | product_details_light | CAT | Product detail | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-266 | product_management_light | SEL | Product management | Seller | light | CONSISTENT-HTML | HTML |  |  | Top-bar 'Creator Studio' |
| M-267 | reels_light | RLS | Reels player | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-268 | seller_dashboard_light | SEL | Seller dashboard | Seller | light | CONSISTENT-HTML | HTML |  |  |  |
| M-269 | seller_storefront_light | CAT | Seller storefront | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-270 | shop_hub_light | DISC | Shop hub | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-271 | shopping_cart_light_1 | CART | Multi-seller cart | Buyer | light | CONSISTENT-HTML | HTML |  |  |  |
| M-272 | shopping_cart_light_2 | CART | Cart step 1 with coupon field | Buyer | dark | VERIFIED | HTML (no usable screenshot) |  | LS | Screenshot BLANK; resolved from HTML; Folder 'shopping_cart' but top-bar 'Order Tracking' |
| M-273 | storefront_shipping_coverage_rates_dark | CAT | Store shipping coverage & rates | Buyer | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-274 | stories_viewer_light | STY | Stories viewer | User | dark | CONSISTENT-HTML | HTML |  |  |  |
| M-275 | tiktok_shop_hub | DISC | Shop hub variant | Buyer | dark | NEAR-DUPLICATE | HTML | shop_hub_light |  | Third-party brand in name; same headings as shop_hub_light |
| M-276 | user_profile_light | SOC | User / store profile | User | light | CONSISTENT-HTML | HTML |  |  |  |
| M-277 | wallet_light | WAL | Wallet & transactions; demo top-up | User | light | CONSISTENT-HTML | HTML |  |  | Demo top-up; wallet scope unapproved |

### 7.2 Super Admin (53 unique; each exists twice, byte-identical)

| ID | Folder | Domain | Feature (v2) | Actor | Theme | Status v2 | Verification | Dup. of | Flags | Notes |
|---|---|---|---|---|---|---|---|---|---|---|
| A-001 | a_b_vira_experiments_a_b_testing_dark | GROW | A/B experiments registry | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-002 | psp_vira_payment_gateways_psp_dark | PAY | PSP / payment gateway management | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-003 | vira_administrators_rbac_dark | ADM | Administrators & granular RBAC matrix, admin sessions, invites | Admin | dark | VERIFIED | VISUAL+HTML |  |  | Pair is byte-identical in both admin trees |
| A-004 | vira_advanced_analytics_bi_dark | ANL | BI: GMV, funnel, geographic distribution | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-005 | vira_advertising_campaigns_dark | ADS | Advertising & campaigns | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-006 | vira_ai_content_hub_dark | AI | AI content generation hub | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-007 | vira_ai_operations_usage_dark | AI | AI providers, usage & cost (incl. local Persian LLM) | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-008 | vira_ai_product_studio_dark | AI | AI product studio: bulk upload & image processing | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-009 | vira_app_banners_placements_dark | CMS | App banners & placements with live preview | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-010 | vira_app_releases_rollouts_dark | OPS | App versions, release pipelines, staged rollout | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-011 | vira_categories_management_dark | CAT | Category tree management | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-012 | vira_comments_reports_dark | MOD | Comment moderation & violation reports | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-013 | vira_custom_reports_report_builder_dark | ANL | Custom report builder & exports | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-014 | vira_data_archive_backup_dark | OPS | Data retention, cold archive, backup jobs | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-015 | vira_discounts_promotions_dark | PROMO | Discounts, coupons, promotions + fraud shield | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-016 | vira_dispute_arbitration_center_dark | SUP | Financial dispute arbitration ("escrow protocol") | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-017 | vira_error_tracking_diagnostics_dark | OPS | Error tracking & crash telemetry | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-018 | vira_escrow_automated_commission_dark | LEDG | Settlement hold ("escrow") & automated commission split | Admin | dark | VERIFIED | VISUAL+HTML |  | G | Pair is byte-identical in both admin trees |
| A-019 | vira_feature_flags_remote_config_dark | OPS | Feature flags & remote config | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-020 | vira_financial_fraud_investigation_dark | RISK | Financial fraud investigation | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-021 | vira_financial_reconciliation_settlements_dark | LEDG | Reconciliation & Paya/Satna/USDT batch settlement | Admin | dark | VERIFIED | VISUAL+HTML |  | G | Pair is byte-identical in both admin trees; contains USDT controls (gated) |
| A-022 | vira_geographic_commerce_discovery_dark | DISC | Geographic commerce & regional discovery tuning | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-023 | vira_kyc_financial_compliance_dark | KYC | KYC & AML console, review queue | Admin | dark | VERIFIED | VISUAL+HTML |  | G | Pair is byte-identical in both admin trees |
| A-024 | vira_live_shopping_management_dark | LIVE | Live shopping monitoring & emergency stop | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-025 | vira_marketplace_growth_loyalty_dark | GROW | Growth & loyalty tiers | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-026 | vira_notifications_messaging_center_dark | NOTIF | Notification centre: SMS lines, FCM, composer | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-027 | vira_orders_management_dark | ORD | Global orders & sub-order inspector | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-028 | vira_payments_management_dark | PAY | Payment transactions (Toman & USDT) | Admin | dark | VERIFIED | VISUAL+HTML |  | G | Search by full card number (V-20); USDT content (gated); Pair is byte-identical in both admin trees; contains USDT controls (gated) |
| A-029 | vira_platform_control_center_dark | OPS | Platform control centre: readiness, kill switches, version governance | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-030 | vira_platform_settings_dark | CFG | Platform business rules: fees, settlement regime | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees; contains USDT controls (gated) |
| A-031 | vira_products_management_dark | CAT | Global catalog moderation | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-032 | vira_recommendations_personalization_dark | DISC | Recommendation weights & touchpoints | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-033 | vira_refunds_returns_management_dark | RET | Refunds & returns finance desk (incl. USDT) | Admin | dark | VERIFIED | VISUAL+HTML |  | G | Pair is byte-identical in both admin trees; contains USDT controls (gated) |
| A-034 | vira_search_engine_search_experience_dark | DISC | Search engine: Persian spelling correction, synonyms | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-035 | vira_security_audit_logs_dark | SEC | SOC & audit log investigation | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-036 | vira_seller_performance_ranking_dark | SEL | Seller performance ranking | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-037 | vira_seller_quality_policy_enforcement_dark | MOD | Seller quality & policy enforcement cases | Admin | dark | CONSISTENT-HTML | HTML |  |  | Pair is byte-identical in both admin trees |
| A-038 | vira_sellers_management | SEL | Sellers & suppliers, verification queue | Admin | light | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-039 | vira_shipping_delivery_dark | ORD | Shipping carriers, coverage (31 provinces) | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-040 | vira_stalls_management | CAT | Stores (stalls) & catalogs | Admin | light | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-041 | vira_static_content_app_pages_dark | CMS | Static pages, terms, knowledge base | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees; contains USDT controls (gated) |
| A-042 | vira_stories_management_dark | MOD | Stories moderation console | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-043 | vira_super_admin_dashboard | ANL | Super admin dashboard (theme unspecified) | Admin | light | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-044 | vira_super_admin_dashboard_light_mode | ANL | Super admin dashboard (light) | Admin | light | NEAR-DUPLICATE | HTML | vira_super_admin_dashboard | G | Pair is byte-identical in both admin trees |
| A-045 | vira_support_dispute_resolution_dark | SUP | Customer support & financial disputes | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-046 | vira_system_operations_monitoring_dark | OPS | NOC: service health, incidents | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-047 | vira_unified_operations_command_center_dark | OPS | Unified operations command centre | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-048 | vira_usdt_networks_exchange_providers_dark | CRY | USDT networks & exchange providers | Admin | dark | GATED | HTML |  | G | Pair is byte-identical in both admin trees |
| A-049 | vira_users_management | IDN | User management, sensitive-action confirmation | Admin | light | VERIFIED | VISUAL+HTML |  | G | Pair is byte-identical in both admin trees |
| A-050 | vira_videos_posts_management_dark | MOD | Videos & posts moderation | Admin | dark | CONSISTENT-HTML | HTML |  | G | Pair is byte-identical in both admin trees |
| A-051 | vira_wallet_monitoring_risk_dark | WAL | Wallet monitoring, risk & ledger changes | Admin | dark | CONSISTENT-HTML | HTML |  | G | Draws dual-control (reason + senior + CFO) — pattern to reuse (V-06); Pair is byte-identical in both admin trees |
| A-052 | vira_wallets_conversion_dark | CRY | Wallets & USDT↔Toman conversion desk | Admin | dark | GATED | HTML |  | G | Pair is byte-identical in both admin trees |
| A-053 | vira_withdrawals_settlements_dark | STL | Seller withdrawal & settlement approvals | Admin | dark | VERIFIED | VISUAL+HTML |  | G | Pair is byte-identical in both admin trees |

## 8. Follow-up before inventory sign-off (v2)

| # | Item | Status |
|---|---|---|
| 1 | Visual review of the 17 UNVERIFIED and 25 CONTRADICTORY screens | **Done** (doc 15 §2) |
| 2 | Record actual rendered theme per screen | **Done** (`rendered_theme`) |
| 3 | Extract form fields for KYC, checkout, payout, seller onboarding | **Done** (parsed inputs; findings in doc 15) |
| 4 | Nominate one canonical design per NEAR-DUPLICATE cluster and archive the rest | Open — design owner |
| 5 | Mark each screen **In MVP / Later / Rejected** with the product owner (add a `decision` column). Already recommended **Rejected**: `_41`, `_142` (in-app card capture), all CRY screens for MVP, lucky-wheel element of `_55` | Open — product owner (part of D-02) |
| 6 | Re-export the 5 blank, 1 missing and 1 truncated screenshots if those designs are kept | Open — design owner |
| 7 | Replace stale top-bar labels (25 screens) and unify navigation (doc 09 §4) | Open — design owner |
