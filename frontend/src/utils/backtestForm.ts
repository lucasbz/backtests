/**
 * Pure validation helpers for `StrategyComparison`'s backtest form, split
 * out from the component so the "when is this submittable" logic can be
 * unit-tested without rendering/interacting with the whole form.
 */

import type { StrategyParam } from '../api/client';

/**
 * Every param the selected strategy declares must have a non-empty value in
 * `strategyParamValues` that parses to a finite number - same shape as
 * `isStopLossValid`'s "blank means invalid, otherwise must parse" check,
 * just iterating the strategy's declared params instead of a single fixed
 * field. Cross-field rules (e.g. shortPeriod < longPeriod) are NOT
 * re-validated here - left to the existing 400-from-submit path.
 */
export function areStrategyParamsValid(
  params: StrategyParam[],
  strategyParamValues: Record<string, string>,
): boolean {
  return params.every((param) => {
    const value = strategyParamValues[param.key];
    return value !== undefined && value !== '' && Number.isFinite(Number(value));
  });
}

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
  // The selected strategy's declared params (`[]` for parameterless
  // strategies, or while `GET /api/strategies` is still loading).
  strategyParams: StrategyParam[];
  strategyParamValues: Record<string, string>;
}

/**
 * The backtest form can be submitted once the required fields are filled
 * in, the stop-loss configuration (if any) is valid, every param the
 * selected strategy declares has a valid value, and no request is already
 * in flight.
 */
export function canSubmitBacktest({
  start,
  end,
  strategy,
  balance,
  stopLossType,
  stopLossValue,
  loading,
  strategyParams,
  strategyParamValues,
}: BacktestFormFields): boolean {
  return (
    start !== '' &&
    end !== '' &&
    strategy !== '' &&
    balance.trim() !== '' &&
    isStopLossValid(stopLossType, stopLossValue) &&
    areStrategyParamsValid(strategyParams, strategyParamValues) &&
    !loading
  );
}
