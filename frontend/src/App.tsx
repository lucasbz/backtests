import { useState } from 'react';
import { TickerBrowser } from './views/TickerBrowser';
import { YearFilter, type YearFilterValue } from './components/YearFilter';
import './App.css';

function App() {
  const [year, setYear] = useState<YearFilterValue>(new Date().getFullYear());

  return (
    <div className="app">
      <header className="app__header">
        <h1>B3 Backtester</h1>
        <YearFilter value={year} onChange={setYear} />
      </header>

      <main className="app__main">
        <TickerBrowser year={year} />
      </main>
    </div>
  );
}

export default App;
