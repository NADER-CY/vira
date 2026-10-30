# Phase 2 — remaining UI differences from the archive

Basis: the 34 sheets in this folder (archive screenshot ↔ Flutter golden), checked on 2026-10-30. Colours, type,
spacing, radii and icons come verbatim from the archive tokens (`design/tokens/vira.mobile-archive.tokens.json`).
Nothing below has been redesigned; each item is either content of a later phase, a recorded reference conflict, or a
security/accuracy text fix that needs the owner's approval to change further.

| Screen | Difference | Kind | Owner action |
|---|---|---|---|
| All feed/profile screens | Photos render as placeholders in goldens (synthetic test data, no network images) | Test data only | — |
| Home (`home_feed_light`, `_33`) | Stories row shows only "your story"; product tag chip on post image absent | Phase 3 (stories) / Phase 4 (product tags) | — |
| Explore (`_26`) | Discovery grid, trend chips, visual/voice search replaced by a "Phase 3" empty state; only people search is live. `_26` is dark-only; the light theme uses the home light skin | Phase 3 content; no light reference | Confirm light treatment |
| Profile (`user_profile_light`) | Store blocks (rating, official badge, top-seller card, storefront button, highlights with products) not drawn for P2 accounts | Phase 4 seller data | — |
| Post detail | Title bar «پست @handle» — no dedicated reference; reuses the feed post card | No reference | Approve or supply reference |
| Edit profile (`_17`) | Pronoun, category, e-mail, gender, business toggles shown with their phase instead of editable | Later phases | — |
| Followers (`_109`) | "Suggested for you" block omitted (recommendations are Phase 9) | Later phase | — |
| Comments (`comments`) | Emoji quick-reaction row and dimmed post image header omitted | Minor, Phase 3 media | Approve |
| Create post (`create_post_light`) | Audience row added in the same option-row design (OD-P2-11); product-link block shows its phase | Conflict / later phase | OD-P2-11 |
| Activity (`_12`) | Only follow requests are live; likes/comments/orders notifications show a Phase 5 state | Phase 5 (notifications) | — |
| Privacy (`_125`) | "Comments and likes" card added (comments-from, hide like counts) in the same card design, because Phase 2 enforces these settings | Addition | **Approve or remove** |
| Security (`_205`) | 2FA authenticator / backup-code toggles off and disabled; "SSL v3 certified" replaced with "HTTPS (TLS)" | Security accuracy (report §5) | **Approve wording** |
| Account (`_206`) | Wallet / open-orders deletion checks say "not applicable yet" | G1 / Phase 7 | — |
| Report (`_213`) | Reported-item card without product photo; safety reasons added as extra cards (OD-P2-10); «۱۰۰٪ ناشناس» / end-to-end wording softened | Conflict / accuracy | OD-P2-10, approve wording |
| OTP (`otp`, `mobile_otp_verification_light`) | 6 boxes instead of 4/5; consents as three unticked boxes | Security (UC-5, UC-9; OD-P2-01) | OD-P2-01 |
| Dark variants of privacy, settings, security, account, edit profile | Light-reference layout used in both themes (OD-P2-05…09) | Conflict | OD-P2-05…09 |

Super Admin consoles keep the P1 sidebar shell accepted in Phase 1; page content follows `vira_users_management`
and `vira_comments_reports_dark`. Differences: masked phones only, no e-mail column, no bulk checkboxes / CSV
export (no P2 backend), Shahkar tile inactive (P4), "instant settlement" header button not reproduced (doc 19 V-06).
