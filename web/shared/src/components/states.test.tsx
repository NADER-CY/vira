import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ApiError } from "../lib/api";
import { ErrorState, FeatureDisabledState } from "./states";
import { ThemeProvider } from "../theme";
import { ThemeToggle } from "./ui";

describe("UI states", () => {
  it("error state shows the API title, trace id and retries", async () => {
    const retry = vi.fn();
    render(<ErrorState error={new ApiError(503, { code: "service_unavailable", title: "سرویس در دسترس نیست", trace_id: "abc" })} onRetry={retry} />);
    expect(screen.getByRole("alert")).toHaveTextContent("سرویس در دسترس نیست");
    expect(screen.getByText("abc")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /تلاش دوباره/ }));
    expect(retry).toHaveBeenCalledOnce();
  });

  it("locked feature names its gate", () => {
    render(<FeatureDisabledState title="کیف پول" phase="G1" locked />);
    expect(screen.getByText(/دروازهٔ G1/)).toBeInTheDocument();
  });

  it("theme toggle switches data-theme on <html>", async () => {
    render(<ThemeProvider storageKey="t"><ThemeToggle /></ThemeProvider>);
    const before = document.documentElement.dataset.theme;
    await userEvent.click(screen.getByRole("button"));
    expect(document.documentElement.dataset.theme).not.toBe(before);
    expect(["light", "dark"]).toContain(document.documentElement.dataset.theme);
  });
});
