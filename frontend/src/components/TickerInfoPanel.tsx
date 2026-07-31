import type { TickerInfo } from '../api/client';
import './TickerInfoPanel.css';

export interface TickerInfoPanelProps {
  info: TickerInfo | null;
  loading: boolean;
  error: string | null;
}

/**
 * Shows the imported date range for the currently selected ticker.
 * Presentational only - the caller (`TickerBrowser`, via `useTickerInfo`)
 * owns the fetch, since the same info is also used to default the backtest
 * form's start/end dates.
 */
export function TickerInfoPanel({ info, loading, error }: TickerInfoPanelProps) {
  return (
    <section className="ticker-info-panel">
      <h2>Ticker Info</h2>

      {loading && <p className="ticker-info-panel__hint">Loading info…</p>}

      {error && (
        <div className="ticker-info-panel__error" role="alert">
          {error}
        </div>
      )}

      {info && (
        <div className="ticker-info-panel__result">
          <div className="ticker-info-panel__result-row">
            <span className="ticker-info-panel__label">Ticker</span>
            <span>{info.ticker}</span>
          </div>
          <div className="ticker-info-panel__result-row">
            <span className="ticker-info-panel__label">Earliest date</span>
            <span>{info.earliest}</span>
          </div>
          <div className="ticker-info-panel__result-row">
            <span className="ticker-info-panel__label">Latest date</span>
            <span>{info.latest}</span>
          </div>
        </div>
      )}
    </section>
  );
}
