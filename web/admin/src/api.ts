import { createClient } from "@vira/shared";

/** Admin realm client: session cookie (HttpOnly, SameSite=Strict) + CSRF header; no tokens in JS (ADR-016). */
export const adminApi = createClient({
  baseUrl: import.meta.env.VITE_ADMIN_API_BASE_URL ?? "http://localhost:8081",
  csrf: true,
});
