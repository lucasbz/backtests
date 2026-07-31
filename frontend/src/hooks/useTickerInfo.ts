import { useEffect, useState } from 'react';
import { ApiError, getTickerInfo, type TickerInfo } from '../api/client';

export interface UseTickerInfoResult {
  info: TickerInfo | null;
  loading: boolean;
  error: string | null;
}

/**
 * Fetches the imported date range for `ticker` (via `GET /api/info`)
 * whenever it changes. `ticker` may be `null` to represent "nothing
 * selected yet", in which case no request is made.
 */
export function useTickerInfo(ticker: string | null): UseTickerInfoResult {
  const [info, setInfo] = useState<TickerInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!ticker) {
      setInfo(null);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);
    setInfo(null);

    getTickerInfo(ticker)
      .then((result) => {
        if (cancelled) return;
        setInfo(result);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : 'Unexpected error looking up ticker.');
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [ticker]);

  return { info, loading, error };
}
