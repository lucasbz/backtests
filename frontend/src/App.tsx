import { useCallback, useState } from 'react';
import { AssetBrowser } from './views/AssetBrowser';
import { ChartPage } from './views/ChartPage';
import { YearFilter, type YearFilterValue } from './components/YearFilter';
import { CurrencyInput } from './components/CurrencyInput';
import { NavBar, type Page } from './components/NavBar';

// Single source of truth for the starting balance both `balance` state
// below (plain decimal string, matching the API's format) and
// `CurrencyInput`'s `initialValue` (major-unit number) are seeded from, so
// they can't silently drift apart on a future edit.
const DEFAULT_BALANCE = 10000;

function App() {
  // No routing library is installed in this project, so which top-level
  // page is showing is just plain local state - see `NavBar`.
  const [page, setPage] = useState<Page>('home');
  const [year, setYear] = useState<YearFilterValue>(new Date().getFullYear());
  // Lifted out of `StrategyComparison` since the balance field now lives in
  // the header - it's threaded down through `AssetBrowser` as a plain prop
  // (same pattern as `year` above), rather than each backtest form owning
  // its own local balance state.
  const [balance, setBalance] = useState(DEFAULT_BALANCE.toFixed(2));
  const handleBalanceChange = useCallback((value: string) => setBalance(value), []);

  return (
    <div className="flex min-h-full w-full flex-1 flex-col">
      <header className="w-full border-b border-border/70 bg-bg">
        <div className="mx-auto flex w-full max-w-7xl items-center justify-between gap-6 px-6 py-4 sm:px-10">
          <div className="flex items-center gap-6">
            <h1 className="text-lg font-semibold text-text-strong sm:text-xl">
              [b3] Bold Beren Backtester
            </h1>
            <NavBar active={page} onNavigate={setPage} />
          </div>
          <div className="flex items-center gap-2.5 py-1.5 pl-4 pr-1.5">
            <label htmlFor="header-balance" className="text-xs font-medium text-muted">
              Balance
            </label>
            <CurrencyInput
              id="header-balance"
              initialValue={DEFAULT_BALANCE}
              onValueChange={handleBalanceChange}
            />
          </div>
        </div>
      </header>

      <div className="w-full border-b border-border/70 bg-surface/40">
        <div className="mx-auto w-full max-w-7xl px-6 py-4 sm:px-10">
          <YearFilter value={year} onChange={setYear} />
        </div>
      </div>

      <main className="mx-auto w-full max-w-7xl flex-1 px-6 py-10 sm:px-10 sm:py-14">
        {/* Both pages stay mounted at all times, matching `AssetListPanel`'s
            own "keep both tabs mounted, just hide the inactive one"
            convention - it preserves each page's own selected-asset state
            when switching back and forth. */}
        <div className={page === 'home' ? '' : 'hidden'}>
          <AssetBrowser year={year} balance={balance} />
        </div>
        <div className={page === 'charts' ? '' : 'hidden'}>
          <ChartPage year={year} />
        </div>
      </main>
    </div>
  );
}

export default App;
