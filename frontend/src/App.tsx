import { useCallback, useState } from 'react';
import { AssetBrowser } from './views/AssetBrowser';
import { YearFilter, type YearFilterValue } from './components/YearFilter';
import { CurrencyInput } from './components/CurrencyInput';

// Single source of truth for the starting balance both `balance` state
// below (plain decimal string, matching the API's format) and
// `CurrencyInput`'s `initialValue` (major-unit number) are seeded from, so
// they can't silently drift apart on a future edit.
const DEFAULT_BALANCE = 10000;

function App() {
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
          <h1 className="text-lg font-semibold text-text-strong sm:text-xl">
            [b3] Bold Beren Backtester
          </h1>
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
        <AssetBrowser year={year} balance={balance} />
      </main>
    </div>
  );
}

export default App;
