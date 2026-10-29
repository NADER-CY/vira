// Visual evidence for the Phase 1 web shells: builds must exist (npm run build). Serves each app with
// `vite preview`, stubs the backend with fixed responses (no real data) and captures light/dark screenshots.
// Usage: node e2e/screenshots.mjs <outDir>
import { spawn } from "node:child_process";
import { mkdirSync } from "node:fs";
import { chromium } from "playwright";

const out = process.argv[2] ?? "screenshots";
mkdirSync(out, { recursive: true });

const admin = { id: "01a1", email: "ops@vira.local", display_name: "مدیر نمونه", roles: ["super_admin"],
  permissions: ["system.read", "flags.read", "flags.write", "audit.read", "admins.read", "config.read"] };
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

async function shoot(browser, scheme, url, file, routes, init) {
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
  await page.screenshot({ path: `${out}/${file}-${scheme}.png`, fullPage: false });
  await ctx.close();
}

const adminSrv = await serve("admin", 4173);
const sellerSrv = await serve("seller", 4174);
const browser = await chromium.launch();
try {
  const logged = { "/v1/auth/me": [admin], "/v1/admin/system/status": [status], "/v1/admin/flags": [{ data: flags }], "/v1/admin/audit-events": [audit], "/v1/admin/roles": [roles] };
  for (const scheme of ["light", "dark"]) {
    await shoot(browser, scheme, "http://localhost:4173/login", "admin-login", { "/v1/auth/me": [{ code: "unauthenticated", status: 401, title: "x" }, 401] });
    await shoot(browser, scheme, "http://localhost:4173/", "admin-dashboard", logged);
    await shoot(browser, scheme, "http://localhost:4173/flags", "admin-flags", logged);
    await shoot(browser, scheme, "http://localhost:4173/audit", "admin-audit", logged);
    await shoot(browser, scheme, "http://localhost:4173/roles", "admin-roles", logged);
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
