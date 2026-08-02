import { useEffect, useState } from 'react';
import { ApiError, getCandles, type Candle } from '../api/client';

export interface UseCandlesResult {
  candles: Candle[] | null;
  loading: boolean;
  error: string | null;
}

/**
 * Fetches daily candles for `asset` between `start` and `end` (via
 * `GET /api/candles`) whenever any of them change. `asset` may be `null`
 * (nothing selected yet), and `start`/`end` may be omitted (e.g. the asset's
 * info hasn't loaded yet) - in either case no request is made, mirroring
 * `useAssetInfo`.
 */
export function useCandles(asset: string | null, start?: string, end?: string): UseCandlesResult {
  const [candles, setCandles] = useState<Candle[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!asset || !start || !end) {
      setCandles(null);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);
    setCandles(null);

    getCandles(asset, start, end)
      .then((result) => {
        if (cancelled) return;
        setCandles(result);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : 'Unexpected error loading candles.');
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [asset, start, end]);

  return { candles, loading, error };
}
