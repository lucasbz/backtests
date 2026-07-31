import { useEffect, useState } from 'react';
import { TickerList } from '../components/TickerList';
import { TickerInfoPanel } from '../components/TickerInfoPanel';
import { StrategyComparison } from '../components/StrategyComparison';
import type { YearFilterValue } from '../components/YearFilter';
import { useTickerInfo } from '../hooks/useTickerInfo';
import { useTickerList } from '../hooks/useTickerList';
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
 * Laid out as three columns: the "Stocks" ticker list on the left edge, the
 * detail panel in the middle, and the "Others" ticker list on the right
 * edge. Both lists are handed their slice of a single shared fetch (see
 * `useTickerList`), but each renders as its own independent `TickerList`
 * instance with its own local search box/state.
 *
 * Each side column can be collapsed independently via its own toggle button
 * (placed on the inner edge, facing the detail panel), freeing up width for
 * the detail panel once a ticker is selected. Collapsing only hides the
 * column visually - the underlying `TickerList` stays mounted so its search
 * text and selection state survive re-expanding.
 *
 * The ticker's info (earliest/latest imported date) is fetched once here
 * and shared: it's shown directly in `TickerInfoPanel` and also used to
 * default the backtest form's start/end dates to the ticker's full range.
 */
export function TickerBrowser({ year }: TickerBrowserProps) {
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const [stocksCollapsed, setStocksCollapsed] = useState(false);
  const [othersCollapsed, setOthersCollapsed] = useState(false);
  const { info, loading, error } = useTickerInfo(selectedTicker);
  const {
    stocks,
    others,
    loading: tickersLoading,
    error: tickersError,
  } = useTickerList(year);

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
      <div
        className={
          stocksCollapsed
            ? 'ticker-browser__side ticker-browser__side--left ticker-browser__side--collapsed'
            : 'ticker-browser__side ticker-browser__side--left'
        }
      >
        <div className="ticker-browser__side-list">
          <TickerList
            label="Stocks"
            year={year}
            items={stocks}
            selected={selectedTicker}
            onSelect={setSelectedTicker}
            loading={tickersLoading}
            error={tickersError}
          />
        </div>
        <button
          type="button"
          className="ticker-browser__toggle"
          onClick={() => setStocksCollapsed((collapsed) => !collapsed)}
          aria-pressed={stocksCollapsed}
          aria-label={stocksCollapsed ? 'Expand Stocks list' : 'Collapse Stocks list'}
        >
          {stocksCollapsed ? '›' : '‹'}
        </button>
      </div>

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

      <div
        className={
          othersCollapsed
            ? 'ticker-browser__side ticker-browser__side--right ticker-browser__side--collapsed'
            : 'ticker-browser__side ticker-browser__side--right'
        }
      >
        <button
          type="button"
          className="ticker-browser__toggle"
          onClick={() => setOthersCollapsed((collapsed) => !collapsed)}
          aria-pressed={othersCollapsed}
          aria-label={othersCollapsed ? 'Expand Others list' : 'Collapse Others list'}
        >
          {othersCollapsed ? '‹' : '›'}
        </button>
        <div className="ticker-browser__side-list">
          <TickerList
            label="Others"
            year={year}
            items={others}
            selected={selectedTicker}
            onSelect={setSelectedTicker}
            loading={tickersLoading}
            error={tickersError}
          />
        </div>
      </div>
    </div>
  );
}
