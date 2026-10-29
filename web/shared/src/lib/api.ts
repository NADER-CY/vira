// Minimal typed HTTP client for the VIRA APIs: RFC 9457 problem+json errors, cookies, CSRF header for the
// admin realm, idempotency keys, timeouts. Types for request/response bodies come from the generated
// OpenAPI definitions in ../api (npm run gen:api).
import { messageFor } from "./messages";

export interface FieldError { field: string; code: string; detail?: string }

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly title: string;
  readonly detail?: string;
  readonly fields: FieldError[];
  readonly traceId?: string;
  readonly extra: Record<string, unknown>;

  constructor(status: number, body: Record<string, unknown>) {
    const code = typeof body.code === "string" ? body.code : status === 0 ? "network_error" : "internal_error";
    const title = typeof body.title === "string" && body.title ? body.title : messageFor(code);
    super(title);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.title = title;
    this.detail = typeof body.detail === "string" ? body.detail : undefined;
    this.fields = Array.isArray(body.errors) ? (body.errors as FieldError[]) : [];
    this.traceId = typeof body.trace_id === "string" ? body.trace_id : undefined;
    const { type: _t, title: _ti, status: _s, code: _c, detail: _d, errors: _e, trace_id: _tr, instance: _i, ...extra } = body;
    this.extra = extra;
  }
}

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  /** Sent as Idempotency-Key (doc 07 §1.3) on unsafe requests that must not be applied twice. */
  idempotencyKey?: string;
  timeoutMs?: number;
}

export interface ApiClientConfig {
  baseUrl: string;
  /** Admin realm: send X-VIRA-CSRF: 1 on state-changing requests (forces a CORS preflight). */
  csrf?: boolean;
  /** Bearer token provider (public realm). */
  token?: () => string | undefined;
  onUnauthenticated?: () => void;
  fetchImpl?: typeof fetch;
}

export interface ApiResponse<T> { data: T; status: number; etag?: string }

export function createClient(cfg: ApiClientConfig) {
  const doFetch = cfg.fetchImpl ?? ((...a: Parameters<typeof fetch>) => fetch(...a));

  async function request<T>(path: string, opts: RequestOptions = {}): Promise<ApiResponse<T>> {
    const method = opts.method ?? "GET";
    const headers: Record<string, string> = { Accept: "application/json", "Accept-Language": "fa-IR", ...opts.headers };
    if (opts.body !== undefined) headers["Content-Type"] = "application/json";
    if (cfg.csrf && method !== "GET") headers["X-VIRA-CSRF"] = "1";
    if (opts.idempotencyKey) headers["Idempotency-Key"] = opts.idempotencyKey;
    const tok = cfg.token?.();
    if (tok) headers.Authorization = `Bearer ${tok}`;

    const timeout = AbortSignal.timeout(opts.timeoutMs ?? 15_000);
    const signal = opts.signal ? AbortSignal.any([opts.signal, timeout]) : timeout;
    let res: Response;
    try {
      res = await doFetch(cfg.baseUrl + path, {
        method, headers, signal, credentials: "include",
        body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
      });
    } catch (e) {
      if (opts.signal?.aborted) throw e;
      const timedOut = timeout.aborted;
      throw new ApiError(0, { code: timedOut ? "timeout" : "network_error" });
    }
    const text = await res.text();
    let body: unknown = undefined;
    if (text) {
      try { body = JSON.parse(text); } catch { body = { detail: text }; }
    }
    if (!res.ok) {
      const err = new ApiError(res.status, (body as Record<string, unknown>) ?? {});
      if (res.status === 401) cfg.onUnauthenticated?.();
      throw err;
    }
    return { data: body as T, status: res.status, etag: res.headers.get("ETag") ?? undefined };
  }

  return {
    request,
    get: <T>(path: string, o?: Omit<RequestOptions, "method" | "body">) => request<T>(path, { ...o, method: "GET" }),
    post: <T>(path: string, body?: unknown, o?: Omit<RequestOptions, "method" | "body">) => request<T>(path, { ...o, method: "POST", body }),
    put: <T>(path: string, body?: unknown, o?: Omit<RequestOptions, "method" | "body">) => request<T>(path, { ...o, method: "PUT", body }),
  };
}

export type ApiClient = ReturnType<typeof createClient>;

/** A random idempotency key (UUID v4). */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}
