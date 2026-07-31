import { useState } from 'react';
import type { YearFilterValue } from './YearFilter';
import './AssetList.css';

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
 * One column of the asset browser: a heading, its own search box, and the
 * (locally filtered) list of assets for a single group ("Stocks" or
 * "Others"). Two independent instances of this component are rendered by
 * `AssetBrowser`, each with its own search state, but both are handed
 * their slice of data from a single shared fetch (see `useAssetList`) so
 * the asset list itself is only ever requested once.
 */
export function AssetList({ label, year, items, selected, onSelect, loading, error }: AssetListProps) {
  const [query, setQuery] = useState('');

  const hasAssets = items.length > 0;
  const normalizedQuery = query.trim().toLowerCase();
  const filteredItems = normalizedQuery
    ? items.filter((asset) => asset.toLowerCase().includes(normalizedQuery))
    : items;

  return (
    <aside className="asset-list">
      <h2>{label}</h2>

      {loading && <p className="asset-list__hint">Loading assets…</p>}

      {error && (
        <div className="asset-list__error" role="alert">
          {error}
        </div>
      )}

      {!loading && !error && !hasAssets && (
        <p className="asset-list__hint">
          {year === 'all' ? 'No assets available.' : `No assets have data for ${year}.`}
        </p>
      )}

      {!loading && !error && hasAssets && (
        <input
          type="search"
          className="asset-list__search"
          placeholder={`Search ${label.toLowerCase()}…`}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={`Search ${label.toLowerCase()}`}
        />
      )}

      {!loading && !error && hasAssets && filteredItems.length === 0 && (
        <p className="asset-list__hint">No assets match &quot;{query.trim()}&quot;.</p>
      )}

      {!loading && !error && filteredItems.length > 0 && (
        <ul className="asset-list__items">
          {filteredItems.map((asset) => (
            <li key={asset}>
              <button
                type="button"
                className={
                  asset === selected
                    ? 'asset-list__item asset-list__item--active'
                    : 'asset-list__item'
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
