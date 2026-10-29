import { describe, expect, it, vi } from "vitest";
import { ApiError, createClient } from "./api";

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json", ...headers } });
}

describe("API client", () => {
  it("sends CSRF header on unsafe admin requests only, and credentials", async () => {
    const fetchImpl = vi.fn(async () => jsonResponse(200, { ok: true }));
    const c = createClient({ baseUrl: "http://x", csrf: true, fetchImpl: fetchImpl as unknown as typeof fetch });
    await c.get("/v1/a");
    await c.post("/v1/b", { a: 1 }, { idempotencyKey: "k1" });
    const [, getInit] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    const [, postInit] = fetchImpl.mock.calls[1] as unknown as [string, RequestInit];
    expect((getInit.headers as Record<string, string>)["X-VIRA-CSRF"]).toBeUndefined();
    expect((postInit.headers as Record<string, string>)["X-VIRA-CSRF"]).toBe("1");
    expect((postInit.headers as Record<string, string>)["Idempotency-Key"]).toBe("k1");
    expect(postInit.credentials).toBe("include");
  });

  it("parses RFC 9457 problems including extension members", async () => {
    const fetchImpl = vi.fn(async () => jsonResponse(409, {
      type: "urn:vira:error:feature_locked", title: "قفل", status: 409, code: "feature_locked", gate: "FG-1b", trace_id: "t1",
    }));
    const c = createClient({ baseUrl: "", fetchImpl: fetchImpl as unknown as typeof fetch });
    const err = await c.put("/v1/admin/flags/payments.real_money", {}).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe("feature_locked");
    expect(err.extra.gate).toBe("FG-1b");
    expect(err.traceId).toBe("t1");
  });

  it("maps network failures to network_error and 401 to the unauthenticated callback", async () => {
    const onUnauth = vi.fn();
    const failing = createClient({ baseUrl: "", fetchImpl: (async () => { throw new TypeError("down"); }) as unknown as typeof fetch });
    await expect(failing.get("/x")).rejects.toMatchObject({ code: "network_error", status: 0 });
    const unauth = createClient({ baseUrl: "", onUnauthenticated: onUnauth,
      fetchImpl: (async () => jsonResponse(401, { code: "unauthenticated", title: "t", status: 401 })) as unknown as typeof fetch });
    await expect(unauth.get("/x")).rejects.toMatchObject({ code: "unauthenticated" });
    expect(onUnauth).toHaveBeenCalledOnce();
  });

  it("returns the ETag for optimistic concurrency", async () => {
    const c = createClient({ baseUrl: "", fetchImpl: (async () => jsonResponse(200, {}, { ETag: '"3"' })) as unknown as typeof fetch });
    expect((await c.get("/x")).etag).toBe('"3"');
  });
});
