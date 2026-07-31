import { useCallback, useState } from 'react';
import { AssetBrowser } from './views/AssetBrowser';
import { YearFilter, type YearFilterValue } from './components/YearFilter';
import { CurrencyInput } from './components/CurrencyInput';
import './App.css';

function App() {
  const [year, setYear] = useState<YearFilterValue>(new Date().getFullYear());
  // Lifted out of `StrategyComparison` since the balance field now lives in
  // the header - it's threaded down through `AssetBrowser` as a plain prop
  // (same pattern as `year` above), rather than each backtest form owning
  // its own local balance state.
  const [balance, setBalance] = useState('10000.00');
  const handleBalanceChange = useCallback((value: string) => setBalance(value), []);

  return (
    <div className="app">
      <header className="app__header">
        <h1>[b3] Bold Beren Backtester</h1>
        <div className="app__header-controls">
          <YearFilter value={year} onChange={setYear} />
          <div className="app__balance">
            <label htmlFor="header-balance">Balance</label>
            <CurrencyInput
              id="header-balance"
              initialValue={10000}
              onValueChange={handleBalanceChange}
            />
          </div>
        </div>
      </header>

      <main className="app__main">
        <AssetBrowser year={year} balance={balance} />
      </main>
    </div>
  );
}

export default App;
