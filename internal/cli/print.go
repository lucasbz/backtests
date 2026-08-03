package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/domain"
)

func printResult(asset string, start, end time.Time, result *backtest.BacktestResult, verbose bool) {
	maxDrawdownAmount := result.MaxDrawdownAmount()
	fmt.Printf(
		"Running Backtest for: %s %s to %s | Strategy: %s | Balance: %s -> %s (Profit: %s, %.2f%%) | G/L/T: %d/%d/%d (WR: %.2f%%) | Max DD: %s (%.2f%%)\n",
		asset, start.Format("2006-01-02"), end.Format("2006-01-02"), result.StrategyName,
		result.StartingBalance.Display(), result.EndingBalance.Display(), result.Profit.Display(), result.ProfitPercentage(),
		result.Gains, result.Losses, result.TotalOperations, result.WinRate(),
		maxDrawdownAmount.Display(), result.MaxDrawdownPercentage(),
	)

	if verbose {
		printOperations(result.Operations)
	}
}

func printOperations(operations []domain.Operation) {
	if len(operations) == 0 {
		fmt.Println("Operations: none")
		return
	}

	fmt.Printf("Operations (%d):\n", len(operations))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tBuy date\tBuy price\tSell date\tSell price\tDays\tQty\tProfit")
	for i, op := range operations {
		profitStr := "?"
		if profit, err := op.Profit(); err == nil {
			profitStr = profit.Display()
		}

		daysStr := "?"
		if days, err := op.Days(); err == nil {
			daysStr = fmt.Sprintf("%d", days)
		}

		fmt.Fprintf(
			w, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			i+1,
			op.BuyOrder.Date, op.BuyOrder.Price.Display(),
			op.SellOrder.Date, op.SellOrder.Price.Display(),
			daysStr, op.BuyOrder.Quantity, profitStr,
		)
	}
	w.Flush()
}
