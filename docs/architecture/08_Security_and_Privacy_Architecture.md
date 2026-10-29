# 08 — Security and Privacy Architecture

**Status:** Phase 0 v3 (final for owner acceptance; v2/v3 additions marked). Baselines: OWASP ASVS 4.0 Level 2 (API & web), OWASP MASVS (L1 + resilience controls for payment flows), OWASP API Security Top 10. Regulatory items are questions for counsel, not legal conclusions.

## 1. Assets and classification

| Class | Examples | Storage rule | Access rule |
|---|---|---|---|
| **C4 — Restricted** | National ID number, ID card images, liveness video/selfie, Sheba/IBAN, bank card owner data, KYC decisions | Field-level encryption (envelope, per-class key), separate bucket for files, blind index for lookups | KYC module only; admin view requires `kyc.evidence.view`, reason, watermark, audited |
| **C3 — Confidential** | Phone number, addresses, private messages, order history, device IDs, precise location, payout history | Encrypted at rest (disk/DB), phone field-encrypted + blind index | Owner, counterpart (e.g. seller sees ship-to for its orders only), scoped admin roles |
| **C2 — Internal** | Seller catalogue drafts, analytics aggregates, risk scores | Encrypted at rest | Role-based |
| **C1 — Public** | Published posts, public profiles, product pages | CDN | Public, honouring blocks/private accounts |
| **Secrets** | DB credentials, PSP keys, JWT signing keys, KMS master keys | Vault/KMS only | Workload identity; humans via break-glass |

Payment card numbers (PAN) are **never** received by VIRA; card entry happens on the PSP page. This keeps VIRA out of card-data scope; must be preserved by never embedding PSP pages in a WebView that can inject scripts.

## 2. Threat model (STRIDE, top items)

| # | Flow / asset | Threat | Category | Control | Residual |
|---|---|---|---|---|---|
| T01 | OTP login | SMS pumping / toll fraud, OTP brute force | DoS / Spoofing | Per-phone, per-device, per-prefix limits; per-IP/ASN only as anomaly signals (mobile CGNAT); 5 attempts; **self-hosted** challenge (proof-of-work or self-hosted CAPTCHA) after anomaly; cost alerting | Medium |
| T02 | Account | SIM swap → account and payout takeover | Spoofing | Detect number change/new device; Shahkar-class phone↔national-ID re-check for KYC’d users; recycled-number restricted state; recovery flow (doc 07 §4.1); 72 h cooling-off for payouts & Sheba changes; TOTP step-up; notify old device | Medium |
| T03 | Seller API | IDOR across stores | Info disclosure | `X-Store-Id` membership check + RLS + UUIDs + cross-tenant tests | Low |
| T04 | Payment callback | Forged callback marking order paid | Tampering | Callback is an untrusted browser-mediated hint bound by nonce; server-to-server verify/settle/inquiry; exact amount match; no state change from callback alone | Low |
| T05 | Payment | Double charge from retries | Tampering | Idempotency keys; `payment_pending` block; inquiry before reversal window | Low |
| T06 | Refund | Refund fraud (return empty box, collusion) | Repudiation / fraud | Evidence, seller decision, thresholds, risk scoring, maker–checker over limit | Medium |
| T07 | Payout | Insider changes Sheba / approves own batch | Elevation | Maker–checker with distinct people; Sheba owner match (national ID ↔ IBAN inquiry — **provider not yet selected**, doc 13 §4); audit; cooling-off | Medium until the Sheba-inquiry provider is contracted, then Low |
| T08 | Admin | Phished admin → mass data export | Elevation / disclosure | VPN, WebAuthn, short sessions, export permission separate, export volume alerts, watermarking | Medium |
| T09 | Media upload | Malicious file (polyglot, image-parser exploit, zip bomb) | Tampering | Quarantine, magic-byte sniff, re-encode everything, size/pixel limits, sandboxed workers with no network | Low |
| T10 | Media | Illegal content incl. CSAM | Legal | Hash-matching, classifier, priority escalation queue, preserve-and-report procedure per D-13 | Medium |
| T11 | Messaging | Scam links / off-platform payment requests | Fraud | Link scanning, warnings when sharing card/IBAN numbers, report flow, buyer-protection messaging | Medium |
| T12 | Feed/comments | Spam, bots, fake engagement | Integrity | Rate limits, device reputation, new-account limits, shadow-scoring | Medium |
| T13 | KYC | Deepfake / replay in liveness | Spoofing | Provider-side active liveness, injection detection, manual review on mismatch | Medium |
| T14 | Search/usernames | Homoglyph impersonation (Arabic ي/ك vs Persian ی/ک, ZWNJ tricks, Latin look-alikes) | Spoofing | Normalised unique index on usernames and store names; confusable check | Low |
| T15 | Mobile client | Token theft from rooted device | Spoofing | Device-bound refresh (PoP), secure storage, short access TTL, risk signal | Medium |
| T16 | Supply chain | Malicious dependency / CI compromise | Tampering | Pinned deps, SBOM, signature verification, protected branches, isolated runners, image signing | Medium |
| T17 | Feature flags | Enabling crypto/wallet via config error | Elevation | GATED flags require two approvers + not exposed to remote-config UI; server checks; build-time exclusion until approved | Low |
| T18 | Infrastructure | Volumetric DDoS / national connectivity disruption | DoS | In-country CDN/WAF, origin cloaking, degrade gracefully, no foreign runtime dependencies | Medium |
| T19 | Live | Unmoderated live broadcast harm | Legal | Seller eligibility, delay buffer option, admin emergency stop ≤ 5 s, report during live | Medium |
| T20 | Data | Excessive retention of biometrics | Privacy | Purge job, retention per D-10, minimised storage (store decision + hashes, not raw video, where provider permits) | Low |

## 3. Authentication

| Population | Factors | Session |
|---|---|---|
| Consumers | SMS OTP; optional TOTP; step-up OTP/TOTP for sensitive actions | Access 10 min; refresh 30 days sliding, rotation with reuse detection (reuse → revoke family) |
| Sellers & staff | Same as consumer + **TOTP required** for owner/finance roles | Same; re-auth for payouts and staff changes |
| Super Admin | Separate realm; WebAuthn (preferred) or TOTP mandatory; VPN | 8 h max, 15 min idle; one active session per admin by default |
| Workloads | Workload identity / short-lived certs; DB passwords issued dynamically from Vault where feasible | — |

Sensitive actions requiring **step-up**: payout request, add/change Sheba, add seller staff, change phone number, disable 2FA, account deletion, viewing full KYC data (admin).

## 4. Authorization

### 4.1 Super Admin roles (initial)

| Role | Key permissions | Notes |
|---|---|---|
| `platform_owner` | Manage admins, approve GATED flag changes (second signature) | Break-glass; ≤ 2 holders |
| `ops_manager` | Users, stores, orders read/write, config (non-financial) | |
| `support_agent` | Read users/orders (masked PII), tickets, **propose** refunds | Masked by default; unmask = reason + audit; refunds to the original payment source within policy auto-execute, anything else needs a `finance_checker`; agents never choose a refund destination (audit F-08) |
| `moderator` | Content queues, actions, appeals | No financial or KYC access |
| `trust_safety_lead` | Escalated/illegal-content queue, law-enforcement packages | |
| `kyc_reviewer` | KYC queue, evidence view | No payouts |
| `finance_maker` | Prepare payout batches, refunds, adjustments | Cannot approve own items |
| `finance_checker` | Approve batches/refunds/adjustments | Cannot create |
| `finance_viewer` | Ledger, recon reports | |
| `catalog_manager` | Categories, product moderation | |
| `marketing_manager` | Banners, coupons (budget-capped), campaigns | Coupon budgets need finance approval |
| `security_auditor` | Read audit logs, sessions | Read-only |
| `analyst` | Aggregated dashboards, no row-level PII | |

The admin RBAC screen (`vira_administrators_rbac_dark`) is the UI for this matrix; permissions are code-defined, roles are data.

### 4.2 Seller staff roles (store-scoped)

| Permission ↓ / Role → | owner | manager | inventory | fulfilment | support | finance |
|---|:-:|:-:|:-:|:-:|:-:|:-:|
| Store profile & policies | ● | ● | | | | |
| Products, prices | ● | ● | ● | | | |
| Stock | ● | ● | ● | ● | | |
| Orders: accept, ship | ● | ● | | ● | | |
| Returns decisions | ● | ● | | | ● | |
| Buyer chat | ● | ● | | | ● | |
| Balance & statements | ● | | | | | ● |
| Payout request, Sheba | ● | | | | | ● (if owner grants) |
| Staff management | ● | | | | | |

**v3 — shared identity across surfaces:** the table applies identically to mobile “Manage My Stall” and the Seller Admin web (doc 04 §10). Staff never gain access to the owner’s personal account data. Staff removal revokes store access on both surfaces immediately (session store-scope invalidation + realtime disconnect from `store.<id>`). Seller web sessions: httpOnly cookie, SameSite=Strict, CSRF token, 12 h max / 30 min idle, TOTP step-up for owner/finance actions.

### 4.3 Maker–checker rules (configurable thresholds)

Payout batch approval (always); refunds above threshold or outside policy; manual ledger adjustments (always, plus finance_viewer notification); fee/commission schedule changes; KYC overrides; GATED feature flag changes; admin role grants of `finance_*`, `kyc_reviewer`, `platform_owner`.

## 5. Abuse prevention & rate limiting

| Surface | Limit (initial, tunable) | Key |
|---|---|---|
| OTP request | 3 / 10 min, 10 / day per phone; 5 / hour per device; per-prefix anomaly limits; per-IP/ASN thresholds high and used as signals only (carrier CGNAT shares IPs across many users) | phone, device, prefix; ip as signal |
| OTP verify | 5 attempts per challenge; 20 / hour per device | challenge, device |
| Auth refresh | 30 / hour per session | session |
| Posts | new accounts: 5 / day; established: 50 / day | user |
| Comments | 60 / hour; duplicate-text detection | user |
| Follows | 100 / hour, 500 / day | user |
| DMs to non-followers | 20 / day (requests) | user |
| Checkout | 10 / hour per user; 3 concurrent pending intents | user, device |
| Search | 60 / min | user/ip |
| Admin exports | row caps + approval above cap | admin |

Implementation: token-bucket in Redis at the API; edge WAF limits for unauthenticated routes; per-route budgets declared in OpenAPI extensions so they are reviewed with the contract.

Other controls: device reputation, disposable/virtual number detection where available, new-account friction, velocity rules in the risk module (cards per account, accounts per card, refunds per buyer, returns ratio per seller).

## 6. Audit logging

* **What:** every admin action (read of C3/C4 data, every write), every money-affecting action by anyone, auth events (login, step-up, 2FA change, session revoke), role/permission changes, flag changes, data exports.
* **Format:** `who (actor id, realm, role) · what (action, resource type/id) · when · where (ip, device, session) · why (reason code + text) · before/after (diff, PII-masked) · result · trace_id`.
* **Integrity:** append-only table, no UPDATE/DELETE grants, daily hash chain (`prev_hash`), shipped hourly to write-once storage in a separate account/project.
* **Review:** weekly automated reports (unusual unmasks, exports, off-hours admin activity) to `security_auditor`.

## 7. Enforcing the crypto / wallet gate

“Disabled” must be true at every layer, not just hidden in the UI:

1. **Build:** no crypto provider client libraries in MVP builds (dependency allowlist check).
2. **API:** crypto routes not registered unless `gated.crypto` is ON **and** the binary was built with the `crypto` tag.
3. **Ledger:** currencies other than `IRR` rejected by `Post()`.
4. **Admin UI:** USDT sections removed from payments, refunds, reconciliation, platform-settings and static-content consoles.
5. **Flags:** GATED flags are not editable from the remote-config screen; changes require a signed change request + `platform_owner` second approval.
6. **Content:** static pages must not mention USDT limits (currently in `vira_static_content_app_pages_dark`).

## 8. Encryption and secrets

| Layer | Control |
|---|---|
| Transit | TLS 1.2+ everywhere (1.3 preferred), HSTS, internal mTLS between edge and origin; certificate pinning in app with backup pins |
| At rest | Disk encryption on DB/object storage; field-level envelope encryption for C4 and phone numbers (AES-256-GCM, data keys wrapped by KMS/Vault Transit) |
| Blind indexes | HMAC-SHA256 with a separate key for equality lookups on phone, national ID, Sheba |
| Keys | Vault (or HSM-backed KMS if the provider offers one) — rotation: data keys yearly or on incident; JWT signing keys quarterly with overlap |
| Secrets in CI/CD | No long-lived secrets in repos; OIDC/workload identity to Vault; secret scanning on every push |
| Backups | Encrypted with a key not stored with the backup |

## 9. Secure media uploads

Pre-signed, size-limited, content-type-bound PUT to quarantine; server-side magic-byte detection; allowlist (JPEG, PNG, HEIC, WebP, MP4/MOV with H.264/HEVC, PDF only for KYC/support); max dimensions (e.g. 8k px) and durations; decode-and-re-encode all images/video (destroys polyglots and metadata incl. GPS EXIF); decoding runs as an FFmpeg/libvips **child process** with no network namespace, a seccomp profile, read-only input, a scratch tmpfs and CPU/time/memory limits — the parent worker (which does need object storage and the queue) streams bytes in and out; documents (PDF) rendered to images for admin viewing; antivirus scan for documents; perceptual hash for moderation and dedupe; signed short-lived URLs for anything non-public.

## 10. Privacy architecture

| Principle | Implementation |
|---|---|
| Minimisation | KYC: store result + minimal identifiers; keep raw biometrics only as long as D-10 requires. Location: store address pins; “nearby” uses coarse location (≈1 km) unless user opts in |
| Purpose limitation | Event payloads carry IDs only; analytics uses pseudonymous IDs |
| Transparency | Ad transparency (“why am I seeing this”) as designed in `_56`; privacy centre in settings |
| User control | Private account, close friends, tag/mention controls, DM requests, block/mute, story audience, notification categories, download-my-data, delete account |
| Deletion | Account deletion: grace period (e.g. 30 days) → per-module erasure jobs → irreversible; financial records retained per law with PII minimised (pseudonymised buyer ID on ledger) |
| Messaging | D-11: server-readable but encrypted at rest (supports dispute evidence & abuse handling) **vs** E2EE for personal DMs (stronger privacy, limits moderation). Recommendation: store↔buyer chats server-readable with clear notice; personal DMs E2EE later if legally permissible |
| Children | Minimum age and parental rules per D-13; no targeted ads to minors |
| Admin access | Masked by default; unmask audited; analysts see aggregates only |

## 11. Monitoring, detection and incident response

* Security signals: auth anomalies (OTP spikes per prefix, refresh-token reuse), admin anomalies (exports, unmasks, off-hours), payment anomalies (verify failures, amount mismatches, refund spikes), infra (WAF blocks, 5xx bursts), integrity (ledger nightly check).
* Severity levels SEV1–SEV4 with paging rules; SEV1 examples: ledger imbalance, payout to wrong account, data breach, payment verification outage.
* **Runbooks (Phase 1 deliverable):** PSP outage, double-charge reports, payout batch failure, suspected admin compromise (revoke all admin sessions, rotate keys), data breach (containment, evidence preservation, legal notification decision), illegal content report, SMS pumping attack, international connectivity loss.
* **Kill switches** (flags, server-enforced): disable checkout, disable payouts, disable new registrations, disable uploads, read-only mode, disable live.
* Post-incident review within 5 business days, blameless, tracked actions.

## 12. Legal & regulatory questions requiring owner + counsel decisions

| ID | Question |
|---|---|
| L-01 | Legal entity, licences and jurisdiction(s) of operation; applicable e-commerce, consumer-protection and data-protection rules (D-01) |
| L-02 | Is the platform collecting funds for sellers (requires which licence, e.g. payment-facilitator status?) or using PSP split settlement (D-05) |
| L-03 | Statutory withdrawal/return period for remote purchases and who bears return shipping (D-08) — UI already references a statutory withdrawal window (`_198`) |
| L-04 | Use of the terms “escrow / حساب امانی / guarantee” in UI copy (D-06) |
| L-05 | Stored value / wallet licensing (D-03) |
| L-06 | Crypto legality, sanctions exposure and central-bank rules (D-04) |
| L-07 | KYC obligations, permissible identity data sources, biometric retention (D-10) |
| L-08 | Tax on commission, seller tax identity, e-invoicing obligations, marketplace withholding (D-07) |
| L-09 | Lawful-access and data-retention obligations for messages and calls (D-11) |
| L-10 | Content liability, takedown timelines, child-safety reporting obligations (D-13) |
| L-11 | Data residency and cross-border transfer (hosting, SMS, push, AI providers) (D-01, D-17, D-18, D-21) |
| L-12 | Validity of in-app digital signature for seller agreements (D-15) |
| L-13 | App-store distribution and sanctions compliance for iOS/Android (D-16) |

## 13. v2 — security requirements confirmed by UI verification (doc 15)

These are **not** architecture changes; they are rules the UI designs must be brought into line with.

| # | Rule | Violating designs |
|---|---|---|
| S-UI-01 | No screen in the VIRA app or web may contain card-number, CVV2, expiry or dynamic-PIN fields, or imitate a bank/PSP page. Payment = redirect to the PSP-hosted page in the system browser / custom tab. | `_41`, `_142` (rejected) |
| S-UI-02 | Card numbers needed for ownership checks are entered only into a PSP/inquiry-provider flow, or masked and tokenised immediately; VIRA stores the inquiry result and last 4 digits only. “Save card” only via PSP tokenisation (D-34). | `_66`, `bank_card_ownership_verification_light`, `_142` |
| S-UI-03 | Admin finance UI: no global “instant settlement” action (currently in 17 console headers); every money-moving action shows amount, destination and policy check, requires a reason code, and goes through maker–checker (§4.3). No early release of holds without a checker. Reuse the dual-control pattern already drawn in `vira_wallet_monitoring_risk_dark`. | payout, hold-release, refund, reconciliation consoles |
| S-UI-04 | Admin data views mask phone, e-mail, national ID, Sheba and card data by default — in list **and** detail views; unmask is a separate, audited action (§10). No search by full card number. | users, orders, stalls, withdrawals, payments consoles |
| S-UI-05 | Admin step-up (WebAuthn/TOTP) is required on approval actions, not only at login; the RBAC page’s “hardware/TOTP 100%” claim must be enforced in the realm (ADR-016). | finance and KYC consoles |
| S-UI-06 | One authentication model (D-22). If a “financial PIN” is adopted for withdrawals it is an additional knowledge factor stored as a salted slow hash with attempt limits — never a replacement for TOTP/WebAuthn step-up. Biometric login uses platform keystore-backed keys (device unlock of the refresh-token key), never biometric data on the server. | `_59`, `_178` |
| S-UI-07 | Crypto elements are absent from MVP builds and copy (doc 08 §7), including admin sidebars (cold storage, multi-sig, OTC). | 78 screens |
| S-UI-08 | Consent is explicit and versioned (terms, privacy, biometric processing); no “by logging in you accept”; **no consent or attestation box is pre-ticked**. | `otp`, `level_2_*` intro, `_142`, `_178`, `_163` |
| S-UI-09 | No government domains, real bank seals or third-party brand marks used to imply endorsement. | `vira_administrators_rbac_dark` (`vira.gov.ir`), `_142` |
| S-UI-10 | Location in courier tracking and “nearby” features follows §10 (coarse by default; rider location visible only during an active delivery). | `_106`, `_44`, `_70` |
