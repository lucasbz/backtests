/**
 * Pure validation helpers for `StrategyComparison`'s backtest form, split
 * out from the component so the "when is this submittable" logic can be
 * unit-tested without rendering/interacting with the whole form.
 */

/**
 * A stop-loss configuration is valid when:
 * - the type is `"none"` (always valid, regardless of the value field), or
 * - the value is blank (blank always means "no stop-loss", matching the
 *   backend's "omitted stopLoss" behavior), or
 * - the type is set to something other than `"none"`/blank *and* the value
 *   parses to a positive number.
 */
export function isStopLossValid(stopLossType: string, stopLossValue: string): boolean {
  return (
    stopLossType === 'none' ||
    stopLossValue === '' ||
    (stopLossType !== '' && Number(stopLossValue) > 0)
  );
}

export interface BacktestFormFields {
  start: string;
  end: string;
  strategy: string;
  balance: string;
  stopLossType: string;
  stopLossValue: string;
  loading: boolean;
}

/**
 * The backtest form can be submitted once the required fields are filled
 * in, the stop-loss configuration (if any) is valid, and no request is
 * already in flight.
 */
export function canSubmitBacktest({
  start,
  end,
  strategy,
  balance,
  stopLossType,
  stopLossValue,
  loading,
}: BacktestFormFields): boolean {
  return (
    start !== '' &&
    end !== '' &&
    strategy !== '' &&
    balance.trim() !== '' &&
    isStopLossValid(stopLossType, stopLossValue) &&
    !loading
  );
}
