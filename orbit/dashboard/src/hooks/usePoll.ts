import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api/client";

// Poll every 2s; the API has no SSE stream (deviation: polling until the
// server grows an event stream). Stops when the component unmounts.
export function usePoll<T>(fn: () => Promise<T>, intervalMs = 2000) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const fnRef = useRef(fn);
  fnRef.current = fn;

  const refresh = useCallback(() => {
    fnRef.current().then(setData, setError);
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, intervalMs);
    return () => clearInterval(id);
  }, [refresh, intervalMs]);

  return { data, error, refresh };
}

export function useInstances() {
  return usePoll(api.listInstances);
}

export function useCapacity() {
  return usePoll(api.capacity);
}
