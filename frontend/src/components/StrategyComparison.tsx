import { useEffect, useMemo, useState, type FormEvent } from 'react';
import {
  ApiError,
  getStopLosses,
  getStrategies,
  runBacktest,
  type BacktestResult,
  type StrategyInfo,
} from '../api/client';
import { BacktestResultCard } from './BacktestResultCard';
import { BaselineResultCard } from './BaselineResultCard';
import { buttonPrimaryClasses, errorBoxClasses, fieldClasses } from '../styles/ui';
import { canSubmitBacktest } from '../utils/backtestForm';

// Buy & Hold is always run as the fixed baseline for comparison, so it's
// never offered as a pickable option below - see `filter` on `strategies`.
const BUY_AND_HOLD = 'buy-and-hold';

export interface StrategyComparisonProps {
  /** Asset to run the backtest against, locked to the current selection. */
  asset: string;
  /**
   * Asset's earliest imported date (from `GET /api/info`), used to
   * pre-fill the start date. Undefined while info is still loading.
   */
  defaultStart?: string;
  /**
   * Asset's latest imported date (from `GET /api/info`), used to
   * pre-fill the end date. Undefined while info is still loading.
   */
  defaultEnd?: string;
  /**
   * Current starting balance (plain decimal string, e.g. "10000.00"). Owned
   * by the header in `App` (lifted up since the field moved there), and
   * fed straight into `sharedRequest` below same as before.
   */
  balance: string;
}

export function StrategyComparison({
  asset,
  defaultStart,
  defaultEnd,
  balance,
}: StrategyComparisonProps) {
  const [strategies, setStrategies] = useState<StrategyInfo[]>([]);
  const [strategiesError, setStrategiesError] = useState<string | null>(null);

  const [stopLossTypes, setStopLossTypes] = useState<string[]>([]);
  const [stopLossTypesError, setStopLossTypesError] = useState<string | null>(null);

  const [start, setStart] = useState('');
  const [end, setEnd] = useState('');
  const [strategy, setStrategy] = useState('');

  // Values for the currently-selected strategy's declared params (see
  // `selectedStrategyInfo` below), keyed by param `key`. String, not
  // number, same reasoning as `stopLossValue` - lets an input be
  // legitimately empty rather than coerced to `0`. Reset to the newly
  // selected strategy's declared defaults below whenever `strategy` (or the
  // fetched `strategies` list) changes.
  const [strategyParamValues, setStrategyParamValues] = useState<Record<string, string>>({});

  // Fields are always visible; leaving the value blank means "no
  // stop-loss" (matches the backend's "omitted stopLoss" behavior) - see
  // `stopLossValid`/`handleSubmit`. Only ever applies to the challenger's
  // request, never the Buy & Hold baseline.
  const [stopLossType, setStopLossType] = useState('');
  const [stopLossValue, setStopLossValue] = useState('');

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
        const pickable = list.filter((info) => info.name !== BUY_AND_HOLD);
        setStrategies(pickable);
        setStrategy((current) => current || pickable[0]?.name || '');
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

  useEffect(() => {
    let cancelled = false;

    getStopLosses()
      .then((list) => {
        if (cancelled) return;
        setStopLossTypes(list);
        // Default to "none" (no stop-loss) rather than whichever type
        // happens to sort first - falls back to the first available type
        // only if "none" isn't registered for some reason.
        setStopLossType((current) => current || (list.includes('none') ? 'none' : list[0]) || '');
      })
      .catch((err) => {
        if (cancelled) return;
        setStopLossTypesError(
          err instanceof ApiError ? err.message : 'Could not load stop-loss types.',
        );
      });

    return () => {
      cancelled = true;
    };
  }, []);

  // Selecting a different asset invalidates any prior result, and the date
  // fields should reflect the new asset's (possibly year-clamped) default
  // range. This is a single effect - keyed on `asset` *and* the defaults -
  // rather than two separate ones, because `defaultStart`/`defaultEnd` can
  // legitimately compute to the same string for two different assets (e.g.
  // both have full coverage for the selected year), which would leave a
  // `[defaultStart, defaultEnd]`-only effect unable to tell a new asset's
  // defaults apart from the old one's and never re-fire. Keying on `asset`
  // as well guarantees this always runs on an asset change, regardless of
  // whether the computed default strings happen to match.
  useEffect(() => {
    setBaselineResult(null);
    setBaselineError(null);
    setChallengerResult(null);
    setChallengerError(null);
    setStart(defaultStart ?? '');
    setEnd(defaultEnd ?? '');
  }, [asset, defaultStart, defaultEnd]);

  const selectedStrategyInfo = strategies.find((info) => info.name === strategy);
  // Memoized so this array keeps a stable reference across renders that
  // don't actually change it (i.e. whenever `strategies`/`strategy` are
  // unchanged), so the reset effect below can safely depend on it without
  // re-firing every render.
  const selectedStrategyParams = useMemo(
    () => selectedStrategyInfo?.params ?? [],
    [selectedStrategyInfo],
  );

  // Changing the selected strategy (or the strategies list finishing its
  // fetch, which is when params first become known) resets the per-param
  // inputs to that strategy's declared defaults - mirrors how selecting a
  // new asset resets other form state in `AssetBrowser`'s effect.
  useEffect(() => {
    setStrategyParamValues(
      Object.fromEntries(selectedStrategyParams.map((param) => [param.key, String(param.default)])),
    );
  }, [selectedStrategyParams]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setLoading(true);
    setBaselineError(null);
    setBaselineResult(null);
    setChallengerError(null);
    setChallengerResult(null);

    // Always request operations from the API; the result card renders them
    // in a collapsible section rather than gating the request on a toggle.
    const sharedRequest = { asset, start, end, balance, verbose: true };

    // The stop-loss only ever applies to the challenger's request - the
    // Buy & Hold baseline always runs without one, so it stays a clean,
    // consistent reference point across runs regardless of what stop-loss
    // (if any) is configured here. "none" (the default) always means no
    // stop-loss, regardless of whatever's in the value field.
    const stopLoss =
      stopLossType !== '' && stopLossType !== 'none' && Number(stopLossValue) > 0
        ? { type: stopLossType, value: Number(stopLossValue) }
        : undefined;

    // Only included for strategies that declare params - omitted entirely
    // for parameterless ones, matching the API's "omit for parameterless
    // strategies" contract.
    const strategyParams =
      selectedStrategyParams.length > 0
        ? Object.fromEntries(
            selectedStrategyParams.map((param) => [param.key, Number(strategyParamValues[param.key])]),
          )
        : undefined;

    const [baselineOutcome, challengerOutcome] = await Promise.allSettled([
      runBacktest({ ...sharedRequest, strategy: BUY_AND_HOLD }),
      runBacktest({ ...sharedRequest, strategy, stopLoss, strategyParams }),
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

  // "none" (the default) or a blank value both mean no stop-loss - valid,
  // optional. Any other type needs a positive trigger value. See
  // `utils/backtestForm.ts` for the (unit-tested) logic.
  const canSubmit = canSubmitBacktest({
    start,
    end,
    strategy,
    balance,
    stopLossType,
    stopLossValue,
    loading,
    strategyParams: selectedStrategyParams,
    strategyParamValues,
  });

  return (
    <section className="text-left">
      <form className="grid grid-cols-1 items-end gap-4 sm:grid-cols-2" onSubmit={handleSubmit}>
        <input id="bt-asset" type="hidden" value={asset} readOnly disabled />

        <div className="flex flex-col gap-1.5">
          <label htmlFor="bt-start" className="text-sm text-text">
            Start date
          </label>
          <input
            id="bt-start"
            type="date"
            className={fieldClasses}
            value={start}
            onChange={(event) => setStart(event.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="bt-end" className="text-sm text-text">
            End date
          </label>
          <input
            id="bt-end"
            type="date"
            className={fieldClasses}
            value={end}
            onChange={(event) => setEnd(event.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="bt-strategy" className="text-sm text-text">
            Strategy
          </label>
          <select
            id="bt-strategy"
            className={fieldClasses}
            value={strategy}
            onChange={(event) => setStrategy(event.target.value)}
            disabled={strategies.length === 0}
          >
            {strategies.length === 0 && <option value="">Loading…</option>}
            {strategies.map((info) => (
              <option key={info.name} value={info.name}>
                {info.name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-row gap-3">
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <label htmlFor="bt-stop-loss-type" className="text-sm text-text">
              Stop-loss type
            </label>
            <select
              id="bt-stop-loss-type"
              className={fieldClasses}
              value={stopLossType}
              onChange={(event) => setStopLossType(event.target.value)}
              disabled={stopLossTypes.length === 0}
            >
              {stopLossTypes.length === 0 && <option value="">Loading…</option>}
              {stopLossTypes.map((name) => (
                <option key={name} value={name}>
                  {name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <label htmlFor="bt-stop-loss-value" className="text-sm text-text">
              Trigger value{stopLossType === 'percent' ? ' (%)' : stopLossType === 'fixed-amount' ? ' (R$ per share)' : stopLossType === 'total-fixed-amount' ? ' (R$ total)' : ''}
            </label>
            <input
              id="bt-stop-loss-value"
              type="number"
              min="0"
              step="0.01"
              inputMode="decimal"
              placeholder="None"
              className={fieldClasses}
              value={stopLossValue}
              onChange={(event) => setStopLossValue(event.target.value)}
              disabled={stopLossType === 'none'}
            />
          </div>
        </div>

        {selectedStrategyParams.map((param) => {
          const inputId = `bt-strategy-param-${param.key}`;
          return (
            <div key={param.key} className="flex flex-col gap-1.5">
              <label htmlFor={inputId} className="text-sm text-text">
                {param.label}
              </label>
              <input
                id={inputId}
                type="number"
                min={param.min}
                max={param.max}
                step={param.step}
                className={fieldClasses}
                value={strategyParamValues[param.key] ?? ''}
                onChange={(event) =>
                  setStrategyParamValues((current) => ({
                    ...current,
                    [param.key]: event.target.value,
                  }))
                }
              />
            </div>
          );
        })}

        <button type="submit" className={`${buttonPrimaryClasses} col-span-full justify-self-start`} disabled={!canSubmit}>
          {loading ? 'Running…' : 'Run comparison'}
        </button>
      </form>

      {strategiesError && (
        <div className={`${errorBoxClasses} mt-4`} role="alert">
          Could not load strategies: {strategiesError}
        </div>
      )}

      {stopLossTypesError && (
        <div className={`${errorBoxClasses} mt-4`} role="alert">
          Could not load stop-loss types: {stopLossTypesError}
        </div>
      )}

      {(baselineResult || baselineError) && (
        <div className="mt-6">
          {baselineError && (
            <div className={errorBoxClasses} role="alert">
              {baselineError}
            </div>
          )}
          {baselineResult && <BaselineResultCard result={baselineResult} />}
        </div>
      )}

      {(challengerResult || challengerError) && (
        <div className="mt-4">
          {challengerError && (
            <div className={errorBoxClasses} role="alert">
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
      )}
    </section>
  );
}
