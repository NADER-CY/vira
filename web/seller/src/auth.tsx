import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { ApiError, type Me } from "@vira/shared";
import { publicApi, session } from "./api";

type Status = "loading" | "authenticated" | "anonymous";
export type Principal = Me;

interface Ctx { status: Status; principal: Principal | null; signOut: () => void }
const AuthCtx = createContext<Ctx | null>(null);

/** Resolves the seller session by validating the stored access token with GET /v1/me (server-side check). */
export function SellerAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status>("loading");
  const [principal, setPrincipal] = useState<Principal | null>(null);

  const resolve = useCallback(async () => {
    if (!session.get()) { setStatus("anonymous"); return; }
    try {
      const { data } = await publicApi.get<Principal>("/v1/me");
      setPrincipal(data);
      setStatus("authenticated");
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) session.clear();
      setPrincipal(null);
      setStatus("anonymous");
    }
  }, []);

  useEffect(() => { void resolve(); }, [resolve]);

  const value = useMemo<Ctx>(() => ({
    status, principal, signOut: () => { session.clear(); setPrincipal(null); setStatus("anonymous"); },
  }), [status, principal]);
  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>;
}

export function useSellerAuth(): Ctx {
  const v = useContext(AuthCtx);
  if (!v) throw new Error("useSellerAuth outside SellerAuthProvider");
  return v;
}
