package cli

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
	"github.com/lucasbz/backtests/internal/util"
)

func RunBacktest(args []string) error {
	fs := flag.NewFlagSet("backtest", flag.ContinueOnError)

	configPath := fs.String("config", "", configFlagUsage)
	asset := fs.String("asset", "", "asset (ticker) to backtest (e.g. PETR4)")
	start := fs.String("start", "", "timeframe start date (YYYY-MM-DD)")
	end := fs.String("end", "", "timeframe end date (YYYY-MM-DD)")
	balance := fs.String("balance", "", "starting cash balance (e.g. 10000.00)")
	verbose := fs.Bool("v", false, "print each buy/sell operation")
	strategyName := fs.String("strategy", "", fmt.Sprintf("strategy to run: %s", strategies.AvailableStrategyNames()))

	if err := fs.Parse(args); err != nil {
		return err
	}

	assetVal, startVal, endVal, balanceVal, strategyVal, verboseVal := *asset, *start, *end, *balance, *strategyName, *verbose
	var strategyParamsVal map[string]float64

	if *configPath != "" {
		cfg, err := loadConfig(*configPath)
		if err != nil {
			return err
		}
		assetVal, startVal, endVal, balanceVal, strategyVal = cfg.Asset, cfg.Start, cfg.End, cfg.Balance, cfg.Strategy
		strategyParamsVal = cfg.StrategyParams
		if !verboseWasSet(fs) {
			verboseVal = cfg.Verbose
		}
	}

	if assetVal == "" || startVal == "" || endVal == "" || strategyVal == "" || balanceVal == "" {
		fs.Usage()
		return fmt.Errorf("-asset, -start, -end, -balance and -strategy are all required")
	}

	startingBalance, err := util.ParsePositiveMoney(balanceVal, domain.Currency, "balance must be greater than zero")
	if err != nil {
		return err
	}

	newStrategy, err := strategies.LoadStrategy(strategyVal, strategyParamsVal)
	if err != nil {
		return err
	}

	startDate, endDate, err := util.ParseDateRange(startVal, endVal)
	if err != nil {
		return err
	}

	bt := &backtest.Backtest{
		Asset:    assetVal,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: newStrategy,
	}

	result, err := bt.Run()
	if err != nil {
		return err
	}

	printResult(assetVal, startDate, endDate, result, verboseVal)
	return nil
}

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
