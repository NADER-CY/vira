// Request/response types generated from /contracts/openapi (npm run gen:api → admin.d.ts, public.d.ts).
// The web apps use these aliases so a contract change breaks the type check, not production.
import type { components as Admin, operations as AdminOps } from "./admin";
import type { components as Public } from "./public";

export type AdminPrincipal = Admin["schemas"]["Admin"];
export type LoginResponse = Admin["schemas"]["LoginResponse"];
export type Flag = Admin["schemas"]["Flag"];
export type ConfigItem = Admin["schemas"]["ConfigItem"];
export type AuditEvent = Admin["schemas"]["AuditEvent"];
export type Role = Admin["schemas"]["Role"];
export type SystemStatus = Admin["schemas"]["SystemStatus"];
export type Readiness = Admin["schemas"]["Readiness"];
export type ClientConfig = Public["schemas"]["ClientConfig"];
export type Problem = Public["schemas"]["Problem"];
export type Me = Public["schemas"]["Me"];

// Phase 2 — users and moderation consoles.
export type AdminUser = Admin["schemas"]["AdminUser"];
export type AdminUserDetail = Admin["schemas"]["AdminUserDetail"];
export type AdminReport = Admin["schemas"]["AdminReport"];
export type AdminUsersSummary = AdminOps["adminUsersSummary"]["responses"][200]["content"]["application/json"];
export type AdminReportsSummary = AdminOps["adminReportsSummary"]["responses"][200]["content"]["application/json"];
