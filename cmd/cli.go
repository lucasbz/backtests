package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
)

func runBacktest(args []string) error {
	fs := flag.NewFlagSet("backtest", flag.ContinueOnError)

	asset := fs.String("asset", "", "asset (ticker) to backtest (e.g. PETR4)")
	start := fs.String("start", "", "timeframe start date (YYYY-MM-DD)")
	end := fs.String("end", "", "timeframe end date (YYYY-MM-DD)")
	balance := fs.String("balance", "", "starting cash balance (e.g. 10000.00)")
	verbose := fs.Bool("v", false, "print each buy/sell operation")

	strategyName := fs.String("strategy", "", fmt.Sprintf("strategy to run: %s", strategies.AvailableStrategyNames()))
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *asset == "" || *start == "" || *end == "" || *strategyName == "" || *balance == "" {
		fs.Usage()
		return fmt.Errorf("-asset, -start, -end, -balance and -strategy are all required")
	}

	startingBalance, err := domain.ParseMoney(*balance)
	if err != nil {
		return fmt.Errorf("parsing -balance: %w", err)
	}
	if !startingBalance.IsPositive() {
		return fmt.Errorf("-balance must be greater than zero, got %q", *balance)
	}

	newStrategy, err := strategies.LoadStrategy(*strategyName)
	if err != nil {
		return err
	}

	startDate, err := time.Parse("2006-01-02", *start)
	if err != nil {
		return fmt.Errorf("parsing -start: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", *end)
	if err != nil {
		return fmt.Errorf("parsing -end: %w", err)
	}

	bt := &backtest.Backtest{
		Asset:    *asset,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: newStrategy,
	}

	result, err := bt.Run()
	if err != nil {
		return err
	}

	printResult(*asset, startDate, endDate, result, *verbose)
	return nil
}

func printResult(asset string, start, end time.Time, result *backtest.Result, verbose bool) {
	fmt.Printf(
		"Running Backtest for: %s %s to %s | Strategy: %s | Balance: %s -> %s (Profit: %s, %.2f%%) | Gains: %d Losses: %d (Win rate: %.2f%%)\n",
		asset, start.Format("2006-01-02"), end.Format("2006-01-02"), result.StrategyName,
		result.StartingBalance.Display(), result.EndingBalance.Display(), result.Profit.Display(), result.ProfitPercentage,
		result.Gains, result.Losses, result.WinRate,
	)

	if verbose {
		fmt.Println("Operations:")
		for _, op := range result.Operations {
			fmt.Printf("  BUY  %s @ %s x%d\n", op.BuyOrder.Date, op.BuyOrder.Price.Display(), op.BuyOrder.Quantity)
			fmt.Printf("  SELL %s @ %s x%d\n", op.SellOrder.Date, op.SellOrder.Price.Display(), op.SellOrder.Quantity)
		}
	}
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	asset := fs.String("asset", "", "asset (ticker) to look up (e.g. PETR4)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *asset == "" {
		fs.Usage()
		return fmt.Errorf("-asset is required")
	}

	earliest, latest, err := cotahist.DateRange(*asset)
	if err != nil {
		return err
	}

	fmt.Printf("%s: data available from %s to %s\n", *asset, earliest, latest)
	return nil
}
