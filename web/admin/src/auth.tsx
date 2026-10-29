import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { ApiError, type AdminPrincipal, type ApiClient, type LoginResponse } from "@vira/shared";

type Status = "loading" | "authenticated" | "anonymous";

interface AuthCtx {
  status: Status;
  admin: AdminPrincipal | null;
  /** Password step; returns the short-lived MFA token for the TOTP step. */
  login: (email: string, password: string) => Promise<LoginResponse>;
  verifyMfa: (mfaToken: string, code: string) => Promise<void>;
  logout: () => Promise<void>;
  can: (permission: string) => boolean;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ api, children }: { api: ApiClient; children: ReactNode }) {
  const [status, setStatus] = useState<Status>("loading");
  const [admin, setAdmin] = useState<AdminPrincipal | null>(null);

  const refresh = useCallback(async () => {
    try {
      const { data } = await api.get<AdminPrincipal>("/v1/auth/me");
      setAdmin(data);
      setStatus("authenticated");
    } catch (e) {
      if (!(e instanceof ApiError) || e.status !== 401) console.warn("session check failed", e);
      setAdmin(null);
      setStatus("anonymous");
    }
  }, [api]);

  useEffect(() => { void refresh(); }, [refresh]);

  const value = useMemo<AuthCtx>(() => ({
    status,
    admin,
    login: async (email, password) => (await api.post<LoginResponse>("/v1/auth/login", { email, password })).data,
    verifyMfa: async (mfaToken, code) => {
      await api.post("/v1/auth/mfa", { mfa_token: mfaToken, code });
      await refresh();
    },
    logout: async () => {
      try { await api.post("/v1/auth/logout"); } finally { setAdmin(null); setStatus("anonymous"); }
    },
    can: (p) => !!admin?.permissions.includes(p),
  }), [api, admin, status, refresh]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAuth(): AuthCtx {
  const v = useContext(Ctx);
  if (!v) throw new Error("useAuth outside AuthProvider");
  return v;
}
