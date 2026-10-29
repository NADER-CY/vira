# VIRA

Persian-first, RTL social-commerce platform (Iran). Monorepo.

| Path | What |
|---|---|
| `backend/` | Go modular monolith — runtimes `api`, `admin-api`, `worker`, `realtime`, `media`, plus `migrate` |
| `mobile/` | Flutter app (Android first, iOS-compatible) |
| `web/admin/` | Super Admin web (React + TypeScript + Vite) |
| `web/seller/` | Seller Admin web (React + TypeScript + Vite) |
| `web/shared/` | Shared design tokens, RTL styles, Vazirmatn, API client |
| `contracts/` | OpenAPI and event schemas |
| `deploy/` | Local Docker Compose stack and staging configuration |
| `docs/architecture/` | Approved Phase 0 v4 architecture (source of truth) |
| `docs/progress/` | Per-phase progress and completion reports |

Status: **Phase 1 — Foundations** (see `docs/progress/P1.md`). Regulated financial capabilities
(real-money payments, settlement, withdrawals, wallet, USDT, credit/BNPL) are disabled and locked until their approval gates.

Development guide: `docs/DEVELOPMENT.md`.

Never commit secrets. Copy `.env.example` to `.env` for local development; `.env` is git-ignored.
