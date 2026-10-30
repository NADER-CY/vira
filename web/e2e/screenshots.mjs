// Visual evidence for the Phase 1 web shells: builds must exist (npm run build). Serves each app with
// `vite preview`, stubs the backend with fixed responses (no real data) and captures light/dark screenshots.
// Usage: node e2e/screenshots.mjs <outDir>
import { spawn } from "node:child_process";
import { mkdirSync } from "node:fs";
import { chromium } from "playwright";

const out = process.argv[2] ?? "screenshots";
mkdirSync(out, { recursive: true });

const admin = { id: "01a1", email: "ops@vira.local", display_name: "مدیر نمونه", roles: ["super_admin"],
  permissions: ["system.read", "flags.read", "flags.write", "audit.read", "admins.read", "config.read", "users.read", "users.write", "reports.read", "reports.write"] };
const status = { service: "admin-api", version: "p1-dev", environment: "development", time: new Date().toISOString(),
  readiness: { status: "ok", service: "admin-api", checks: [
    { name: "postgres", status: "ok", duration_ms: 2 }, { name: "valkey", status: "ok", duration_ms: 1 }, { name: "object_storage", status: "ok", duration_ms: 4 }] },
  regulated_capabilities: [
    ["payments.real_money", "FG-1b"], ["settlement.live", "FG-1b"], ["payouts.withdrawals", "FG-1b"], ["wallet.topup", "G1"],
    ["usdt.deposits", "G2"], ["usdt.withdrawals", "G2"], ["usdt.conversion", "G2"], ["credit.bnpl", "G3"]].map(([key, gate]) => ({ key, gate, state: "locked_off" })) };
const flags = [
  ...status.regulated_capabilities.map((c) => ({ key: c.key, description: "قابلیت مالی قانون‌مند", enabled: false, regulated: true, gate: c.gate, audiences: ["public"], version: 1, updated_at: new Date(Date.now() - 86400e3).toISOString() })),
  { key: "app.maintenance_mode", description: "پاسخ ۵۰۳ نگهداری در API عمومی", enabled: false, regulated: false, audiences: ["public"], version: 1, updated_at: new Date(Date.now() - 3600e3).toISOString() },
  { key: "seller.web_portal", description: "ورود فروشندگان به پنل وب (فاز ۴)", enabled: false, regulated: false, audiences: ["seller"], version: 1, updated_at: new Date(Date.now() - 7200e3).toISOString() },
];
const audit = { data: [
  { id: "e1", created_at: new Date(Date.now() - 60e3).toISOString(), actor_type: "admin", actor_id: "01a1", action: "flags.update", target_type: "feature_flag", target_id: "payments.real_money", outcome: "denied", ip: "127.0.0.1" },
  { id: "e2", created_at: new Date(Date.now() - 120e3).toISOString(), actor_type: "admin", actor_id: "01a1", action: "admin.login", outcome: "success", ip: "127.0.0.1" },
  { id: "e3", created_at: new Date(Date.now() - 180e3).toISOString(), actor_type: "anonymous", action: "admin.login", outcome: "failure", ip: "10.0.0.7" },
], next_cursor: null };
const roles = { data: [
  { key: "super_admin", name_fa: "مدیر ارشد", description: "همهٔ دسترسی‌ها", permissions: admin.permissions.concat(["admins.write"]) },
  { key: "auditor", name_fa: "حسابرس", description: "فقط خواندن", permissions: ["audit.read", "flags.read", "system.read"] },
  { key: "operator", name_fa: "اپراتور", description: "عملیات روزمره", permissions: ["system.read", "flags.read"] },
] };

// Phase 2 consoles — synthetic, clearly fictional accounts (no real data in CI artefacts).
const ago = (ms) => new Date(Date.now() - ms).toISOString();
const people = [
  ["نمونه کاربر یک", "sample_user_1", "active", 0], ["نمونه فروشگاه دو", "sample_shop_2", "active", 2], ["کاربر آزمایشی سه", "test_user_3", "restricted", 1],
  ["حساب نمایشی چهار", "demo_account_4", "suspended", 3], ["کاربر نمونه پنج", "sample_user_5", "active", 0],
].map(([display_name, handle, st, open], i) => ({ id: `00000000-0000-4000-8000-00000000000${i + 1}`, handle, display_name,
  phone_masked: `09*****${String(1000 + i * 1111).slice(-4)}`, status: st, is_private: i % 2 === 1, created_at: ago((i + 3) * 86400e3 * 9),
  last_login_at: ago((i + 1) * 3600e3 * 5), posts: 4 + i, followers: 10 * (i + 1), open_reports: open, active_sessions: st === "suspended" ? 0 : 1 }));
const usersSummary = { total: 1284, active: 1250, signups_today: 42, restricted: 23, suspended: 11, active_sessions: 977 };
const userDetail = { ...people[1], bio: "حساب نمایشی برای تصاویر مستند", intends_to_sell: true, following: 31,
  status_history: [{ from: "restricted", to: "active", reason: "رفع محدودیت پس از بررسی", actor_type: "admin", created_at: ago(5 * 86400e3) }],
  sessions: [{ id: "s1", platform: "android", model: "Pixel 8", created_at: ago(2 * 86400e3), last_used_at: ago(3600e3), revoked_at: null },
    { id: "s2", platform: "ios", model: "iPhone 15", created_at: ago(9 * 86400e3), last_used_at: ago(4 * 86400e3), revoked_at: ago(4 * 86400e3), revoke_reason: "logout" }] };
const reportRows = [
  ["post", "counterfeit", "فروش کالای بدون ضمانت اصالت (متن نمایشی)", "open", 3], ["comment", "harassment", "نظر توهین‌آمیز نمایشی", "open", 1],
  ["user", "scam", "", "open", 2], ["post", "spam", "تبلیغ تکراری نمایشی", "actioned", 0], ["comment", "hate", "متن نمایشی گزارش‌شده", "dismissed", 0],
].map(([target_type, reason, target_preview, st, n], i) => ({ id: `10000000-0000-4000-8000-00000000000${i + 1}`, reporter_handle: `reporter_${i + 1}`,
  target_type, target_id: `20000000-0000-4000-8000-00000000000${i + 1}`, target_owner_handle: people[i].handle, target_preview, reason,
  details: i === 0 ? "توضیح نمایشی شاکی" : "", state: st, created_at: ago((i + 1) * 3600e3 * 2), open_reports_on_target: n }));
const reportsSummary = { open: 38, open_posts: 21, open_comments: 12, open_users: 5, actioned_today: 14, dismissed_today: 6, total: 514 };

function serve(dir, port) {
  // Own process group so the whole preview server can be stopped (npx would leave vite running).
  const p = spawn(process.execPath, ["../node_modules/vite/bin/vite.js", "preview", "--port", String(port), "--strictPort"],
    { cwd: dir, stdio: ["ignore", "pipe", "inherit"], detached: true });
  return new Promise((resolve, reject) => {
    p.stdout.on("data", (d) => { if (String(d).includes(String(port))) resolve(p); });
    p.on("exit", (c) => reject(new Error(`preview ${dir} exited ${c}`)));
    setTimeout(() => resolve(p), 8000);
  });
}

const json = (body, status = 200) => ({ status, contentType: "application/json", body: JSON.stringify(body) });

async function shoot(browser, scheme, url, file, routes, init, act) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: scheme, locale: "fa-IR", timezoneId: "Asia/Tehran" });
  if (init) await ctx.addInitScript(init);
  await ctx.route(/localhost:808[01]\//, (route) => {
    const path = new URL(route.request().url()).pathname;
    const r = routes[path];
    return route.fulfill(r ? json(...r) : json({ code: "not_found", status: 404, title: "یافت نشد" }, 404));
  });
  const page = await ctx.newPage();
  await page.goto(url, { waitUntil: "load", timeout: 30_000 });
  await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
  await page.waitForTimeout(400);
  if (act) {
    await act(page);
    await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
    await page.waitForTimeout(300);
  }
  await page.screenshot({ path: `${out}/${file}-${scheme}.png`, fullPage: false });
  await ctx.close();
}

const adminSrv = await serve("admin", 4173);
const sellerSrv = await serve("seller", 4174);
const browser = await chromium.launch();
try {
  const logged = { "/v1/auth/me": [admin], "/v1/admin/system/status": [status], "/v1/admin/flags": [{ data: flags }], "/v1/admin/audit-events": [audit], "/v1/admin/roles": [roles],
    "/v1/admin/users/summary": [usersSummary], "/v1/admin/users": [{ data: people, next_cursor: "next" }], [`/v1/admin/users/${people[1].id}`]: [userDetail],
    "/v1/admin/reports/summary": [reportsSummary], "/v1/admin/reports": [{ data: reportRows.filter((r) => r.state === "open"), next_cursor: null }] };
  for (const scheme of ["light", "dark"]) {
    await shoot(browser, scheme, "http://localhost:4173/login", "admin-login", { "/v1/auth/me": [{ code: "unauthenticated", status: 401, title: "x" }, 401] });
    await shoot(browser, scheme, "http://localhost:4173/", "admin-dashboard", logged);
    await shoot(browser, scheme, "http://localhost:4173/flags", "admin-flags", logged);
    await shoot(browser, scheme, "http://localhost:4173/audit", "admin-audit", logged);
    await shoot(browser, scheme, "http://localhost:4173/roles", "admin-roles", logged);
    await shoot(browser, scheme, "http://localhost:4173/users", "admin-users", logged);
    await shoot(browser, scheme, "http://localhost:4173/users", "admin-user-desk", logged, null,
      (page) => page.getByRole("button", { name: `مشاهدهٔ ${people[1].handle}` }).click());
    await shoot(browser, scheme, "http://localhost:4173/moderation", "admin-moderation", logged, null,
      (page) => page.getByRole("button", { name: "بررسی" }).first().click());
    const cfg = { "/v1/config": [{ api_version: "v1", flags: { "seller.web_portal": false }, regulated_locked: {}, config: {} }] };
    await shoot(browser, scheme, "http://localhost:4174/login", "seller-login", cfg);
    await shoot(browser, scheme, "http://localhost:4174/", "seller-shell", { ...cfg, "/v1/me": [{ subject: "seller-demo", kind: "user", scopes: ["seller"] }] },
      () => sessionStorage.setItem("vira.seller.session", "demo-token-for-screenshots"));
  }
  console.log("screenshots written to", out);
} finally {
  await browser.close();
  for (const srv of [adminSrv, sellerSrv]) {
    try { process.kill(-srv.pid, "SIGTERM"); } catch { /* already stopped */ }
  }
}
process.exit(0);
