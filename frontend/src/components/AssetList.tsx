import { useState } from 'react';
import type { YearFilterValue } from './YearFilter';
import { errorBoxClasses, fieldClasses } from '../styles/ui';

export interface AssetListProps {
  /** Heading for this column, e.g. "Stocks" or "Others". */
  label: string;
  /** Restricts context for the "no assets" empty-state message. */
  year: YearFilterValue;
  /** The assets for this group only - already fetched/split by the caller. */
  items: string[];
  selected: string | null;
  onSelect: (asset: string) => void;
  loading: boolean;
  error: string | null;
}

/**
 * One group's worth of the asset browser: a heading, its own search box,
 * and the (locally filtered) list of assets for a single group ("Stocks"
 * or "Others"). Two independent instances of this component are rendered
 * as tabs by `AssetListPanel`, each with its own search state, but both
 * are handed their slice of data from a single shared fetch (see
 * `useAssetList`) so the asset list itself is only ever requested once.
 */
export function AssetList({ label, year, items, selected, onSelect, loading, error }: AssetListProps) {
  const [query, setQuery] = useState('');

  const hasAssets = items.length > 0;
  const normalizedQuery = query.trim().toLowerCase();
  const filteredItems = normalizedQuery
    ? items.filter((asset) => asset.toLowerCase().includes(normalizedQuery))
    : items;

  return (
    <aside className="w-[260px] flex-shrink-0 text-left">
      <h2 className="mb-3 text-xl font-semibold text-text-strong">{label}</h2>

      {loading && <p className="text-sm text-text">Loading assets…</p>}

      {error && (
        <div className={errorBoxClasses} role="alert">
          {error}
        </div>
      )}

      {!loading && !error && !hasAssets && (
        <p className="text-sm text-text">
          {year === 'all' ? 'No assets available.' : `No assets have data for ${year}.`}
        </p>
      )}

      {!loading && !error && hasAssets && (
        <input
          type="search"
          className={`${fieldClasses} mb-3`}
          placeholder={`Search ${label.toLowerCase()}…`}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={`Search ${label.toLowerCase()}`}
        />
      )}

      {!loading && !error && hasAssets && filteredItems.length === 0 && (
        <p className="text-sm text-text">No assets match &quot;{query.trim()}&quot;.</p>
      )}

      {!loading && !error && filteredItems.length > 0 && (
        <ul className="m-0 flex max-h-[75vh] list-none flex-col gap-1 overflow-y-auto p-0">
          {filteredItems.map((asset) => (
            <li key={asset}>
              <button
                type="button"
                className={
                  asset === selected
                    ? 'w-full rounded-xl border border-accent/50 bg-accent-soft px-3 py-2 text-left text-sm text-text-strong transition-all duration-150'
                    : 'w-full rounded-xl border border-transparent bg-surface px-3 py-2 text-left text-sm text-text transition-all duration-150 hover:translate-x-0.5 hover:border-border hover:text-text-strong'
                }
                onClick={() => onSelect(asset)}
              >
                {asset}
              </button>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
