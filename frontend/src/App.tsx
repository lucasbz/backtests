import { TickerBrowser } from './views/TickerBrowser';
import './App.css';

function App() {
  return (
    <div className="app">
      <header className="app__header">
        <h1>B3 Backtester</h1>
      </header>

      <main className="app__main">
        <TickerBrowser />
      </main>
    </div>
  );
}

export default App;
