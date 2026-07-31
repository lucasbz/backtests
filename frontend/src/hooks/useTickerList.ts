import { useEffect, useState } from 'react';
import { ApiError, getTickers, type TickersResponse } from '../api/client';
import type { YearFilterValue } from '../components/YearFilter';

export interface UseTickerListResult {
  stocks: string[];
  others: string[];
  loading: boolean;
  error: string | null;
}

const EMPTY_TICKERS: TickersResponse = { stocks: [], others: [] };

/**
 * Fetches the available tickers (via `GET /api/tickers`) for `year`,
 * splitting them into `stocks` and `others` as returned by the API. This is
 * the single fetch shared by the "Stocks" and "Others" columns in
 * `TickerBrowser` - each column does its own local search-filtering over
 * the slice it's handed, but neither calls the API itself.
 */
export function useTickerList(year: YearFilterValue): UseTickerListResult {
  const [tickers, setTickers] = useState<TickersResponse>(EMPTY_TICKERS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    getTickers(year === 'all' ? undefined : year)
      .then((result) => {
        if (cancelled) return;
        setTickers(result);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : 'Could not load tickers.');
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
    stocks: tickers.stocks,
    others: tickers.others,
    loading,
    error,
  };
}
