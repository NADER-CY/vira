import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@vira/shared";
import { App } from "./App";
import { SellerAuthProvider } from "./auth";

function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}
const fetchMock = vi.fn();

beforeEach(() => {
  sessionStorage.clear();
  fetchMock.mockReset();
  fetchMock.mockImplementation(async (url: string, init: RequestInit) => {
    const path = new URL(url).pathname;
    if (path === "/v1/config") return json(200, { api_version: "v1", flags: { "seller.web_portal": false }, regulated_locked: { "payments.real_money": "FG-1b" }, config: {} });
    if (path === "/v1/me") {
      const auth = (init.headers as Record<string, string>).Authorization;
      return auth === "Bearer good" ? json(200, { subject: "u-1", kind: "user", scopes: ["seller"] }) : json(401, { code: "unauthenticated", status: 401, title: "x" });
    }
    return json(404, { code: "not_found", status: 404, title: "x" });
  });
  vi.stubGlobal("fetch", fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

const renderAt = (path: string) => render(
  <ThemeProvider><MemoryRouter initialEntries={[path]}><SellerAuthProvider><App /></SellerAuthProvider></MemoryRouter></ThemeProvider>,
);

describe("Seller Admin shell", () => {
  it("anonymous visitors land on the login page, which explains the portal is not active yet", async () => {
    renderAt("/");
    expect(await screen.findByText("پنل فروشندگان ویرا")).toBeInTheDocument();
    expect(await screen.findByText("ورود فروشندگان از وب هنوز فعال نشده است")).toBeInTheDocument();
  });

  it("an invalid stored token is rejected server-side and cleared", async () => {
    sessionStorage.setItem("vira.seller.session", "bad");
    renderAt("/");
    expect(await screen.findByText("پنل فروشندگان ویرا")).toBeInTheDocument();
    expect(sessionStorage.getItem("vira.seller.session")).toBeNull();
  });

  it("a valid session (verified by GET /v1/me) shows the shell with gated finance sections", async () => {
    sessionStorage.setItem("vira.seller.session", "good");
    renderAt("/");
    expect(await screen.findByText("نمای کلی فروشگاه", { selector: "h1" })).toBeInTheDocument();
    expect(screen.getByText("تسویه و برداشت").closest("[aria-disabled]")).toHaveTextContent("FG-1b");
    expect(screen.getByText("u-1")).toBeInTheDocument();
  });
});
