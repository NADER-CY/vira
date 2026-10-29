# VIRA — Phase 0 Architecture Package

**Status:** **v4 — final, ready for product-owner acceptance** (v4 = Claude-led execution strategy, doc 20). Not yet accepted. **Phase 1 is NOT AUTHORISED.** Phase 0 is documentation only: no code, no infrastructure provisioning, no repository changes; the UI reference archive was not modified.
**Date:** 2026-10-29 · **Revision:** v4 · **Language:** English (source UI is Persian)

**Start here:** doc 20 (Claude-led execution strategy, v4) → doc 16 (final GO/NO-GO) → doc 13 §0 (final decision register) → doc 01 (summary).

## Contents

| # | Document | What it answers |
|---|---|---|
| 01 | [Executive Summary](01_Executive_Summary.md) | What VIRA is, what we recommend, what blocks Phase 1 |
| 02 | [UI Screen Inventory](02_UI_Screen_Inventory.md) + [CSV](02a_VIRA_Screen_Inventory.csv) | **v2:** all 330 unique screens verified (HTML for all; visual for 63); rendered theme, flags, parsed inputs |
| 03 | [Product Requirements & Gap Analysis](03_Product_Requirements_and_Gap_Analysis.md) | Candidate requirements, missing flows, contradictions, MVP boundary |
| 04 | [System & Domain Architecture](04_System_and_Domain_Architecture.md) | Modules, boundaries, consistency, topology, trust boundaries |
| 05 | [Architecture Decision Records](05_Architecture_Decision_Records.md) | 22 ADRs + **v3 final acceptance matrix** |
| 06 | [Database Architecture & ERD](06_Database_Architecture_and_ERD.md) | Schemas, ERDs, order/ledger design, reconciliation, backups |
| 07 | [API & Event Contracts](07_API_and_Event_Contracts.md) | Conventions, endpoints, realtime protocol, critical workflows, events, jobs, retries |
| 08 | [Security & Privacy Architecture](08_Security_and_Privacy_Architecture.md) | Threat model, authn/authz, abuse, audit, crypto gate, privacy, legal questions |
| 09 | [Design System & Navigation](09_Design_System_and_Navigation_Architecture.md) | Token reconciliation, RTL rules, IA, states, accessibility |
| 10 | [Infrastructure, Scalability & Cost](10_Infrastructure_Scalability_and_Cost.md) | Environments, sizing tiers, cost drivers, SLOs, DR objectives |
| 11 | [Testing & Release Strategy](11_Testing_and_Release_Strategy.md) | Test layers, financial-correctness suite, release & rollback |
| 12 | [Phased Development Roadmap](12_Phased_Development_Roadmap.md) | P0–P7 (MVP) with all 10 required attributes; **v3: financial gates FG-1a (model decision) and FG-1b (real-money activation), long-term tracks P8–P12, gated tracks G1–G3** |
| 13 | [Decision Register, Dependencies & Risks](13_Open_Decisions_and_Risk_Register.md) | **v3 §0: final product-owner decision register** (B-1…B-7, D-01…D-35 classified), Phase 1 blockers, 52 risks, external dependencies |
| 14 | [Independent Architecture Audit](14_Independent_Architecture_Audit.md) | Round 1 audit, round 2 (UI) addendum, **round 3 final consistency review (§7)** |
| 15 | [UI Verification Report](15_UI_Verification_Report.md) | **v2:** archive integrity, resolution of the 42 open records, findings V-01…V-20 (independently spot-checked), remaining Phase 0 blockers, GO/NO-GO |
| 16 | [Phase 0 Final GO/NO-GO Report](16_Phase0_GO_NO-GO_Report.md) | **v3 final:** complete / open / prohibited, Phase 1 blockers, approval gates (v2 report kept as appendix) |
| 17 | [Budget & Team Scenarios](17_Budget_and_Team_Scenarios.md) | **v3:** Scenarios A/B/C — roles, costs (Toman list-price snapshot, to be quoted), timelines, capacity, triggers |
| 18 | [Iranian Hosting Provider Comparison](18_Iranian_Hosting_Provider_Comparison.md) | **v3:** providers matrix, verified vs assumed, primary + recovery strategy, vendor questions |
| 19 | [UI Security Correction Specification](19_UI_Security_Correction_Specification.md) | **v3:** redesign specs UC-1…UC-10 for the rejected patterns |
| 20 | [Development Execution Strategy](20_Development_Execution_Strategy.md) | **v4:** Claude-led execution, simplified roadmap, minimal dev infrastructure, Phase 0 summary, Phase 1 readiness checklist |

## Conventions used across documents

* Screen IDs `M-001…M-277` (mobile) and `A-001…A-053` (Super Admin) — see the CSV.
* Requirement IDs `R-<DOMAIN>-nn`, gaps `G-…`, contradictions `C-nn`, decisions `D-nn`, risks `R-nn`, legal questions `L-nn`, ADRs `ADR-nnn`, audit findings `F-nn`.
* Money is integer Rial (IRR) in every data structure; Toman is display only.
* Diagrams are Mermaid (render on GitHub/GitLab or any Mermaid viewer); all were validated with mermaid-cli.

## Evidence

v1 used only the UI audit manifest. **v2** verified the original `ویرا.zip`: all 383 HTML and 6 DESIGN.md files match the manifest hashes; every screen was parsed; rendered themes were measured from the screenshots; the 42 previously open records and the priority financial, identity, checkout, seller and admin screens were inspected visually. The archive was never modified.

## Revision history

| Version | Date | What changed |
|---|---|---|
| v1 | 2026-10-28 | Architecture package 00–14 from the UI manifest; round-1 independent audit (30 findings) incorporated |
| v2 | 2026-10-28 | Original archive verified; 42 open screen records resolved; docs 15, 16 added; 02, 03, 08, 09, 12, 13, 14 updated; independent spot-check incorporated |
| v3 | 2026-10-29 | Owner decisions B-1…B-7 recorded; docs 17 (budget & team), 18 (hosting), 19 (UI security corrections) added; ADR acceptance matrix and amendments (003, 009, 010, 011, 012, 017, 019); shared seller identity (04 §10, 07 §8, 08 §4.2); financial-model gate (06 §6.5); roadmap gates (12 §v3); final register (13 §0); final GO/NO-GO (16); architect self-review (14 §7) and independent final review — 22 findings (1 HIGH, 10 MEDIUM, 11 LOW), all resolved: FG-1 split into FG-1a/FG-1b, RETAINED decision class, permission-scoped seller topics, B-4 mobile distribution evaluation (11 §4.1), managed-DB costing, price provenance relabelled (14 §8) |
| v4 | 2026-10-29 | Owner instruction “Claude-Led End-to-End Development”: doc 20 added (execution strategy, simplified roadmap with launch-readiness gate L, minimal dev infrastructure, Phase 0 summary, Phase 1 readiness checklist); team scenarios in doc 17 superseded for development staffing (infrastructure models kept); B-3/D-14 and P1 blockers updated (13 §0, 16); FG-1b staffing condition replaced; risks R-48…R-52; staging cost added to `evidence/cost_model.py`. Architecture and ADRs unchanged |

Earlier content is preserved in place (marked v1/v2) or as appendices; v3 sections supersede them where they differ.

## Supporting files

* `evidence/` — HTML extraction of all screens, both independent reviews, contact sheets used for visual verification, and `cost_model.py` (reproducible infrastructure cost model for doc 17).
