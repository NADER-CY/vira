import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@vira/shared";
import { App } from "./App";
import { AuthProvider } from "./auth";
import { adminApi } from "./api";

const admin = { id: "a1", email: "ops@vira.ir", display_name: "مدیر آزمایشی", roles: ["super_admin"],
  permissions: ["system.read", "flags.read", "flags.write", "audit.read", "admins.read", "config.read", "users.read", "users.write", "reports.read", "reports.write"] };
const flags = [
  { key: "payments.real_money", description: "پرداخت واقعی", enabled: false, regulated: true, gate: "FG-1b", audiences: ["public"], version: 1 },
  { key: "app.maintenance_mode", description: "حالت نگهداری", enabled: false, regulated: false, audiences: ["public"], version: 3 },
];
const status = { service: "admin-api", version: "dev", environment: "development", time: "2026-10-29T09:00:00Z",
  readiness: { status: "ok", service: "admin-api", checks: [{ name: "postgres", status: "ok", duration_ms: 2 }] },
  regulated_capabilities: [{ key: "payments.real_money", gate: "FG-1b", state: "locked_off" }, { key: "usdt.conversion", gate: "G2", state: "locked_off" }] };

const user = { id: "0b7c8a52-5a0c-4d0e-9a51-3f5d3c3e1a01", handle: "sara_shop", display_name: "سارا رستمی", phone_masked: "09*****4567",
  status: "active", is_private: false, created_at: "2026-10-20T08:00:00Z", last_login_at: "2026-10-30T07:00:00Z",
  posts: 3, followers: 10, open_reports: 1, active_sessions: 1 };
const userDetail = { ...user, bio: "", intends_to_sell: false, following: 4, status_history: [],
  sessions: [{ id: "s1", platform: "android", model: "Pixel", created_at: "2026-10-29T08:00:00Z", last_used_at: "2026-10-30T07:00:00Z", revoked_at: null }] };
const report = { id: "6f1e2d3c-0000-4000-8000-000000000001", reporter_handle: "ali", target_type: "post", target_id: "p1",
  target_owner_handle: "sara_shop", target_preview: "فروش کالای تقلبی", reason: "counterfeit", state: "open",
  created_at: "2026-10-30T06:00:00Z", open_reports_on_target: 2 };

function json(statusCode: number, body: unknown) {
  return new Response(JSON.stringify(body), { status: statusCode, headers: { "Content-Type": "application/json" } });
}

let loggedIn = false;
const calls: { url: string; init: RequestInit }[] = [];

beforeEach(() => {
  loggedIn = false;
  calls.length = 0;
  vi.stubGlobal("fetch", vi.fn(async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const path = new URL(url).pathname;
    if (path === "/v1/auth/me") return loggedIn ? json(200, admin) : json(401, { code: "unauthenticated", title: "لطفاً دوباره وارد شوید.", status: 401 });
    if (path === "/v1/auth/login") return json(200, { mfa_required: true, mfa_token: "m1", mfa_expires_in: 300 });
    if (path === "/v1/auth/mfa") {
      const body = JSON.parse(String(init.body));
      if (body.code !== "123456") return json(401, { code: "mfa_invalid", title: "کد تأیید نادرست یا منقضی است.", status: 401 });
      loggedIn = true;
      return json(200, admin);
    }
    if (path === "/v1/admin/system/status") return json(200, status);
    if (path === "/v1/admin/flags") return json(200, { data: flags });
    if (path === "/v1/admin/users/summary") return json(200, { total: 12, active: 10, signups_today: 2, restricted: 1, suspended: 1, active_sessions: 7 });
    if (path === "/v1/admin/users") return json(200, { data: [user], next_cursor: null });
    if (path === `/v1/admin/users/${user.id}`) return json(200, userDetail);
    if (path === `/v1/admin/users/${user.id}/status`) return json(200, { id: user.id, previous_status: "active", status: "suspended" });
    if (path === "/v1/admin/reports/summary") return json(200, { open: 2, open_posts: 2, open_comments: 0, open_users: 0, actioned_today: 1, dismissed_today: 0, total: 3 });
    if (path === "/v1/admin/reports") return json(200, { data: [report], next_cursor: null });
    if (path === `/v1/admin/reports/${report.id}/resolve`) return json(200, { ...report, state: "actioned", resolution: "remove_content", resolution_note: "تقلبی" });
    return json(404, { code: "not_found", title: "یافت نشد", status: 404 });
  }));
});
afterEach(() => vi.unstubAllGlobals());

function renderAt(path: string) {
  return render(
    <ThemeProvider><MemoryRouter initialEntries={[path]}><AuthProvider api={adminApi}><App /></AuthProvider></MemoryRouter></ThemeProvider>,
  );
}

describe("Super Admin shell", () => {
  it("redirects anonymous users to the two-step login and signs in with password + TOTP", async () => {
    renderAt("/flags");
    expect(await screen.findByText("ورود به مدیریت ارشد ویرا")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("ایمیل سازمانی"), "ops@vira.ir");
    await userEvent.type(screen.getByLabelText("رمز عبور"), "correct horse battery");
    await userEvent.click(screen.getByRole("button", { name: "ادامه" }));

    const codeInput = await screen.findByLabelText("کد ۶ رقمی برنامهٔ احراز هویت");
    await userEvent.type(codeInput, "۱۱۱۱۱۱"); // Persian digits are normalised
    await userEvent.click(screen.getByRole("button", { name: "ورود" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("کد تأیید نادرست");

    await userEvent.clear(codeInput);
    await userEvent.type(codeInput, "123456");
    await userEvent.click(screen.getByRole("button", { name: "ورود" }));
    // Returns to the page originally requested.
    expect(await screen.findByText("فیچر فلگ‌ها و پیکربندی")).toBeInTheDocument();

    const login = calls.find((c) => c.url.endsWith("/v1/auth/login"))!;
    expect((login.init.headers as Record<string, string>)["X-VIRA-CSRF"]).toBe("1");
    expect(login.init.credentials).toBe("include");
  });

  it("shows regulated flags locked with their gate and their switch disabled", async () => {
    loggedIn = true;
    renderAt("/flags");
    const row = await screen.findByTestId("flag-payments.real_money");
    expect(within(row).getByText(/قفل تا FG-1b/)).toBeInTheDocument();
    expect(within(row).getByRole("switch")).toBeDisabled();
    expect(within(screen.getByTestId("flag-app.maintenance_mode")).getByRole("switch")).toBeEnabled();
  });

  it("dashboard lists locked regulated capabilities and dependency health", async () => {
    loggedIn = true;
    renderAt("/");
    expect(await screen.findByText("تبدیل USDT ↔ تومان")).toBeInTheDocument();
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.getAllByText("قفل").length).toBeGreaterThanOrEqual(2);
  });

  it("keeps later-phase and gated sections visible but inactive", async () => {
    loggedIn = true;
    renderAt("/");
    await screen.findByText("داشبورد مدیریت ارشد");
    const wallet = screen.getByText("کیف پول").closest("[aria-disabled]");
    expect(wallet).toHaveAttribute("aria-disabled", "true");
    expect(wallet).toHaveTextContent("G1");
    await waitFor(() => expect(document.documentElement.dataset.theme).toMatch(/light|dark/));
  });

  it("users console: KPI tiles, masked phones, search filter and a reasoned status change", async () => {
    loggedIn = true;
    renderAt("/users");
    expect(await screen.findByText("کنسول نظارت و راهبری کاربران")).toBeInTheDocument();
    expect(await screen.findByText("سارا رستمی")).toBeInTheDocument();
    expect(screen.getByText("09*****4567")).toBeInTheDocument();
    expect(screen.getByText("نرخ احراز هویت شاهکار")).toBeInTheDocument(); // stays visible, inactive until P4

    await userEvent.type(screen.getByLabelText("نام کاربری، شناسه یا شماره همراه"), "sara");
    await userEvent.click(screen.getByRole("button", { name: "جستجو" }));
    await waitFor(() => expect(calls.some((c) => c.url.includes("/v1/admin/users?") && c.url.includes("q=sara"))).toBe(true));

    await userEvent.click(screen.getByRole("button", { name: "مشاهدهٔ sara_shop" }));
    expect(await screen.findByText("میز بررسی حساب")).toBeInTheDocument();
    const submit = screen.getByRole("button", { name: "ثبت تغییر وضعیت" });
    await userEvent.selectOptions(screen.getByLabelText("وضعیت جدید"), "suspended");
    expect(submit).toBeDisabled(); // a reason is mandatory
    await userEvent.type(screen.getByLabelText(/دلیل/), "کلاهبرداری تأییدشده");
    await userEvent.click(submit);
    expect(await screen.findByText(/وضعیت حساب به «مسدود» تغییر کرد/)).toBeInTheDocument();
    const post = calls.find((c) => c.url.endsWith(`/v1/admin/users/${user.id}/status`))!;
    expect(JSON.parse(String(post.init.body))).toEqual({ status: "suspended", reason: "کلاهبرداری تأییدشده" });
    expect((post.init.headers as Record<string, string>)["X-VIRA-CSRF"]).toBe("1");
  });

  it("moderation console: queue, inspection desk and a verdict with a mandatory note", async () => {
    loggedIn = true;
    renderAt("/moderation");
    expect(await screen.findByText("مرکز ممیزی نظرات، تعاملات و رسیدگی به تخلفات")).toBeInTheDocument();
    expect(await screen.findByText("فروش کالای تقلبی")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "بررسی" }));
    const verdict = await screen.findByRole("button", { name: "تأیید گزارش و حذف محتوا" });
    expect(verdict).toBeDisabled();
    await userEvent.type(screen.getByLabelText("یادداشت محرمانه ممیز (الزامی)"), "تقلبی");
    await userEvent.click(verdict);
    expect(await screen.findByText(/این پرونده بسته شده است/)).toBeInTheDocument();
    const post = calls.find((c) => c.url.endsWith(`/v1/admin/reports/${report.id}/resolve`))!;
    expect(JSON.parse(String(post.init.body))).toEqual({ resolution: "remove_content", note: "تقلبی" });
  });
});
