import { useCallback, useEffect, useRef, useState } from "react";

/** Loads data with loading/error states and a reload function; ignores results of superseded calls. */
export function useAsync<T>(load: (signal: AbortSignal) => Promise<T>, deps: unknown[]) {
  const [state, setState] = useState<{ data?: T; error?: unknown; loading: boolean }>({ loading: true });
  const seq = useRef(0);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const run = useCallback(load, deps);

  const reload = useCallback(() => {
    const id = ++seq.current;
    const ctl = new AbortController();
    setState((s) => ({ ...s, loading: true, error: undefined }));
    run(ctl.signal).then(
      (data) => { if (id === seq.current) setState({ data, loading: false }); },
      (error) => { if (id === seq.current && !ctl.signal.aborted) setState({ error, loading: false }); },
    );
    return () => ctl.abort();
  }, [run]);

  useEffect(() => reload(), [reload]);
  return { ...state, reload };
}
