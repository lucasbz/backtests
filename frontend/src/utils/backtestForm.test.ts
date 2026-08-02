import { describe, expect, it } from 'vitest';
import {
  areStrategyParamsValid,
  canSubmitBacktest,
  isStopLossValid,
  type BacktestFormFields,
} from './backtestForm';
import type { StrategyParam } from '../api/client';

describe('isStopLossValid', () => {
  it('is always valid for type "none", regardless of the value field', () => {
    expect(isStopLossValid('none', '')).toBe(true);
    expect(isStopLossValid('none', '10')).toBe(true);
    expect(isStopLossValid('none', '-5')).toBe(true);
    expect(isStopLossValid('none', 'not-a-number')).toBe(true);
  });

  it('is valid whenever the value is blank, meaning "no stop-loss"', () => {
    expect(isStopLossValid('', '')).toBe(true);
    expect(isStopLossValid('percent', '')).toBe(true);
    expect(isStopLossValid('fixed-amount', '')).toBe(true);
  });

  it('requires a positive numeric value for any other type', () => {
    expect(isStopLossValid('percent', '5')).toBe(true);
    expect(isStopLossValid('fixed-amount', '0.01')).toBe(true);
  });

  it('rejects other types with a zero, negative, or non-numeric value', () => {
    expect(isStopLossValid('percent', '0')).toBe(false);
    expect(isStopLossValid('percent', '-1')).toBe(false);
    expect(isStopLossValid('percent', 'abc')).toBe(false);
  });

  it('rejects a blank type paired with a non-blank value', () => {
    expect(isStopLossValid('', '5')).toBe(false);
  });
});

describe('areStrategyParamsValid', () => {
  const shortPeriod: StrategyParam = { key: 'shortPeriod', label: 'Short period', default: 10, min: 1, step: 1 };
  const longPeriod: StrategyParam = { key: 'longPeriod', label: 'Long period', default: 30, min: 2, step: 1 };

  it('is true when no params are declared, regardless of strategyParamValues', () => {
    expect(areStrategyParamsValid([], {})).toBe(true);
    expect(areStrategyParamsValid([], { unrelated: 'abc' })).toBe(true);
  });

  it('is true when every declared param has a numeric value', () => {
    expect(
      areStrategyParamsValid([shortPeriod, longPeriod], { shortPeriod: '10', longPeriod: '30' }),
    ).toBe(true);
  });

  it('is false when a declared param is missing from strategyParamValues', () => {
    expect(areStrategyParamsValid([shortPeriod, longPeriod], { shortPeriod: '10' })).toBe(false);
  });

  it('is false when a declared param is blank', () => {
    expect(
      areStrategyParamsValid([shortPeriod, longPeriod], { shortPeriod: '10', longPeriod: '' }),
    ).toBe(false);
  });

  it('is false when a declared param is non-numeric', () => {
    expect(
      areStrategyParamsValid([shortPeriod, longPeriod], { shortPeriod: '10', longPeriod: 'abc' }),
    ).toBe(false);
  });
});

describe('canSubmitBacktest', () => {
  function fields(overrides: Partial<BacktestFormFields> = {}): BacktestFormFields {
    return {
      start: '2024-01-01',
      end: '2024-12-31',
      strategy: 'two-candle-breakout',
      balance: '10000.00',
      stopLossType: 'none',
      stopLossValue: '',
      loading: false,
      strategyParams: [],
      strategyParamValues: {},
      ...overrides,
    };
  }

  it('is true when all required fields are filled, stop-loss is valid, and not loading', () => {
    expect(canSubmitBacktest(fields())).toBe(true);
  });

  it('is false when start is blank', () => {
    expect(canSubmitBacktest(fields({ start: '' }))).toBe(false);
  });

  it('is false when end is blank', () => {
    expect(canSubmitBacktest(fields({ end: '' }))).toBe(false);
  });

  it('is false when strategy is blank', () => {
    expect(canSubmitBacktest(fields({ strategy: '' }))).toBe(false);
  });

  it('is false when balance is blank or whitespace-only', () => {
    expect(canSubmitBacktest(fields({ balance: '' }))).toBe(false);
    expect(canSubmitBacktest(fields({ balance: '   ' }))).toBe(false);
  });

  it('is false when the stop-loss configuration is invalid', () => {
    expect(canSubmitBacktest(fields({ stopLossType: 'percent', stopLossValue: '0' }))).toBe(
      false,
    );
  });

  it('is false while a request is already loading', () => {
    expect(canSubmitBacktest(fields({ loading: true }))).toBe(false);
  });

  it('is unaffected by strategyParamValues when the strategy declares no params', () => {
    expect(canSubmitBacktest(fields({ strategyParams: [], strategyParamValues: {} }))).toBe(true);
    expect(
      canSubmitBacktest(fields({ strategyParams: [], strategyParamValues: { extra: '' } })),
    ).toBe(true);
  });

  it('is true when the strategy declares params and all are filled with numeric values', () => {
    const strategyParams: StrategyParam[] = [
      { key: 'shortPeriod', label: 'Short period', default: 10, min: 1, step: 1 },
      { key: 'longPeriod', label: 'Long period', default: 30, min: 2, step: 1 },
    ];
    expect(
      canSubmitBacktest(
        fields({ strategyParams, strategyParamValues: { shortPeriod: '10', longPeriod: '30' } }),
      ),
    ).toBe(true);
  });

  it('is false when the strategy declares params and one is missing or empty', () => {
    const strategyParams: StrategyParam[] = [
      { key: 'shortPeriod', label: 'Short period', default: 10, min: 1, step: 1 },
      { key: 'longPeriod', label: 'Long period', default: 30, min: 2, step: 1 },
    ];
    expect(
      canSubmitBacktest(fields({ strategyParams, strategyParamValues: { shortPeriod: '10' } })),
    ).toBe(false);
    expect(
      canSubmitBacktest(
        fields({ strategyParams, strategyParamValues: { shortPeriod: '10', longPeriod: '' } }),
      ),
    ).toBe(false);
  });

  it('is false when the strategy declares params and one is non-numeric', () => {
    const strategyParams: StrategyParam[] = [
      { key: 'period', label: 'RSI period', default: 14, min: 2, step: 1 },
    ];
    expect(
      canSubmitBacktest(fields({ strategyParams, strategyParamValues: { period: 'abc' } })),
    ).toBe(false);
  });
});
