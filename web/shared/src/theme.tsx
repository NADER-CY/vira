import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

export type ThemeMode = "light" | "dark" | "system";
type Resolved = "light" | "dark";

interface ThemeCtx { mode: ThemeMode; resolved: Resolved; setMode: (m: ThemeMode) => void; toggle: () => void }

const Ctx = createContext<ThemeCtx | null>(null);

function readStored(key: string): ThemeMode {
  try {
    const v = localStorage.getItem(key);
    return v === "light" || v === "dark" || v === "system" ? v : "system";
  } catch {
    return "system";
  }
}

function systemPrefersDark(): boolean {
  return typeof window !== "undefined" && typeof window.matchMedia === "function" && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

/** Light/dark theme (doc 09: both themes required). Persists the choice per browser; defaults to the OS setting. */
export function ThemeProvider({ children, storageKey = "vira.theme" }: { children: ReactNode; storageKey?: string }) {
  const [mode, setModeState] = useState<ThemeMode>(() => readStored(storageKey));
  const [sysDark, setSysDark] = useState(systemPrefersDark);

  useEffect(() => {
    if (typeof window.matchMedia !== "function") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const on = (e: MediaQueryListEvent) => setSysDark(e.matches);
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, []);

  const resolved: Resolved = mode === "system" ? (sysDark ? "dark" : "light") : mode;

  useEffect(() => {
    document.documentElement.dataset.theme = resolved;
  }, [resolved]);

  const setMode = useCallback((m: ThemeMode) => {
    setModeState(m);
    try { localStorage.setItem(storageKey, m); } catch { /* storage unavailable: keep in memory */ }
  }, [storageKey]);

  const value = useMemo<ThemeCtx>(() => ({
    mode, resolved, setMode, toggle: () => setMode(resolved === "dark" ? "light" : "dark"),
  }), [mode, resolved, setMode]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useTheme(): ThemeCtx {
  const v = useContext(Ctx);
  if (!v) throw new Error("useTheme outside ThemeProvider");
  return v;
}
