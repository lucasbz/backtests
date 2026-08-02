import { useEffect, useState } from 'react';
import { ApiError, getIndicator, type IndicatorPoint } from '../api/client';

export interface IndicatorSelection {
  type: 'sma' | 'ema';
  period: number;
}

export interface IndicatorSeriesResult extends IndicatorSelection {
  points: IndicatorPoint[];
}

export interface UseIndicatorsResult {
  series: IndicatorSeriesResult[];
  loading: boolean;
  error: string | null;
}

/**
 * Fetches one `GET /api/indicators/{type}` series per entry in `selected`
 * (in parallel), for `asset` over `[start, end]`. `asset`/`start`/`end` may
 * be missing (nothing selected/loaded yet) and `selected` may be empty - in
 * either case no request is made, mirroring `useCandles`/`useAssetInfo`.
 *
 * A failure on any single indicator request surfaces as an overall `error`
 * (rather than a partial-success result), matching how every other fetch
 * hook in this app handles failure.
 */
export function useIndicators(
  asset: string | null,
  start: string | undefined,
  end: string | undefined,
  selected: IndicatorSelection[],
): UseIndicatorsResult {
  const [series, setSeries] = useState<IndicatorSeriesResult[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // A stable string key so the effect only re-runs when the actual set of
  // selected type/period pairs changes, not whenever the caller passes a
  // new array/object identity for an equivalent selection.
  const selectionKey = selected.map((entry) => `${entry.type}:${entry.period}`).join(',');

  useEffect(() => {
    if (!asset || !start || !end || selected.length === 0) {
      setSeries([]);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);
    setSeries([]);

    Promise.all(selected.map((entry) => getIndicator(entry.type, asset, start, end, entry.period)))
      .then((results) => {
        if (cancelled) return;
        setSeries(selected.map((entry, index) => ({ ...entry, points: results[index] })));
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : 'Unexpected error loading indicators.');
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
    // `selected` is intentionally represented by `selectionKey` here (see
    // above) rather than listed directly.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [asset, start, end, selectionKey]);

  return { series, loading, error };
}
