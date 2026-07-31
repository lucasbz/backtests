import type { AssetInfo } from '../api/client';
import './AssetInfoPanel.css';

export interface AssetInfoPanelProps {
  info: AssetInfo | null;
  loading: boolean;
  error: string | null;
}

/**
 * Shows the imported date range for the currently selected asset.
 * Presentational only - the caller (`AssetBrowser`, via `useAssetInfo`)
 * owns the fetch, since the same info is also used to default the backtest
 * form's start/end dates.
 */
export function AssetInfoPanel({ info, loading, error }: AssetInfoPanelProps) {
  return (
    <section className="asset-info-panel">
      {loading && <p className="asset-info-panel__hint">Loading info…</p>}

      {error && (
        <div className="asset-info-panel__error" role="alert">
          {error}
        </div>
      )}

      {info && (
        <div className="asset-info-panel__result">
          <div className="asset-info-panel__result-row">
            <h2>{info.asset}</h2>
          </div>
          <div className="asset-info-panel__result-row">
            <span>From <b>{info.earliest}</b> to <b>{info.latest}</b></span>
          </div>
        </div>
      )}
    </section>
  );
}
