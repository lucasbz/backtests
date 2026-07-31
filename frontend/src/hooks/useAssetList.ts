import { useEffect, useState } from 'react';
import { ApiError, getAssets, type AssetsResponse } from '../api/client';
import type { YearFilterValue } from '../components/YearFilter';

export interface UseAssetListResult {
  stocks: string[];
  others: string[];
  loading: boolean;
  error: string | null;
}

const EMPTY_ASSETS: AssetsResponse = { stocks: [], others: [] };

/**
 * Fetches the available assets (via `GET /api/assets`) for `year`,
 * splitting them into `stocks` and `others` as returned by the API. This is
 * the single fetch shared by the "Stocks" and "Others" columns in
 * `AssetBrowser` - each column does its own local search-filtering over
 * the slice it's handed, but neither calls the API itself.
 */
export function useAssetList(year: YearFilterValue): UseAssetListResult {
  const [assets, setAssets] = useState<AssetsResponse>(EMPTY_ASSETS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    getAssets(year === 'all' ? undefined : year)
      .then((result) => {
        if (cancelled) return;
        setAssets(result);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : 'Could not load assets.');
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [year]);

  return {
    stocks: assets.stocks,
    others: assets.others,
    loading,
    error,
  };
}
