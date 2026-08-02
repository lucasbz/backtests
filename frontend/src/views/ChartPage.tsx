import { useEffect, useState } from 'react';
import { AssetListPanel } from '../components/AssetListPanel';
import { CandlestickChart } from '../components/CandlestickChart';
import type { YearFilterValue } from '../components/YearFilter';
import { useAssetInfo } from '../hooks/useAssetInfo';
import { useAssetList } from '../hooks/useAssetList';

export interface ChartPageProps {
  /** Restricts the asset list to symbols with data for this year, or 'all'. */
  year: YearFilterValue;
}

/**
 * Second top-level page (alongside `AssetBrowser`): browse the available
 * assets and inspect one's candlestick chart, with none of the backtest/
 * comparison concerns `AssetBrowser` layers on top. Owns its own
 * `selectedAsset` state, independent of `AssetBrowser`'s - see `App`, which
 * keeps both pages mounted at all times so each remembers its own
 * last-picked asset when you switch away and back.
 *
 * Laid out the same way as `AssetBrowser`: the asset list panel on the
 * left edge, and the chart filling the rest of the width once an asset is
 * selected.
 */
export function ChartPage({ year }: ChartPageProps) {
  const [selectedAsset, setSelectedAsset] = useState<string | null>(null);
  const { info } = useAssetInfo(selectedAsset);
  const {
    stocks,
    others,
    loading: assetsLoading,
    error: assetsError,
  } = useAssetList(year);

  // Changing the year filter can drop the currently selected asset from
  // the list (it may not have data for that year), so clear the selection
  // rather than risk showing a chart for an asset no longer shown.
  useEffect(() => {
    setSelectedAsset(null);
  }, [year]);

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
          <CandlestickChart asset={selectedAsset} start={info?.earliest} end={info?.latest} />
        ) : (
          <p className="py-10 text-text">Select an asset to get started.</p>
        )}
      </div>
    </div>
  );
}
