import { useEffect, useState } from 'react';
import { AssetList } from '../components/AssetList';
import { AssetInfoPanel } from '../components/AssetInfoPanel';
import { StrategyComparison } from '../components/StrategyComparison';
import type { YearFilterValue } from '../components/YearFilter';
import { useAssetInfo } from '../hooks/useAssetInfo';
import { useAssetList } from '../hooks/useAssetList';
import './AssetBrowser.css';

export interface AssetBrowserProps {
  /** Restricts the asset list to symbols with data for this year, or 'all'. */
  year: YearFilterValue;
  /**
   * Current starting balance (plain decimal string, e.g. "10000.00"), owned
   * by the header in `App` and threaded down to `StrategyComparison`.
   */
  balance: string;
}

/**
 * Browse the available assets, then inspect the selected one's imported
 * date range and run a backtest against it. This is the single entry point
 * into the app - selecting an asset drives both the info panel and the
 * backtest form below it.
 *
 * Laid out as three columns: the "Stocks" asset list on the left edge, the
 * detail panel in the middle, and the "Others" asset list on the right
 * edge. Both lists are handed their slice of a single shared fetch (see
 * `useAssetList`), but each renders as its own independent `AssetList`
 * instance with its own local search box/state.
 *
 * Each side column can be collapsed independently via its own toggle button
 * (placed on the inner edge, facing the detail panel), freeing up width for
 * the detail panel once an asset is selected. Collapsing only hides the
 * column visually - the underlying `AssetList` stays mounted so its search
 * text and selection state survive re-expanding.
 *
 * The asset's info (earliest/latest imported date) is fetched once here
 * and shared: it's shown directly in `AssetInfoPanel` and also used to
 * default the backtest form's start/end dates to the asset's full range.
 */
export function AssetBrowser({ year, balance }: AssetBrowserProps) {
  const [selectedAsset, setSelectedAsset] = useState<string | null>(null);
  const [stocksCollapsed, setStocksCollapsed] = useState(false);
  const [othersCollapsed, setOthersCollapsed] = useState(false);
  const { info, loading, error } = useAssetInfo(selectedAsset);
  const {
    stocks,
    others,
    loading: assetsLoading,
    error: assetsError,
  } = useAssetList(year);

  // Changing the year filter can drop the currently selected asset from
  // the list (it may not have data for that year), so clear the selection
  // rather than risk showing a backtest form for an asset no longer shown.
  useEffect(() => {
    setSelectedAsset(null);
  }, [year]);

  // When a specific year is selected, default the comparison form to that
  // year's range instead of the asset's full range - but clamped to the
  // asset's actual available data, so we never suggest a date that has no
  // data behind it. Dates are `YYYY-MM-DD` strings, which sort the same
  // lexicographically as chronologically, so plain string comparison works.
  let defaultStart = info?.earliest;
  let defaultEnd = info?.latest;
  if (year !== 'all') {
    const yearStart = `${year}-01-01`;
    const yearEnd = `${year}-12-31`;
    defaultStart = info?.earliest && info.earliest > yearStart ? info.earliest : yearStart;
    defaultEnd = info?.latest && info.latest < yearEnd ? info.latest : yearEnd;
  }

  return (
    <div className="asset-browser">
      <div
        className={
          stocksCollapsed
            ? 'asset-browser__side asset-browser__side--left asset-browser__side--collapsed'
            : 'asset-browser__side asset-browser__side--left'
        }
      >
        <div className="asset-browser__side-list">
          <AssetList
            label="Stocks"
            year={year}
            items={stocks}
            selected={selectedAsset}
            onSelect={setSelectedAsset}
            loading={assetsLoading}
            error={assetsError}
          />
        </div>
        <button
          type="button"
          className="asset-browser__toggle"
          onClick={() => setStocksCollapsed((collapsed) => !collapsed)}
          aria-pressed={stocksCollapsed}
          aria-label={stocksCollapsed ? 'Expand Stocks list' : 'Collapse Stocks list'}
        >
          {stocksCollapsed ? '›' : '‹'}
        </button>
      </div>

      <div className="asset-browser__detail">
        {selectedAsset ? (
          <>
            <AssetInfoPanel info={info} loading={loading} error={error} />
            <StrategyComparison
              asset={selectedAsset}
              defaultStart={defaultStart}
              defaultEnd={defaultEnd}
              balance={balance}
            />
          </>
        ) : (
          <p className="asset-browser__empty">Select an asset to get started.</p>
        )}
      </div>

      <div
        className={
          othersCollapsed
            ? 'asset-browser__side asset-browser__side--right asset-browser__side--collapsed'
            : 'asset-browser__side asset-browser__side--right'
        }
      >
        <button
          type="button"
          className="asset-browser__toggle"
          onClick={() => setOthersCollapsed((collapsed) => !collapsed)}
          aria-pressed={othersCollapsed}
          aria-label={othersCollapsed ? 'Expand Others list' : 'Collapse Others list'}
        >
          {othersCollapsed ? '‹' : '›'}
        </button>
        <div className="asset-browser__side-list">
          <AssetList
            label="Others"
            year={year}
            items={others}
            selected={selectedAsset}
            onSelect={setSelectedAsset}
            loading={assetsLoading}
            error={assetsError}
          />
        </div>
      </div>
    </div>
  );
}
