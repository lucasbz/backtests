import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BaselineResultCard } from './BaselineResultCard';
import { formatCurrency, formatPercentage } from '../utils/format';
import type { BacktestResult } from '../api/client';

/**
 * `toHaveTextContent` collapses all whitespace in the rendered node's text
 * (including `Intl.NumberFormat`'s non-breaking space between "R$" and the
 * amount) down to regular spaces before comparing, so the expected string
 * needs the same treatment or the assertion fails on an invisible
 * NBSP-vs-space mismatch.
 */
function normalizeText(value: string): string {
  return value.replace(/\u00a0/g, ' ');
}

function buildResult(overrides: Partial<BacktestResult> = {}): BacktestResult {
  return {
    strategyName: 'Buy & Hold',
    startingBalance: 10000,
    endingBalance: 12500,
    profit: 2500,
    profitPercentage: 25,
    totalOperations: 1,
    gains: 1,
    losses: 0,
    winRate: 100,
    maxDrawdownAmount: 800,
    maxDrawdownPercentage: 8,
    operations: [
      {
        date: '2024-06-01',
        buyOrder: { date: '2024-01-02', price: 10, quantity: 1000 },
        sellOrder: { date: '2024-06-01', price: 12.5, quantity: 1000 },
        days: 151,
      },
    ],
    ...overrides,
  };
}

describe('BaselineResultCard', () => {
  it('renders the condensed Balance -> Profit header line with the "|" separated from its neighbors', () => {
    render(<BaselineResultCard result={buildResult()} />);

    const header = screen.getByText('Buy & Hold').closest('div')?.parentElement;
    expect(header).toHaveTextContent(
      normalizeText(
        `${formatCurrency(10000)} → ${formatCurrency(12500)} | ${formatCurrency(2500)} (${formatPercentage(25)})`,
      ),
    );
  });

  it('never renders a max drawdown line, even though the result carries that data', () => {
    render(<BaselineResultCard result={buildResult()} />);

    expect(screen.queryByText(/max drawdown/i)).not.toBeInTheDocument();
  });

  it('renders no operations table, unlike BacktestResultCard', () => {
    const { container } = render(<BaselineResultCard result={buildResult()} />);

    expect(container.querySelector('details')).toBeNull();
    expect(container.querySelector('table')).toBeNull();
  });
});
