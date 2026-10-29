import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@vira/shared";
import { App } from "./App";
import { AuthProvider } from "./auth";
import { adminApi } from "./api";

const admin = { id: "a1", email: "ops@vira.ir", display_name: "مدیر آزمایشی", roles: ["super_admin"],
  permissions: ["system.read", "flags.read", "flags.write", "audit.read", "admins.read", "config.read"] };
const flags = [
  { key: "payments.real_money", description: "پرداخت واقعی", enabled: false, regulated: true, gate: "FG-1b", audiences: ["public"], version: 1 },
  { key: "app.maintenance_mode", description: "حالت نگهداری", enabled: false, regulated: false, audiences: ["public"], version: 3 },
];
const status = { service: "admin-api", version: "dev", environment: "development", time: "2026-10-29T09:00:00Z",
  readiness: { status: "ok", service: "admin-api", checks: [{ name: "postgres", status: "ok", duration_ms: 2 }] },
  regulated_capabilities: [{ key: "payments.real_money", gate: "FG-1b", state: "locked_off" }, { key: "usdt.conversion", gate: "G2", state: "locked_off" }] };

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
});
