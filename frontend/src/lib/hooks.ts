import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

/** Runs `load` whenever `deps` change; answers of outdated calls are ignored. */
export function useLoad<T>(load: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T>();
  const [error, setError] = useState<string>();
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let current = true;
    setError(undefined);
    load().then(
      (d) => current && setData(d),
      (e: Error) => current && setError(e.message),
    );
    return () => {
      current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);

  return { data, error, reload: () => setTick((t) => t + 1) };
}

/** Replaces the URL with the canonical one, e.g. after a title change made the slug stale. */
export function useCanonicalPath(path: string | undefined) {
  const location = useLocation();
  const navigate = useNavigate();
  useEffect(() => {
    if (path && location.pathname !== path) navigate(path + location.search, { replace: true });
  }, [path, location.pathname, location.search, navigate]);
}
