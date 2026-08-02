import type { BacktestResult } from '../api/client';
import { formatCurrency, formatPercentage } from '../utils/format';
import { cardClasses } from '../styles/ui';

export interface BaselineResultCardProps {
  result: BacktestResult;
}

/**
 * Buy & Hold always runs as the fixed comparison baseline (see
 * StrategyComparison.tsx), so its card is intentionally simpler than
 * BacktestResultCard's - e.g. no operations table, no max drawdown, since
 * the baseline is a single buy-then-sell and isn't the thing being
 * evaluated. Deliberately a separate component (rather than a variant/prop
 * on BacktestResultCard) so the two can keep diverging in style
 * independently.
 */
export function BaselineResultCard({ result }: BaselineResultCardProps) {
  return (
    <div className={`${cardClasses} border-dashed border-border`}>
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <div>
          <span className="mb-1 block text-xs font-semibold uppercase tracking-wide text-text/70">
            Baseline
          </span>
          <h3 className="text-lg font-semibold text-text-strong">{result.strategyName}</h3>
        </div>
        <span className="text-sm text-text-strong">
          {formatCurrency(result.startingBalance)} &rarr; {formatCurrency(result.endingBalance)} |{' '}
          <span className={result.profit >= 0 ? 'font-medium text-gain' : 'font-medium text-loss'}>
            {formatCurrency(result.profit)} ({formatPercentage(result.profitPercentage)})
          </span>
        </span>
      </div>
    </div>
  );
}
