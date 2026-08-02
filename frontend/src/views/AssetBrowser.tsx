import { useEffect, useState } from 'react';
import { AssetListPanel } from '../components/AssetListPanel';
import { AssetInfoPanel } from '../components/AssetInfoPanel';
import { CandlestickChart } from '../components/CandlestickChart';
import { StrategyComparison } from '../components/StrategyComparison';
import type { YearFilterValue } from '../components/YearFilter';
import { useAssetInfo } from '../hooks/useAssetInfo';
import { useAssetList } from '../hooks/useAssetList';

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
 * Laid out as two regions: the asset list panel on the left edge, and the
 * detail panel filling the rest of the width. The left panel (see
 * `AssetListPanel`) hosts both the "Stocks" and "Others" groups as tabs -
 * both are handed their slice of a single shared fetch (see
 * `useAssetList`), but each renders as its own independent `AssetList`
 * instance with its own local search box/state, and both stay mounted
 * regardless of which tab is active.
 *
 * The whole panel can be collapsed via a single toggle button (placed on
 * its right edge, facing the detail panel), freeing up width for the
 * detail panel once an asset is selected. Collapsing only hides the panel
 * visually - the underlying `AssetList` instances stay mounted so their
 * search text and selection state survive re-expanding.
 *
 * The asset's info (earliest/latest imported date) is fetched once here
 * and shared: it's shown directly in `AssetInfoPanel` and also used to
 * default the backtest form's start/end dates to the asset's full range.
 */
export function AssetBrowser({ year, balance }: AssetBrowserProps) {
  const [selectedAsset, setSelectedAsset] = useState<string | null>(null);
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
    <div className="flex items-stretch gap-6">
      <AssetListPanel
        year={year}
        stocks={stocks}
        others={others}
        selected={selectedAsset}
        onSelect={setSelectedAsset}
        loading={assetsLoading}
        error={assetsError}
      />

      <div className="flex min-w-0 flex-1 flex-col gap-10">
        {selectedAsset ? (
          <>
            <AssetInfoPanel info={info} loading={loading} error={error} />
            <CandlestickChart asset={selectedAsset} start={info?.earliest} end={info?.latest} />
            <StrategyComparison
              asset={selectedAsset}
              defaultStart={defaultStart}
              defaultEnd={defaultEnd}
              balance={balance}
            />
          </>
        ) : (
          <p className="py-10 text-text">Select an asset to get started.</p>
        )}
      </div>
    </div>
  );
}
