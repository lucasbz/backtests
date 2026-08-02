import { describe, expect, it } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { BacktestResultCard } from './BacktestResultCard';
import { formatCurrency, formatPercentage } from '../utils/format';
import type { BacktestResult } from '../api/client';

/**
 * `toHaveTextContent` normalizes the rendered node's text by collapsing all
 * whitespace (including `Intl.NumberFormat`'s non-breaking space between
 * "R$" and the amount) down to regular spaces before comparing - so the
 * string being compared against needs the same treatment, or an
 * otherwise-matching assertion fails on an invisible NBSP-vs-space
 * mismatch.
 */
function normalizeText(value: string): string {
  return value.replace(/\u00a0/g, ' ');
}

/**
 * The first operation's buy/sell prices and quantity are deliberately huge
 * rather than "realistic" trading numbers: they're chosen so that naive
 * float64 subtraction of the two already-rounded major-unit prices before
 * multiplying by quantity accumulates enough cancellation error to land on
 * a visibly different cent value (R$ 5.000.000,82) than the exact
 * integer-cents computation (R$ 5.000.000,00) - see BacktestResultCard's
 * Profit column comment. Small, everyday price/quantity combinations don't
 * diverge enough to survive `formatCurrency`'s rounding to cents, which
 * would make a test built on them tautological.
 */
function buildResult(overrides: Partial<BacktestResult> = {}): BacktestResult {
  return {
    strategyName: 'Two Candle Breakout',
    startingBalance: 10000,
    endingBalance: 15000,
    profit: 5000,
    profitPercentage: 50,
    totalOperations: 2,
    gains: 1,
    losses: 1,
    winRate: 50,
    maxDrawdownAmount: 1234.5,
    maxDrawdownPercentage: 12.345,
    operations: [
      {
        date: '2024-01-03',
        buyOrder: { date: '2024-01-02', price: 8395244.37, quantity: 500000000 },
        sellOrder: { date: '2024-01-03', price: 8395244.38, quantity: 500000000 },
        days: 1,
      },
      {
        date: '2024-02-10',
        buyOrder: { date: '2024-02-05', price: 55, quantity: 100 },
        sellOrder: { date: '2024-02-10', price: 50, quantity: 100 },
        days: 5,
      },
    ],
    ...overrides,
  };
}

describe('BacktestResultCard', () => {
  it('renders the max drawdown row with the formatted amount and percentage', () => {
    render(<BacktestResultCard result={buildResult()} />);

    const row = screen.getByText('Max drawdown').closest('div');
    expect(row).toHaveTextContent(
      normalizeText(`${formatCurrency(1234.5)} (${formatPercentage(12.345)})`),
    );
  });

  it('renders one operations-table row per operation inside the collapsible details', () => {
    const { container } = render(<BacktestResultCard result={buildResult()} />);

    const details = container.querySelector('details');
    expect(details).not.toBeNull();

    const bodyRows = within(details as HTMLElement).getAllByRole('rowgroup')[1];
    expect(within(bodyRows).getAllByRole('row')).toHaveLength(2);
  });

  it("computes each operation's Profit column in exact integer cents, not naive float subtraction", () => {
    const { container } = render(<BacktestResultCard result={buildResult()} />);

    const dataRows = within(container.querySelector('tbody') as HTMLElement).getAllByRole('row');
    const [gainRow, lossRow] = dataRows;

    const gainProfitCell = within(gainRow).getAllByRole('cell').at(-1)!;
    expect(gainProfitCell).toHaveTextContent(normalizeText(formatCurrency(5000000)));

    const lossProfitCell = within(lossRow).getAllByRole('cell').at(-1)!;
    expect(lossProfitCell).toHaveTextContent(normalizeText(formatCurrency(-500)));
  });

  it("renders each operation's Days column", () => {
    const { container } = render(<BacktestResultCard result={buildResult()} />);

    const dataRows = within(container.querySelector('tbody') as HTMLElement).getAllByRole('row');
    const [gainRow, lossRow] = dataRows;

    // Column order: #, Buy date, Buy price, Sell date, Sell price, Days, Qty, Profit.
    expect(within(gainRow).getAllByRole('cell')[5]).toHaveTextContent('1');
    expect(within(lossRow).getAllByRole('cell')[5]).toHaveTextContent('5');
  });

  it('colors the Profit column by gain/loss', () => {
    const { container } = render(<BacktestResultCard result={buildResult()} />);

    const dataRows = within(container.querySelector('tbody') as HTMLElement).getAllByRole('row');
    const [gainRow, lossRow] = dataRows;

    expect(within(gainRow).getAllByRole('cell').at(-1)!.className).toContain('text-gain');
    expect(within(lossRow).getAllByRole('cell').at(-1)!.className).toContain('text-loss');
  });
});
