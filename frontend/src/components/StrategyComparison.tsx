import { useCallback, useEffect, useState, type FormEvent } from 'react';
import {
  ApiError,
  getStrategies,
  runBacktest,
  type BacktestResult,
} from '../api/client';
import { CurrencyInput } from './CurrencyInput';
import { BacktestResultCard } from './BacktestResultCard';
import './StrategyComparison.css';

// Buy & Hold is always run as the fixed baseline for comparison, so it's
// never offered as a pickable option below - see `filter` on `strategies`.
const BUY_AND_HOLD = 'buy-and-hold';

export interface StrategyComparisonProps {
  /** Ticker to run the backtest against, locked to the current selection. */
  ticker: string;
  /**
   * Ticker's earliest imported date (from `GET /api/info`), used to
   * pre-fill the start date. Undefined while info is still loading.
   */
  defaultStart?: string;
  /**
   * Ticker's latest imported date (from `GET /api/info`), used to
   * pre-fill the end date. Undefined while info is still loading.
   */
  defaultEnd?: string;
}

export function StrategyComparison({ ticker, defaultStart, defaultEnd }: StrategyComparisonProps) {
  const [strategies, setStrategies] = useState<string[]>([]);
  const [strategiesError, setStrategiesError] = useState<string | null>(null);

  const [start, setStart] = useState('');
  const [end, setEnd] = useState('');
  const [strategy, setStrategy] = useState('');
  const [balance, setBalance] = useState('10000.00');
  const handleBalanceChange = useCallback((value: string) => setBalance(value), []);

  const [baselineResult, setBaselineResult] = useState<BacktestResult | null>(null);
  const [baselineError, setBaselineError] = useState<string | null>(null);
  const [challengerResult, setChallengerResult] = useState<BacktestResult | null>(null);
  const [challengerError, setChallengerError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    getStrategies()
      .then((list) => {
        if (cancelled) return;
        // Buy & Hold is always run as the fixed baseline (see below), so it
        // is filtered out of the pickable list here.
        const pickable = list.filter((name) => name !== BUY_AND_HOLD);
        setStrategies(pickable);
        setStrategy((current) => current || pickable[0] || '');
      })
      .catch((err) => {
        if (cancelled) return;
        setStrategiesError(
          err instanceof ApiError ? err.message : 'Could not load strategies.',
        );
      });

    return () => {
      cancelled = true;
    };
  }, []);

  // Selecting a different ticker invalidates any prior result, and the date
  // fields should reflect the new ticker's (possibly year-clamped) default
  // range. This is a single effect - keyed on `ticker` *and* the defaults -
  // rather than two separate ones, because `defaultStart`/`defaultEnd` can
  // legitimately compute to the same string for two different tickers (e.g.
  // both have full coverage for the selected year), which would leave a
  // `[defaultStart, defaultEnd]`-only effect unable to tell a new ticker's
  // defaults apart from the old one's and never re-fire. Keying on `ticker`
  // as well guarantees this always runs on a ticker change, regardless of
  // whether the computed default strings happen to match.
  useEffect(() => {
    setBaselineResult(null);
    setBaselineError(null);
    setChallengerResult(null);
    setChallengerError(null);
    setStart(defaultStart ?? '');
    setEnd(defaultEnd ?? '');
  }, [ticker, defaultStart, defaultEnd]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setLoading(true);
    setBaselineError(null);
    setBaselineResult(null);
    setChallengerError(null);
    setChallengerResult(null);

    // Always request operations from the API; the result card renders them
    // in a collapsible section rather than gating the request on a toggle.
    const sharedRequest = { ticker, start, end, balance, verbose: true };

    const [baselineOutcome, challengerOutcome] = await Promise.allSettled([
      runBacktest({ ...sharedRequest, strategy: BUY_AND_HOLD }),
      runBacktest({ ...sharedRequest, strategy }),
    ]);

    if (baselineOutcome.status === 'fulfilled') {
      setBaselineResult(baselineOutcome.value);
    } else {
      const err = baselineOutcome.reason;
      setBaselineError(err instanceof ApiError ? err.message : 'Unexpected error running backtest.');
    }

    if (challengerOutcome.status === 'fulfilled') {
      setChallengerResult(challengerOutcome.value);
    } else {
      const err = challengerOutcome.reason;
      setChallengerError(
        err instanceof ApiError ? err.message : 'Unexpected error running backtest.',
      );
    }

    setLoading(false);
  }

  const canSubmit =
    start !== '' && end !== '' && strategy !== '' && balance.trim() !== '' && !loading;

  return (
    <section className="strategy-comparison">
      <form className="strategy-comparison__form" onSubmit={handleSubmit}>
        <input id="bt-ticker" type="hidden" value={ticker} readOnly disabled />

        <div className="strategy-comparison__field">
          <label htmlFor="bt-start">Start date</label>
          <input
            id="bt-start"
            type="date"
            value={start}
            onChange={(event) => setStart(event.target.value)}
          />
        </div>

        <div className="strategy-comparison__field">
          <label htmlFor="bt-end">End date</label>
          <input
            id="bt-end"
            type="date"
            value={end}
            onChange={(event) => setEnd(event.target.value)}
          />
        </div>

        <div className="strategy-comparison__field">
          <label htmlFor="bt-balance">Starting balance</label>
          <CurrencyInput
            id="bt-balance"
            initialValue={10000}
            onValueChange={handleBalanceChange}
          />
        </div>
        
        <div className="strategy-comparison__field">
          <label htmlFor="bt-strategy">Strategy to compare against Buy &amp; Hold</label>
          <select
            id="bt-strategy"
            value={strategy}
            onChange={(event) => setStrategy(event.target.value)}
            disabled={strategies.length === 0}
          >
            {strategies.length === 0 && <option value="">Loading…</option>}
            {strategies.map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </div>


        <button type="submit" disabled={!canSubmit}>
          {loading ? 'Running…' : 'Run comparison'}
        </button>
      </form>

      {strategiesError && (
        <div className="strategy-comparison__error" role="alert">
          Could not load strategies: {strategiesError}
        </div>
      )}

      {(baselineResult || challengerResult || baselineError || challengerError) && (
        <div className="strategy-comparison__results">
          <div className="strategy-comparison__result-column">
            {baselineError && (
              <div className="strategy-comparison__error" role="alert">
                {baselineError}
              </div>
            )}
            {baselineResult && (
              <BacktestResultCard
                result={baselineResult}
                label="Baseline"
                variant="baseline"
              />
            )}
          </div>

          <div className="strategy-comparison__result-column">
            {challengerError && (
              <div className="strategy-comparison__error" role="alert">
                {challengerError}
              </div>
            )}
            {challengerResult && (
              <BacktestResultCard
                result={challengerResult}
                label="Your pick"
                variant="challenger"
              />
            )}
          </div>
        </div>
      )}
    </section>
  );
}
