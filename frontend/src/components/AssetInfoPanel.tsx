import type { AssetInfo } from '../api/client';
import { errorBoxClasses } from '../styles/ui';

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
    <section className="text-left">
      {loading && <p className="text-sm text-text">Loading info…</p>}

      {error && (
        <div className={errorBoxClasses} role="alert">
          {error}
        </div>
      )}

      {info && (
        <div className="overflow-hidden rounded-2xl border border-border bg-surface shadow-lg shadow-black/20">
          <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-3">
            <h2 className="text-xl font-semibold text-text-strong">{info.asset}</h2>
          </div>
          <div className="flex items-center justify-between gap-4 px-5 py-3">
            <span className="text-text">
              From <b className="font-semibold text-text-strong">{info.earliest}</b> to{' '}
              <b className="font-semibold text-text-strong">{info.latest}</b>
            </span>
          </div>
        </div>
      )}
    </section>
  );
}
