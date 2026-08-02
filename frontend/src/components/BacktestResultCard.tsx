import type { BacktestResult } from '../api/client';
import { formatCurrency, formatPercentage } from '../utils/format';
import { cardClasses } from '../styles/ui';

export interface BacktestResultCardProps {
  result: BacktestResult;
  /**
   * Optional heading override / tag shown above `result.strategyName`,
   * e.g. "Baseline" or "Your pick" - lets callers visually distinguish
   * multiple cards shown side by side (comparison view).
   */
  label?: string;
  /**
   * Buy & Hold (the baseline) has its own BaselineResultCard now, so this
   * is only ever the challenger in practice - kept as a variant rather than
   * removed outright in case another non-baseline "kind" of card shows up
   * later.
   */
  variant?: 'challenger';
}

export function BacktestResultCard({ result, label, variant }: BacktestResultCardProps) {
  const borderClasses = variant === 'challenger' ? 'border-accent/30' : 'border-border';

  return (
    <div className={`${cardClasses} ${borderClasses}`}>
      {label && (
        <span className="mb-1 block text-xs font-semibold uppercase tracking-wide text-text/70">
          {label}
        </span>
      )}
      <h3 className="mb-4 text-lg font-semibold text-text-strong">{result.strategyName}</h3>

      <div className="grid gap-y-2 gap-x-10">
        <div className="flex items-center justify-between gap-3">
          <span className="text-text">Balance</span>
          <span className="text-sm text-text-strong">
            {formatCurrency(result.startingBalance)} &rarr; {formatCurrency(result.endingBalance)} |{' '}
            <span className={result.profit >= 0 ? 'font-medium text-gain' : 'font-medium text-loss'}>
              {formatCurrency(result.profit)} ({formatPercentage(result.profitPercentage)})
            </span>
          </span>
        </div>
        <div className="flex items-center justify-between gap-3">
          <span className="text-text">Operations</span>
          <span
            className={
              result.gains >= result.losses ? 'font-medium text-gain' : 'font-medium text-loss'
            }
          >
            T: {result.totalOperations} G: {result.gains} / L: {result.losses} ({formatPercentage(result.winRate)})
          </span>
        </div>
        <div className="flex items-center justify-between gap-3">
          <span className="text-text">Max drawdown</span>
          <span className="text-text-strong">
            {formatCurrency(result.maxDrawdownAmount)} ({formatPercentage(result.maxDrawdownPercentage)})
          </span>
        </div>
      </div>

      {result.operations && result.operations.length > 0 && (
        <details className="mt-4">
          <summary className="-mx-1.5 cursor-pointer select-none rounded-lg px-1.5 py-1 font-semibold text-text-strong transition-colors duration-150 hover:bg-accent-soft">
            Operations ({result.operations.length})
          </summary>
          <table className="mt-2 w-full border-collapse text-xs">
            <thead className="bg-raised">
              <tr>
                <th className="border border-border px-2 py-1 text-left">#</th>
                <th className="border border-border px-2 py-1 text-right">Buy date</th>
                <th className="border border-border px-2 py-1 text-right">Buy price</th>
                <th className="border border-border px-2 py-1 text-right">Sell date</th>
                <th className="border border-border px-2 py-1 text-right">Sell price</th>
                <th className="border border-border px-2 py-1 text-right">Days</th>
                <th className="border border-border px-2 py-1 text-right">Qty</th>
                <th className="border border-border px-2 py-1 text-right">Profit</th>
              </tr>
            </thead>
            <tbody>
              {result.operations.map((op, index) => {
                // Buy and sell quantity are always equal - partial exits
                // aren't supported - so there's a single Qty column, same
                // as the CLI's operations table. Profit isn't sent by the
                // API per-operation, so it's derived the same way the
                // backend does: prices are converted to integer cents
                // first, then (sellCents - buyCents) * quantity is computed
                // in integer cents, matching Operation.Profit()'s
                // sellTotal_cents - buyTotal_cents exactly. Subtracting the
                // already-rounded major-unit floats first (as before) is
                // exposed to float64 rounding/cancellation error that the
                // backend's integer math never has, so it's converted back
                // to major units only at the very end, for display.
                const buyCents = Math.round(op.buyOrder.price * 100);
                const sellCents = Math.round(op.sellOrder.price * 100);
                const profitCents = (sellCents - buyCents) * op.buyOrder.quantity;
                const profit = profitCents / 100;
                return (
                  <tr key={`${op.buyOrder.date}-${op.sellOrder.date}-${index}`}>
                    <td className="border border-border px-2 py-1 text-left">{index + 1}</td>
                    <td className="border border-border px-2 py-1 text-right">{op.buyOrder.date}</td>
                    <td className="border border-border px-2 py-1 text-right">
                      {formatCurrency(op.buyOrder.price)}
                    </td>
                    <td className="border border-border px-2 py-1 text-right">{op.sellOrder.date}</td>
                    <td className="border border-border px-2 py-1 text-right">
                      {formatCurrency(op.sellOrder.price)}
                    </td>
                    <td className="border border-border px-2 py-1 text-right">{op.days}</td>
                    <td className="border border-border px-2 py-1 text-right">{op.buyOrder.quantity}</td>
                    <td
                      className={`border border-border px-2 py-1 text-right font-medium ${profit >= 0 ? 'text-gain' : 'text-loss'}`}
                    >
                      {formatCurrency(profit)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </details>
      )}

      {result.operations && result.operations.length === 0 && (
        <p className="mt-4 text-sm text-text">No operations were recorded.</p>
      )}
    </div>
  );
}
