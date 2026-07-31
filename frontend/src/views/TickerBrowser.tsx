import { useState } from 'react';
import { TickerList } from '../components/TickerList';
import { TickerInfoPanel } from '../components/TickerInfoPanel';
import { StrategyComparison } from '../components/StrategyComparison';
import { useTickerInfo } from '../hooks/useTickerInfo';
import './TickerBrowser.css';

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
export function TickerBrowser() {
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const { info, loading, error } = useTickerInfo(selectedTicker);

  return (
    <div className="ticker-browser">
      <TickerList selected={selectedTicker} onSelect={setSelectedTicker} />

      <div className="ticker-browser__detail">
        {selectedTicker ? (
          <>
            <TickerInfoPanel info={info} loading={loading} error={error} />
            <StrategyComparison
              ticker={selectedTicker}
              defaultStart={info?.earliest}
              defaultEnd={info?.latest}
            />
          </>
        ) : (
          <p className="ticker-browser__empty">Select a ticker to get started.</p>
        )}
      </div>
    </div>
  );
}
