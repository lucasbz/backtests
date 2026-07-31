import type { BacktestResult } from '../api/client';
import { formatCurrency, formatPercentage } from '../utils/format';
import './BacktestResultCard.css';

export interface BacktestResultCardProps {
  result: BacktestResult;
  /**
   * Optional heading override / tag shown above `result.strategyName`,
   * e.g. "Baseline" or "Your pick" - lets callers visually distinguish
   * multiple cards shown side by side (comparison view).
   */
  label?: string;
  /** Adds a modifier class so baseline vs. challenger can be styled differently. */
  variant?: 'baseline' | 'challenger';
}

export function BacktestResultCard({ result, label, variant }: BacktestResultCardProps) {
  return (
    <div
      className={
        variant
          ? `backtest-result-card backtest-result-card--${variant}`
          : 'backtest-result-card'
      }
    >
      {label && <span className="backtest-result-card__label">{label}</span>}
      <h3>{result.strategyName}</h3>

      <div className="backtest-result-card__summary">
        <div className="backtest-result-card__stat">
          <span className="backtest-result-card__field-label">Balance</span>
          <span>{formatCurrency(result.startingBalance)} &rarr; {formatCurrency(result.endingBalance)}</span>
        </div>
        <div className="backtest-result-card__stat">
          <span className="backtest-result-card__field-label">Profit</span>
          <span
            className={
              result.profit >= 0 ? 'backtest-result-card__gain' : 'backtest-result-card__loss'
            }
          >
            {formatCurrency(result.profit)} ({formatPercentage(result.profitPercentage)})
          </span>
        </div>
        <div className="backtest-result-card__stat">
          <span className="backtest-result-card__field-label">Operations</span>
          <span
            className={
              result.gains >= result.losses ? 'backtest-result-card__gain' : 'backtest-result-card__loss'
            }
          >
            T: {result.totalOperations} G: {result.gains} / L: {result.losses}
          </span>
        </div>
        <div className="backtest-result-card__stat">
          <span className="backtest-result-card__field-label">Win rate</span>
          <span>{formatPercentage(result.winRate)}</span>
        </div>
      </div>

      {result.operations && result.operations.length > 0 && (
        <details className="backtest-result-card__operations-details">
          <summary>Operations ({result.operations.length})</summary>
          <table className="backtest-result-card__operations">
            <thead>
              <tr>
                <th>#</th>
                <th>Buy date</th>
                <th>Buy price</th>
                <th>Buy qty</th>
                <th>Sell date</th>
                <th>Sell price</th>
                <th>Sell qty</th>
              </tr>
            </thead>
            <tbody>
              {result.operations.map((op, index) => (
                <tr key={`${op.buyOrder.date}-${op.sellOrder.date}-${index}`}>
                  <td>{index + 1}</td>
                  <td>{op.buyOrder.date}</td>
                  <td>{formatCurrency(op.buyOrder.price)}</td>
                  <td>{op.buyOrder.quantity}</td>
                  <td>{op.sellOrder.date}</td>
                  <td>{formatCurrency(op.sellOrder.price)}</td>
                  <td>{op.sellOrder.quantity}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </details>
      )}

      {result.operations && result.operations.length === 0 && (
        <p className="backtest-result-card__hint">No operations were recorded.</p>
      )}
    </div>
  );
}
