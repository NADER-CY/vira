# 19 — UI Security Correction Specification

**Status:** Phase 0 v3 · 2026-10-29 · Owner decision **B-7**: security corrections accepted in principle.
**Scope:** redesign *specifications* only. The reference archive `ویرا.zip` is **not modified**, and no screen is implemented in Phase 0. Implementation happens in the phase that owns each journey (doc 12).

## 1. Ground rules for every redesign

1. **Preserve the identity.** Keep VIRA’s visual language from the archive: crimson primary (single seed per DS-01), dark and light themes generated from tokens, rounded cards, bold Persian headings in Vazirmatn, RTL layout, step indicators, status timelines, existing icons and illustrations. Re-use the archive screens named below as layout templates.
2. **Preserve the journey.** The user should reach the same outcome in the same number of steps or fewer. Only the unsafe element changes.
3. **Security rules are normative** (doc 08 §13, S-UI-01…10). Where this document and a screen disagree, this document wins.
4. **Copy** is given as intent; final Persian copy goes through content review. Sample strings are illustrative.
5. Every corrected flow has **acceptance criteria** that QA will test (doc 11).

## 2. Correction index

| # | Unsafe pattern (B-7) | Affected reference screens | Owning phase |
|---|---|---|---|
| UC-1 | In-app collection of card number, CVV2, expiry, dynamic PIN | `_41`, `_142`; related copy in `_92` | P6 |
| UC-2 | Card number entered/scanned for ownership or KYC | `_66`, `bank_card_ownership_verification_light` | P4 |
| UC-3 | Saving raw card details in VIRA | `_142` (save-card toggle, pre-ticked), `_129` (“saved Shetab cards”) | P6 |
| UC-4 | Searching payment records by full card number | `vira_payments_management_dark` | P6 |
| UC-5 | Pre-selected biometric, payment or financial consents | `level_2_identity_verification_introduction_light`, `_142`, `_178`, `_163`; implicit consent on `otp` | P2 (consent), P4 (KYC), P6/P7 (money) |
| UC-6 | Single-actor approval of high-risk disbursements | `vira_withdrawals_settlements_dark`, `vira_financial_reconciliation_settlements_dark`, `vira_escrow_automated_commission_dark`, `vira_refunds_returns_management_dark`; “instant settlement” header button on 17 consoles | P6/P7 |
| UC-7 | Unnecessary display of unmasked sensitive data | `vira_users_management`, `vira_orders_management_dark` (detail), `vira_stalls_management`, `vira_withdrawals_settlements_dark` | P2 onward |
| UC-8 | Crypto payment options enabled/pre-selected by default | `_164`, `_168`, `_169`, `_170`, `_172`, `_177`; admin USDT sections | MVP: removed; gated track G2 |
| UC-9 (aligned) | OTP length inconsistent (4 / 5 digits) | `otp`, `mobile_otp_verification_light` | P2 |
| UC-10 (aligned) | Refund destination defaults to in-app wallet | `_90`, `_120`, `_124`, `_197` | P7 (policy D-03/D-08) |

---

## UC-1 Payment: hand-off to the PSP-hosted page

**Problem.** `_41` and `_142` render card fields inside VIRA (and `_142` imitates a bank gateway). This puts card data in VIRA’s scope and trains users to type card secrets into non-bank pages.

**Preserved intent.** A clear, reassuring payment step showing the amount, order and security status, followed by a result screen.

**Corrected flow**

1. **Checkout review** (template: `_80`, `_69`): addresses, parcels per store, totals. Primary button: «پرداخت امن از درگاه بانکی» (“Pay securely via bank gateway”).
2. **Hand-off sheet** (new, styled like the `_16` card): amount, order reference, the PSP’s name/logo as provided by the contracted PSP, and the line «اطلاعات کارت را فقط در صفحهٔ درگاه بانک وارد کنید؛ ویرا هرگز رمز دوم یا CVV2 را از شما نمی‌خواهد» (reuses the anti-phishing copy already in `_66`/`_83`). Button opens the PSP URL in the **system browser / custom tab** (never an in-app WebView that can read fields).
3. **Return** via app link → **Result screens** kept from the archive: success `_132`, failed `_16` / `_129`, pending `_219` / `_223` (“do not pay again”, live inquiry).
4. Payment method list (`_164` template) shows **only approved methods** (card IPG at MVP), none pre-selected unless there is exactly one.

**Remove:** all card inputs, captcha, virtual keyboard, “request dynamic PIN” button, bank seals/text imitating a bank page, “sandbox test card” strings outside staging.

**Acceptance criteria**
* No screen in the app contains an input with card number, CVV2, expiry or PIN semantics (automated UI-text and widget scan in CI).
* The PSP page opens outside the app’s WebView; returning without completing shows the pending/failed state from the server, never a client-decided result.
* `_92`-style sandbox guidance appears **only** in the staging build flavour.

## UC-2 Ownership and KYC without in-app card capture

**Problem.** `_66` and `bank_card_ownership_verification_light` collect the full 16-digit card number (and `_66` offers a camera scan of the card) to prove the user owns the account.

**Preserved intent.** Confirm the payer/seller identity matches their bank account, with the same two-step progress UI and the “verified” success card.

**Corrected flow**

1. Primary method: **Sheba (IBAN) ownership inquiry** — user enters Sheba (`IR` + 24 digits, LTR-isolated field with checksum validation); VIRA sends Sheba + national ID to the inquiry provider (doc 13 §4) and shows the masked result: holder name, bank, `IR48 **** **** **** **** 0560`.
2. If the provider requires a **card** inquiry instead: the card number is entered on a **provider-hosted** page, or sent **client-to-provider only — never via the VIRA API**; VIRA stores only the inquiry result, bank, **first-6 / last-4**, and a provider reference.
3. Camera card scanning: **removed** at MVP (avoids image capture of PAN/CVV). If later required, OCR must run on-device, never upload the image, and redact CVV areas.
4. Success card (template: `_73`, `buyer_level_1_approved_light`) shows the masked account and the new limits.

**Acceptance criteria:** no request body or log line from VIRA’s app/API contains a full PAN (CI log-scanner + DLP rule); database schema has no column capable of storing a PAN; masked values only in UI.

## UC-3 Saved cards

**Corrected design.** If the contracted PSP offers **tokenised card-on-file** (D-34), the PSP page itself asks the user whether to save the card; VIRA receives only a token, card mask and expiry month/year (for display). In VIRA’s payment-method list (`_129` template) saved cards appear as «کارت ذخیره‌شده در درگاه … ۶۰۳۷‑۹۹**‑****‑۴۴۱۹» with a **Remove** action. If tokenisation is not offered, the saved-cards section is not shown. There is **no** “save card in VIRA” toggle anywhere.

**Acceptance criteria:** no saved-card UI unless the PSP token API is integrated; removal deletes the token at the PSP and in VIRA.

## UC-4 Admin payment search

**Corrected design** (template: `vira_payments_management_dark` layout preserved): search fields = order reference (`VR-…`), payment intent ID, PSP reference / RRN, masked card (**first-6 + last-4** as two separate fields), amount range, date range, status. Results show masked card only. No free-text full-PAN search box.

**Acceptance criteria:** search API rejects inputs matching a 16-digit card pattern; audit records every search query (filters only).

## UC-5 Consent and attestations

**Corrected pattern** (applies to all listed screens):

* Each purpose has its **own** checkbox (e.g. terms of use; privacy notice; biometric processing for KYC; saving a card at the PSP; crypto risk disclosure when that track is approved; withdrawal-destination ownership attestation).
* **Unticked by default.** Primary button disabled until required boxes are ticked; optional purposes never block the flow.
* Each label links to the full text; the version of the text is recorded.
* Sign-up (`otp`): replace «با ورود … می‌پذیرید» with an explicit checkbox for terms + privacy; add an **age confirmation** per D-13.
* KYC level 2 intro (`level_2_identity_verification_introduction_light`): keep the layout; biometric consent checkbox unticked, with a short explanation of retention (D-10).
* Withdrawal confirmation (`_178`): ownership attestation unticked; TOTP/OTP step-up kept.

**Acceptance criteria:** consent events stored as `(user, purpose, text_version, granted_at, surface, ip/device)`; withdrawal of consent available in Settings → Privacy; UI tests assert no checkbox is pre-ticked.

## UC-6 Admin disbursements with dual control

**Reference pattern to reuse:** `vira_wallet_monitoring_risk_dark` already draws dual control (reason field → senior approval → CFO inbox). Apply that pattern to every money-out console.

**Corrected flow for payouts, early hold release and refunds above threshold**

1. **Maker** prepares a request/batch: amount, destination (masked), policy checks (KYC level, cooling-off after Sheba/phone change, open disputes, risk score), **reason code + note**. Submit → status `awaiting_approval`.
2. **Checker** (different person, permission `*.approve`) sees the same evidence plus the maker’s identity, approves or rejects with a reason, using **WebAuthn/TOTP step-up**.
3. Execution is performed by the system (job), not by a button that moves money synchronously; status timeline shows `approved → submitted → confirmed/failed/returned`.
4. **Emergency freeze** stays a single-actor control (it stops money, it does not move it), audited.

**Remove:** the global «تسویه فوری» header button from all consoles; single-password batch approval; one-click “manual release” without a checker; direct “send to Paya web service” without approval state.

**Acceptance criteria:** the API rejects approval by the maker; every disbursement needs **maker + one checker** (two different people); amounts above the high-value threshold need **maker + two checkers** (three different people, one with `finance_checker` and one with `platform_owner` or CFO role); every state change produces an audit event; integration tests cover reject, partial failure and bank returns (doc 11 §2 #4e).

## UC-7 Masking of sensitive personal data in admin

**Corrected pattern:**

| Data | Default display | Reveal |
|---|---|---|
| Mobile number | `۰۹۱۲•••۴۵۶۷` | Button «نمایش» → reason required → audited → visible 60 s |
| E-mail | `s•••@domain` | same |
| National ID | `۰۰۸•••••۰۱۲` | only roles with `kyc.*` or `support.unmask`; reason required |
| Address | City + district | Full address only for fulfilment/support roles on that order |
| Sheba / card | masked (first/last digits) | never fully revealed in admin |

Same rule in **list and detail** views, exports (masked unless an export-with-PII permission and approval exist), and screenshots in tickets.

**Acceptance criteria:** role × field matrix tests; audit event per reveal; exports watermarked with admin ID.

## UC-8 Crypto options not enabled by default

**MVP:** USDT payment, USDT payout, conversion, crypto wallet and admin USDT sections are **not rendered** (feature flags OFF at build and server, doc 08 §7). Payment-method lists show only approved Toman methods.

**When gated track G2 is approved (future):** USDT options are never pre-selected or promoted with discounts by default; ordering is neutral; each use shows a risk disclosure with an unticked acknowledgement; eligibility (KYC tier, seller tier) is enforced server-side; rate quotes are time-limited and shown with fees; withdrawals keep address-book allow-listing, cooling-off and dual control above thresholds.

## UC-9 OTP input (aligned)

Use a single OTP component with **6 digits**, auto-fill from SMS where the OS allows, resend countdown, lockout message after 5 attempts, and “edit number”. Templates: `mobile_otp_verification_light` layout, `otp` welcome card.

## UC-10 Refund destination (aligned, policy pending D-03/D-08)

Seller decision screens (`_90`, `_120`) and buyer return screens (`_124`, `_197`) keep their layout, but the refund destination shown is the **policy default** (refund to the original payment source via the PSP). An alternative destination (e.g. a verified Sheba in the buyer’s name) appears only if policy allows it and always requires owner-match verification and a finance checker (audit F-08). Wallet credit appears only if the wallet track (G1) is approved.

## 3. Traceability

| UC | Security rules (doc 08) | Findings (doc 15) | Tests (doc 11) |
|---|---|---|---|
| UC-1 | S-UI-01, §1 | V-01 | UI scan for card fields; PSP simulator matrix |
| UC-2 | S-UI-02 | V-01 | PAN DLP log scan; schema check |
| UC-3 | S-UI-02 | V-01, V-18 | token lifecycle tests |
| UC-4 | S-UI-04 | V-20 | API input validation |
| UC-5 | S-UI-08 | V-18 | no pre-ticked boxes; consent log |
| UC-6 | S-UI-03, S-UI-05, §4.3 | V-06 | maker ≠ checker; thresholds |
| UC-7 | S-UI-04, §10 | V-06 | role × field matrix |
| UC-8 | S-UI-07, §7 | V-02, V-19 | build/flag absence tests |
| UC-9 | §5 | V-08 | OTP lockout tests |
| UC-10 | §4.1, audit F-08 | V-04 | refund-destination policy tests |
