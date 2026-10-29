import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router";
import { LoadingState } from "./states";

/** Route guard: shows a loading state while the session is resolved, redirects to login when absent. */
export function ProtectedRoute({ status, loginPath = "/login", children }: {
  status: "loading" | "authenticated" | "anonymous"; loginPath?: string; children: ReactNode;
}) {
  const loc = useLocation();
  if (status === "loading") return <div className="v-auth"><div className="v-auth__card"><LoadingState rows={3} /></div></div>;
  if (status === "anonymous") return <Navigate to={loginPath} replace state={{ from: loc.pathname }} />;
  return <>{children}</>;
}
