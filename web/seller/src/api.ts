import { createClient } from "@vira/shared";

const SESSION_KEY = "vira.seller.session";

/** Seller web uses the public API with the same user identity as the mobile app (doc 04 §10). The access token
 * lives in sessionStorage for this tab only; the sign-in flow (OTP) and store selection arrive in Phase 4. */
export const session = {
  get(): string | undefined {
    try { return sessionStorage.getItem(SESSION_KEY) ?? undefined; } catch { return undefined; }
  },
  clear(): void {
    try { sessionStorage.removeItem(SESSION_KEY); } catch { /* storage unavailable */ }
  },
};

export const publicApi = createClient({
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080",
  token: () => session.get(),
});
