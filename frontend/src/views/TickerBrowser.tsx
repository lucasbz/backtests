import { useEffect, useState } from 'react';
import { TickerList } from '../components/TickerList';
import { TickerInfoPanel } from '../components/TickerInfoPanel';
import { StrategyComparison } from '../components/StrategyComparison';
import type { YearFilterValue } from '../components/YearFilter';
import { useTickerInfo } from '../hooks/useTickerInfo';
import './TickerBrowser.css';

export interface TickerBrowserProps {
  /** Restricts the ticker list to symbols with data for this year, or 'all'. */
  year: YearFilterValue;
}

/**
 * Browse the available tickers, then inspect the selected one's imported
 * date range and run a backtest against it. This is the single entry point
 * into the app - selecting a ticker drives both the info panel and the
 * backtest form below it.
 *
 * The ticker's info (earliest/latest imported date) is fetched once here
 * and shared: it's shown directly in `TickerInfoPanel` and also used to
 * default the backtest form's start/end dates to the ticker's full range.
 */
export function TickerBrowser({ year }: TickerBrowserProps) {
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const { info, loading, error } = useTickerInfo(selectedTicker);

  // Changing the year filter can drop the currently selected ticker from
  // the list (it may not have data for that year), so clear the selection
  // rather than risk showing a backtest form for a ticker no longer shown.
  useEffect(() => {
    setSelectedTicker(null);
  }, [year]);

  // When a specific year is selected, default the comparison form to that
  // year's range instead of the ticker's full range - but clamped to the
  // ticker's actual available data, so we never suggest a date that has no
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
    <div className="ticker-browser">
      <TickerList year={year} selected={selectedTicker} onSelect={setSelectedTicker} />

      <div className="ticker-browser__detail">
        {selectedTicker ? (
          <>
            <TickerInfoPanel info={info} loading={loading} error={error} />
            <StrategyComparison
              ticker={selectedTicker}
              defaultStart={defaultStart}
              defaultEnd={defaultEnd}
            />
          </>
        ) : (
          <p className="ticker-browser__empty">Select a ticker to get started.</p>
        )}
      </div>
    </div>
  );
}
