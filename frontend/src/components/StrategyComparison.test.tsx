import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { StrategyComparison } from './StrategyComparison';
import {
  ApiError,
  getStopLosses,
  getStrategies,
  runBacktest,
  type BacktestResult,
  type StrategyInfo,
} from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    getStrategies: vi.fn(),
    getStopLosses: vi.fn(),
    runBacktest: vi.fn(),
  };
});

const mockedGetStrategies = vi.mocked(getStrategies);
const mockedGetStopLosses = vi.mocked(getStopLosses);
const mockedRunBacktest = vi.mocked(runBacktest);

function buildResult(overrides: Partial<BacktestResult> = {}): BacktestResult {
  return {
    strategyName: 'Buy & Hold',
    startingBalance: 10000,
    endingBalance: 12000,
    profit: 2000,
    profitPercentage: 20,
    totalOperations: 1,
    gains: 1,
    losses: 0,
    winRate: 100,
    maxDrawdownAmount: 100,
    maxDrawdownPercentage: 1,
    operations: [],
    ...overrides,
  };
}

/** Keeps the returned promise pending forever, to inspect pre-fetch state. */
function pendingForever<T>(): Promise<T> {
  return new Promise<T>(() => {});
}

/** Builds a zero-param `StrategyInfo` for each given name. */
function strategyInfos(...names: string[]): StrategyInfo[] {
  return names.map((name) => ({ name, params: [] }));
}

function renderComparison(props: Partial<React.ComponentProps<typeof StrategyComparison>> = {}) {
  return render(
    <StrategyComparison
      asset="PETR4"
      defaultStart="2024-01-01"
      defaultEnd="2024-12-31"
      balance="10000.00"
      {...props}
    />,
  );
}

async function waitForFetchesToSettle() {
  await waitFor(() => expect(screen.getByLabelText('Strategy')).not.toBeDisabled());
  await waitFor(() => expect(screen.getByLabelText('Stop-loss type')).not.toBeDisabled());
}

beforeEach(() => {
  mockedGetStrategies.mockReset();
  mockedGetStopLosses.mockReset();
  mockedRunBacktest.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe('StrategyComparison', () => {
  it('starts with the strategy and stop-loss dropdowns disabled and showing a loading placeholder', () => {
    mockedGetStrategies.mockReturnValue(pendingForever());
    mockedGetStopLosses.mockReturnValue(pendingForever());

    renderComparison();

    const strategySelect = screen.getByLabelText('Strategy');
    expect(strategySelect).toBeDisabled();
    expect(within(strategySelect).getByText('Loading…')).toBeInTheDocument();

    const stopLossSelect = screen.getByLabelText('Stop-loss type');
    expect(stopLossSelect).toBeDisabled();
    expect(within(stopLossSelect).getByText('Loading…')).toBeInTheDocument();

    expect(screen.getByRole('button', { name: 'Run comparison' })).toBeDisabled();
  });

  it('populates the strategy dropdown after fetching, excluding buy-and-hold', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('bter', 'buy-and-hold', 'two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['fixed-amount', 'none', 'percent']);

    renderComparison();

    await waitForFetchesToSettle();

    const strategySelect = screen.getByLabelText('Strategy') as HTMLSelectElement;
    const optionValues = within(strategySelect)
      .getAllByRole('option')
      .map((option) => (option as HTMLOptionElement).value);
    expect(optionValues).toEqual(['bter', 'two-candle-breakout']);
    expect(optionValues).not.toContain('buy-and-hold');
  });

  it('populates the stop-loss dropdown after fetching, defaulting the selection to "none"', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['fixed-amount', 'none', 'percent']);

    renderComparison();

    await waitForFetchesToSettle();

    const stopLossSelect = screen.getByLabelText('Stop-loss type') as HTMLSelectElement;
    const optionValues = within(stopLossSelect)
      .getAllByRole('option')
      .map((option) => (option as HTMLOptionElement).value);
    expect(optionValues).toEqual(['fixed-amount', 'none', 'percent']);
    expect(stopLossSelect.value).toBe('none');
  });

  it('surfaces a strategies-fetch failure as an alert, without blocking the rest of the form', async () => {
    mockedGetStrategies.mockRejectedValue(new ApiError(500, 'strategies blew up'));
    mockedGetStopLosses.mockResolvedValue(['none']);

    renderComparison();

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not load strategies: strategies blew up',
    );
  });

  it('keeps the trigger value field disabled while stop-loss type is "none", and enables it otherwise', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none', 'percent']);
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    const valueField = screen.getByLabelText(/Trigger value/);
    expect(valueField).toBeDisabled();

    await user.selectOptions(screen.getByLabelText('Stop-loss type'), 'percent');

    expect(valueField).not.toBeDisabled();
  });

  it('disables submit until the required date fields are filled in', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none']);

    renderComparison({ defaultStart: undefined, defaultEnd: undefined });
    await waitForFetchesToSettle();

    const submit = screen.getByRole('button', { name: 'Run comparison' });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByLabelText('Start date'), { target: { value: '2024-01-01' } });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByLabelText('End date'), { target: { value: '2024-12-31' } });
    expect(submit).not.toBeDisabled();
  });

  it('runs the baseline and challenger concurrently, defaulting stopLossType to "none" (no stopLoss sent)', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none', 'percent']);
    mockedRunBacktest.mockImplementation(async (payload) =>
      payload.strategy === 'buy-and-hold'
        ? buildResult({ strategyName: 'Buy & Hold' })
        : buildResult({ strategyName: 'Two Candle Breakout' }),
    );
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    await user.click(screen.getByRole('button', { name: 'Run comparison' }));

    expect(await screen.findByText('Buy & Hold')).toBeInTheDocument();
    expect(await screen.findByText('Two Candle Breakout')).toBeInTheDocument();
    expect(screen.getByText('Your pick')).toBeInTheDocument();

    expect(mockedRunBacktest).toHaveBeenCalledTimes(2);
    const [baselineCall, challengerCall] = mockedRunBacktest.mock.calls.map(([payload]) => payload);
    expect(baselineCall.strategy).toBe('buy-and-hold');
    expect(baselineCall.stopLoss).toBeUndefined();
    expect(challengerCall.strategy).toBe('two-candle-breakout');
    expect(challengerCall.stopLoss).toBeUndefined();
  });

  it('applies the configured stop-loss only to the challenger request, never the baseline', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none', 'percent']);
    mockedRunBacktest.mockResolvedValue(buildResult());
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    await user.selectOptions(screen.getByLabelText('Stop-loss type'), 'percent');
    await user.type(screen.getByLabelText(/Trigger value/), '5');

    await user.click(screen.getByRole('button', { name: 'Run comparison' }));

    await waitFor(() => expect(mockedRunBacktest).toHaveBeenCalledTimes(2));
    const [baselineCall, challengerCall] = mockedRunBacktest.mock.calls.map(([payload]) => payload);
    expect(baselineCall.strategy).toBe('buy-and-hold');
    expect(baselineCall.stopLoss).toBeUndefined();
    expect(challengerCall.stopLoss).toEqual({ type: 'percent', value: 5 });
  });

  it('renders the baseline error and the challenger result independently when only the baseline fails', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none']);
    mockedRunBacktest.mockImplementation(async (payload) =>
      payload.strategy === 'buy-and-hold'
        ? Promise.reject(new ApiError(500, 'baseline exploded'))
        : Promise.resolve(buildResult({ strategyName: 'Two Candle Breakout' })),
    );
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    await user.click(screen.getByRole('button', { name: 'Run comparison' }));

    expect(await screen.findByText('baseline exploded')).toBeInTheDocument();
    expect(screen.getByText('Two Candle Breakout')).toBeInTheDocument();
    expect(screen.queryByText('Buy & Hold')).not.toBeInTheDocument();
  });

  it('renders the challenger error and the baseline result independently when only the challenger fails', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none']);
    mockedRunBacktest.mockImplementation(async (payload) =>
      payload.strategy === 'buy-and-hold'
        ? Promise.resolve(buildResult({ strategyName: 'Buy & Hold' }))
        : Promise.reject(new ApiError(500, 'challenger exploded')),
    );
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    await user.click(screen.getByRole('button', { name: 'Run comparison' }));

    expect(await screen.findByText('challenger exploded')).toBeInTheDocument();
    expect(screen.getByText('Buy & Hold')).toBeInTheDocument();
    expect(screen.queryByText('Two Candle Breakout')).not.toBeInTheDocument();
  });

  it('renders no param inputs, and omits strategyParams from the request, for a zero-param strategy', async () => {
    mockedGetStrategies.mockResolvedValue(strategyInfos('two-candle-breakout'));
    mockedGetStopLosses.mockResolvedValue(['none']);
    mockedRunBacktest.mockResolvedValue(buildResult());
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    expect(screen.queryByLabelText('Short period')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Run comparison' }));

    await waitFor(() => expect(mockedRunBacktest).toHaveBeenCalledTimes(2));
    const [, challengerCall] = mockedRunBacktest.mock.calls.map(([payload]) => payload);
    expect(challengerCall.strategyParams).toBeUndefined();
  });

  it('renders one labeled, default-prefilled input per declared param when a param strategy is selected', async () => {
    mockedGetStrategies.mockResolvedValue([
      {
        name: 'sma-crossover',
        params: [
          { key: 'shortPeriod', label: 'Short period', default: 10, min: 1, step: 1 },
          { key: 'longPeriod', label: 'Long period', default: 30, min: 2, step: 1 },
        ],
      },
    ]);
    mockedGetStopLosses.mockResolvedValue(['none']);

    renderComparison();
    await waitForFetchesToSettle();

    const shortPeriod = await screen.findByLabelText('Short period');
    const longPeriod = screen.getByLabelText('Long period');
    expect((shortPeriod as HTMLInputElement).value).toBe('10');
    expect((longPeriod as HTMLInputElement).value).toBe('30');
  });

  it('disables submit until all declared param fields are filled, then includes strategyParams on submit', async () => {
    mockedGetStrategies.mockResolvedValue([
      {
        name: 'rsi-threshold',
        params: [
          { key: 'period', label: 'RSI period', default: 14, min: 2, step: 1 },
          { key: 'oversold', label: 'Oversold threshold', default: 30, min: 0, max: 100, step: 1 },
          { key: 'overbought', label: 'Overbought threshold', default: 70, min: 0, max: 100, step: 1 },
        ],
      },
    ]);
    mockedGetStopLosses.mockResolvedValue(['none']);
    mockedRunBacktest.mockResolvedValue(buildResult());
    const user = userEvent.setup();

    renderComparison();
    await waitForFetchesToSettle();

    const submit = screen.getByRole('button', { name: 'Run comparison' });
    expect(submit).not.toBeDisabled();

    const periodField = screen.getByLabelText('RSI period');
    await user.clear(periodField);
    expect(submit).toBeDisabled();

    await user.type(periodField, '21');
    expect(submit).not.toBeDisabled();

    await user.click(submit);

    await waitFor(() => expect(mockedRunBacktest).toHaveBeenCalledTimes(2));
    const [, challengerCall] = mockedRunBacktest.mock.calls.map(([payload]) => payload);
    expect(challengerCall.strategyParams).toEqual({ period: 21, oversold: 30, overbought: 70 });
  });
});
