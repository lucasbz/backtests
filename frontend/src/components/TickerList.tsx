import { useEffect, useState } from 'react';
import { ApiError, getTickers } from '../api/client';
import type { YearFilterValue } from './YearFilter';
import './TickerList.css';

export interface TickerListProps {
  /** Restricts the list to tickers with data for this year, or 'all'. */
  year: YearFilterValue;
  selected: string | null;
  onSelect: (ticker: string) => void;
}

export function TickerList({ year, selected, onSelect }: TickerListProps) {
  const [tickers, setTickers] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    getTickers(year === 'all' ? undefined : year)
      .then((list) => {
        if (cancelled) return;
        setTickers(list);
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

  return (
    <aside className="ticker-list">
      <h2>Tickers</h2>

      {loading && <p className="ticker-list__hint">Loading tickers…</p>}

      {error && (
        <div className="ticker-list__error" role="alert">
          {error}
        </div>
      )}

      {!loading && !error && tickers.length === 0 && (
        <p className="ticker-list__hint">
          {year === 'all' ? 'No tickers available.' : `No tickers have data for ${year}.`}
        </p>
      )}

      {!loading && !error && tickers.length > 0 && (
        <ul className="ticker-list__items">
          {tickers.map((ticker) => (
            <li key={ticker}>
              <button
                type="button"
                className={
                  ticker === selected
                    ? 'ticker-list__item ticker-list__item--active'
                    : 'ticker-list__item'
                }
                onClick={() => onSelect(ticker)}
              >
                {ticker}
              </button>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
