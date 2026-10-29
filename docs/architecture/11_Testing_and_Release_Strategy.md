# 11 — Testing and Release Strategy

**Status:** Phase 0 v3 (final for owner acceptance; v2/v3 additions marked).

## 1. Test pyramid and gates

| Layer | Scope | Tooling (indicative) | Gate |
|---|---|---|---|
| Static | Lint, vet, architecture rules (module imports, no float in money pkgs), secret scan, licence scan | golangci-lint, custom go/analysis, gitleaks, dependency licence check | Every PR |
| Unit | Pure domain logic: pricing, fee calc, state machines, Persian normalisation, Jalali formatting | Go `testing`, Dart `test`, Vitest | Every PR, ≥ 80% on money & authz packages |
| Property-based | Ledger invariants (every entry balances; replays don’t double-post), inventory never negative under random concurrent ops, price allocation sums | `rapid` (Go) | Every PR touching money/inventory |
| Integration | Module + real PostgreSQL/Redis/NATS/MinIO in containers; RLS isolation; migrations up from empty and from previous release | Testcontainers | Every PR |
| Contract | API responses validate against OpenAPI; events validate against JSON Schema; generated clients compile | Schemathesis-class fuzzing, schema validators | Every PR |
| Provider simulation | PSP simulator implementing: success, decline, timeout, duplicate callback, late callback, amount mismatch, verify failure, inquiry-only success, reversal | In-house simulator service | Every PR in payments; nightly full matrix |
| End-to-end | Critical journeys J1, J3, J5–J8, J10, J13, J14 on staging | Flutter `integration_test` + Patrol-class; Playwright for web | Release candidate |
| Visual / golden | Every design-system component and key screens: light/dark × RTL × text scale 1.0/2.0 | Flutter golden tests; Playwright screenshots | Every PR touching UI |
| Accessibility | Semantics, contrast, focus order | Flutter semantics tests, axe for web | RC |
| Performance | Load at tier target × 2; soak 8 h; spike; reconnect storms | k6 | Before each phase exit and before major releases |
| Resilience | Kill DB primary, kill NATS node, PSP outage, SMS outage, object-storage slowdown | Fault injection in staging | Phase exits for P5, P6, P7 |
| Security | SAST (gosec, semgrep), dependency audit, DAST (ZAP) on staging, mobile (MobSF-class), manual pentest | — | PR (SAST) / RC (DAST) / phase exit (external pentest for P6 and P7) |
| Restore drill | Restore prod backup to isolated env + ledger integrity | Automated | Monthly |

## 2. Financial correctness test suite (non-negotiable)

1. Double-entry invariant on every journal across randomised scenario generators.
2. Idempotency: replay every PSP callback 1–5 times, in any order, concurrently → exactly one capture and one journal.
3. Race: callback and inquiry job verify simultaneously → one capture.
4. Late success after checkout expiry (T7) → per-seller re-reservation; unreservable parts refunded automatically; no order left in `awaiting_payment`.
4a. Races (audit F-02): `ExpireCheckout` vs callback vs inquiry vs buyer retry, run concurrently in every interleaving the scheduler can produce — `captured` is never overwritten, stock is never released while an attempt is non-terminal, and a verified payment always ends captured or refunded.
4b. Verify succeeded but T2 failed (injected DB error) → reconcile job completes the capture.
4c. PSP returns “already verified/settled” on a duplicate verify → mapped to success after inquiry, never to failure.
4d. Post-release refund larger than available balance → seller receivable created; payouts blocked until netted.
4e. Payout rejected by checker, failed, and returned days after confirmation → funds back in available balance exactly once.
5. Partial returns across multiple sellers in one purchase → per-seller balances exact to the Rial.
6. Rounding: discount allocation across items/sellers sums exactly to the discount.
7. Payout failure → funds return to available balance; no money created or lost.
8. Reconciliation: synthetic PSP report with missing/extra/mismatched rows produces exactly the expected breaks.
9. Fee schedule change mid-flight does not affect existing orders (snapshot).
10. Ledger cannot be updated/deleted by any application role (permission test).

## 3. Security & isolation test suite

* Cross-tenant: for every seller endpoint, a token for Store A attempting Store B’s resource IDs returns 404/403, including list filters and exports.
* IDOR on buyer resources (orders, addresses, tickets, messages).
* Seller realtime and notification scoping: each staff role subscribes only to the `store.<id>.<scope>` topics and receives only the notification categories its permissions allow; role change or removal revokes subscriptions immediately (doc 04 §10.4–10.5).
* Authz matrix tests generated from the permission table (doc 08 §4) — every role × every admin route.
* Rate-limit tests for OTP and checkout.
* Upload abuse corpus: polyglots, oversized dimensions, malformed MP4, EXIF GPS present → stripped.
* GATED features: crypto routes absent in default build; flag flip without second approval fails.

## 4. Mobile release

| Aspect | Policy |
|---|---|
| Channels | Android: domestic stores and/or direct APK (D-16); iOS per D-16 |
| Versioning | Semver + build number; `min_supported_version` and `recommended_version` in remote config |
| Staged rollout | Where the store supports it: 5% → 20% → 50% → 100% over ≥ 72 h; otherwise flag-gated features with server-side percentage rollout |
| Crash gate | Halt if crash-free sessions < 99.5% or new top-crash |
| Feature flags | All new features ship dark; enabled per cohort |
| Old versions | API supports N-2 app versions; force-update only for security or money-correctness bugs |
| Signing | Keys in HSM/secure vault; two people for release signing |

### 4.1 Mobile distribution evaluation (v3, owner decision B-4)

**Status:** evaluation for D-16. Evidence is limited: the relevant store and news pages could not be read automatically (doc `evidence/sources.md`), so platform-policy statements below are **S** (widely reported, to be confirmed) or **U** (unknown). Counsel must confirm anything touching sanctions.

| Channel | Fit for VIRA | Evidence | Key risks / unknowns |
|---|---|---|---|
| **Cafe Bazaar (کافه‌بازار)** — domestic Android store | Primary Android channel | S: largest domestic Android store; developer accounts for Iranian persons/companies | U: whether a physical-goods marketplace may use its own PSP instead of store billing (expected yes for physical goods — confirm); review times; staged rollout support |
| **Myket (مایکت)** — domestic Android store | Secondary Android channel | S | Same questions as Bazaar |
| **Direct APK** from VIRA’s website | Fallback and fastest update path | Technically certain | Users must allow unknown sources; signing-key protection; VIRA’s own update check (`config/bootstrap` min-version gate, ADR-003 amendment) |
| **Google Play** | Reach for users with Play only | S: access for Iran-based developer accounts is restricted under sanctions policies | LEGAL: whether any lawful publishing path exists for VIRA’s entity |
| **Apple App Store** | Native iOS | S: Apple does not offer developer programme access to Iran-based persons/entities under sanctions | LEGAL: lawful path (e.g. a non-Iranian publishing entity) — counsel decision |
| **Iranian iOS stores** (enterprise/ad-hoc certificate or profile installs) | Not recommended for a financial app | S: certificates are periodically revoked; distribution model conflicts with Apple’s terms | App breaks without notice; trust and security concerns for payments/KYC |
| **PWA / web app on iOS (Safari, add to home screen)** | Interim iOS reach | Technically feasible; web push for installed web apps exists on recent iOS versions (S) | Reduced camera/media capability vs native; payment hand-off works via redirect; technology (Flutter web vs React) chosen at its gate |

**Recommendation (for owner decision D-16):** release **Android** through **Cafe Bazaar + Myket + direct APK**; keep the Flutter code **iOS-ready** (ADR-003 amendment); offer iOS users a **web app (PWA)** covering browsing, social viewing, cart and checkout as an interim path; pursue native iOS only after counsel confirms a lawful App Store route. Google Play only if counsel confirms a lawful path. **Not a Phase 1 blocker**; needed before the first public release.

**Questions for the stores and counsel:** (1) developer-account requirements (legal entity, national ID, e-Namad); (2) whether external payment via VIRA’s PSP is allowed for physical goods and for seller subscriptions/ads (digital services); (3) review SLAs, staged rollout and forced-update support; (4) policies on social content, live streaming and user-generated content; (5) store revenue share for digital items; (6) data the store collects; (7) lawful App Store / Google Play publishing options; (8) obligations if an iOS build is distributed by a foreign affiliate.

## 5. Backend release

* Trunk-based development, short-lived branches, required review (two reviewers for `payments`, `ledger`, `settlement`, `authz`, migrations).
* Build once → sign image → deploy to dev → staging (automated E2E + smoke) → prod (manual approval).
* `vira-api`: blue/green with automated rollback on error-rate or latency SLO breach within 15 min.
* `vira-realtime`: drain connections gradually; clients reconnect with jitter.
* `vira-worker`: rolling; jobs are idempotent so mid-job restarts are safe.
* Migrations: expand/contract only; migration step runs before deploy; contract step only after all running code no longer uses old structures.
* Change freeze windows around peak sales events (e.g. Yalda, Nowruz) for money modules.

## 6. Rollback principles

| Change type | Rollback |
|---|---|
| Stateless code | Redeploy previous image (minutes) |
| Feature | Flip flag off (seconds) |
| Schema (expand) | Leave in place; old code ignores new columns |
| Schema (contract) | Not rolled back — only executed after a full release cycle of verification |
| Ledger posting rule bug | Never delete; post reversing entries via audited adjustment tooling with maker–checker |
| Mobile | Remote-config disable + hotfix release; cannot recall installed binaries — hence server-side gating |

## 7. Definition of Done (every story)

Contract updated · tests at each relevant layer · authz rule added to matrix · audit events defined (if admin/money) · metrics & alerts · Persian copy reviewed · RTL golden updated · feature flag · runbook updated (if operational) · threat-model delta noted.
