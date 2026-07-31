import { useEffect, useState } from 'react';
import { ApiError, getAssetInfo, type AssetInfo } from '../api/client';

export interface UseAssetInfoResult {
  info: AssetInfo | null;
  loading: boolean;
  error: string | null;
}

/**
 * Fetches the imported date range for `asset` (via `GET /api/info`)
 * whenever it changes. `asset` may be `null` to represent "nothing
 * selected yet", in which case no request is made.
 */
export function useAssetInfo(asset: string | null): UseAssetInfoResult {
  const [info, setInfo] = useState<AssetInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!asset) {
      setInfo(null);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);
    setInfo(null);

    getAssetInfo(asset)
      .then((result) => {
        if (cancelled) return;
        setInfo(result);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : 'Unexpected error looking up asset.');
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [asset]);

  return { info, loading, error };
}
