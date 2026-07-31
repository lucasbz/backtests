import { useState } from 'react';
import type { YearFilterValue } from './YearFilter';
import './TickerList.css';

export interface TickerListProps {
  /** Heading for this column, e.g. "Stocks" or "Others". */
  label: string;
  /** Restricts context for the "no tickers" empty-state message. */
  year: YearFilterValue;
  /** The tickers for this group only - already fetched/split by the caller. */
  items: string[];
  selected: string | null;
  onSelect: (ticker: string) => void;
  loading: boolean;
  error: string | null;
}

/**
 * One column of the ticker browser: a heading, its own search box, and the
 * (locally filtered) list of tickers for a single group ("Stocks" or
 * "Others"). Two independent instances of this component are rendered by
 * `TickerBrowser`, each with its own search state, but both are handed
 * their slice of data from a single shared fetch (see `useTickerList`) so
 * the ticker list itself is only ever requested once.
 */
export function TickerList({ label, year, items, selected, onSelect, loading, error }: TickerListProps) {
  const [query, setQuery] = useState('');

  const hasTickers = items.length > 0;
  const normalizedQuery = query.trim().toLowerCase();
  const filteredItems = normalizedQuery
    ? items.filter((ticker) => ticker.toLowerCase().includes(normalizedQuery))
    : items;

  return (
    <aside className="ticker-list">
      <h2>{label}</h2>

      {loading && <p className="ticker-list__hint">Loading tickers…</p>}

      {error && (
        <div className="ticker-list__error" role="alert">
          {error}
        </div>
      )}

      {!loading && !error && !hasTickers && (
        <p className="ticker-list__hint">
          {year === 'all' ? 'No tickers available.' : `No tickers have data for ${year}.`}
        </p>
      )}

      {!loading && !error && hasTickers && (
        <input
          type="search"
          className="ticker-list__search"
          placeholder={`Search ${label.toLowerCase()}…`}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={`Search ${label.toLowerCase()}`}
        />
      )}

      {!loading && !error && hasTickers && filteredItems.length === 0 && (
        <p className="ticker-list__hint">No tickers match &quot;{query.trim()}&quot;.</p>
      )}

      {!loading && !error && filteredItems.length > 0 && (
        <ul className="ticker-list__items">
          {filteredItems.map((ticker) => (
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
