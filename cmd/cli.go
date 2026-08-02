package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
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
	configPath := fs.String("config", "", configFlagUsage)

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
		applyConfigDefaults(cfg, explicitFlags(fs), &assetVal, &startVal, &endVal, &balanceVal, &strategyVal, &verboseVal)
		strategyParamsVal = cfg.StrategyParams
	}

	if assetVal == "" || startVal == "" || endVal == "" || strategyVal == "" || balanceVal == "" {
		fs.Usage()
		return fmt.Errorf("-asset, -start, -end, -balance and -strategy are all required")
	}

	startingBalance, err := domain.ParseMoney(balanceVal)
	if err != nil {
		return fmt.Errorf("parsing -balance: %w", err)
	}
	if !startingBalance.IsPositive() {
		return fmt.Errorf("-balance must be greater than zero, got %q", balanceVal)
	}

	newStrategy, err := strategies.LoadStrategy(strategyVal, strategyParamsVal)
	if err != nil {
		return err
	}

	startDate, err := time.Parse("2006-01-02", startVal)
	if err != nil {
		return fmt.Errorf("parsing -start: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", endVal)
	if err != nil {
		return fmt.Errorf("parsing -end: %w", err)
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

func printResult(asset string, start, end time.Time, result *backtest.Result, verbose bool) {
	fmt.Printf(
		"Running Backtest for: %s %s to %s | Strategy: %s | Balance: %s -> %s (Profit: %s, %.2f%%) | G/L/T: %d/%d/%d (WR: %.2f%%) | Max DD: %s (%.2f%%)\n",
		asset, start.Format("2006-01-02"), end.Format("2006-01-02"), result.StrategyName,
		result.StartingBalance.Display(), result.EndingBalance.Display(), result.Profit.Display(), result.ProfitPercentage,
		result.Gains, result.Losses, result.TotalOperations, result.WinRate,
		result.MaxDrawdownAmount.Display(), result.MaxDrawdownPercentage,
	)

	if verbose {
		printOperations(result.Operations)
	}
}

// printOperations prints one row per operation as an aligned table (# | buy
// date/price | sell date/price | days held | qty | profit). There's a
// single Qty column rather than separate buy/sell quantities since partial
// exits aren't supported - buy and sell quantity are always equal for an
// operation. Profit's sign alone conveys gain/loss, so there's no separate
// outcome column either.
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

// baselineStrategyName is the -strategy value reserved for the fixed
// baseline runCompare always runs, mirroring
// frontend/src/components/StrategyComparison.tsx's BUY_AND_HOLD constant.
const baselineStrategyName = "buy-and-hold"

// runCompare runs two backtests over the same asset/timeframe/balance: a
// fixed Buy & Hold baseline, and a user-chosen "challenger" strategy, then
// prints both results side by side along with a one-line summary of which
// one won. It's the CLI counterpart of StrategyComparison.tsx in the web
// frontend.
func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)

	asset := fs.String("asset", "", "asset (ticker) to backtest (e.g. PETR4)")
	start := fs.String("start", "", "timeframe start date (YYYY-MM-DD)")
	end := fs.String("end", "", "timeframe end date (YYYY-MM-DD)")
	balance := fs.String("balance", "", "starting cash balance (e.g. 10000.00)")
	verbose := fs.Bool("v", false, "print each buy/sell operation")
	configPath := fs.String("config", "", configFlagUsage)

	strategyName := fs.String("strategy", "", fmt.Sprintf(
		"challenger strategy to run against the buy-and-hold baseline: %s", strategies.AvailableStrategyNames(),
	))
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
		applyConfigDefaults(cfg, explicitFlags(fs), &assetVal, &startVal, &endVal, &balanceVal, &strategyVal, &verboseVal)
		strategyParamsVal = cfg.StrategyParams
	}

	if assetVal == "" || startVal == "" || endVal == "" || strategyVal == "" || balanceVal == "" {
		fs.Usage()
		return fmt.Errorf("-asset, -start, -end, -balance and -strategy are all required")
	}

	if strategyVal == baselineStrategyName {
		return fmt.Errorf("-strategy cannot be %q: buy-and-hold is always run as the baseline, so comparing it against itself is meaningless", baselineStrategyName)
	}

	startingBalance, err := domain.ParseMoney(balanceVal)
	if err != nil {
		return fmt.Errorf("parsing -balance: %w", err)
	}
	if !startingBalance.IsPositive() {
		return fmt.Errorf("-balance must be greater than zero, got %q", balanceVal)
	}

	// The baseline is always plain Buy & Hold, which ignores params, so it
	// never needs strategyParamsVal (that's only meaningful for the
	// challenger, named by -strategy/the config file's "strategy" field).
	baselineStrategy, err := strategies.LoadStrategy(baselineStrategyName, nil)
	if err != nil {
		return err
	}
	challengerStrategy, err := strategies.LoadStrategy(strategyVal, strategyParamsVal)
	if err != nil {
		return err
	}

	startDate, err := time.Parse("2006-01-02", startVal)
	if err != nil {
		return fmt.Errorf("parsing -start: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", endVal)
	if err != nil {
		return fmt.Errorf("parsing -end: %w", err)
	}

	baselineBT := &backtest.Backtest{
		Asset:    assetVal,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: baselineStrategy,
	}
	baselineResult, err := baselineBT.Run()
	if err != nil {
		return err
	}

	challengerBT := &backtest.Backtest{
		Asset:    assetVal,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: challengerStrategy,
	}
	challengerResult, err := challengerBT.Run()
	if err != nil {
		return err
	}

	fmt.Println("=== Baseline: Buy & Hold ===")
	printResult(assetVal, startDate, endDate, baselineResult, verboseVal)

	fmt.Printf("=== Challenger: %s ===\n", strategyVal)
	printResult(assetVal, startDate, endDate, challengerResult, verboseVal)

	fmt.Println()
	printComparisonSummary(baselineResult, challengerResult)

	return nil
}

// printComparisonSummary prints a single line calling out which of the two
// results won, and by how much, comparing ProfitPercentage (rather than raw
// EndingBalance) since both runs share the same starting balance and
// ProfitPercentage is what's already displayed per-result by printResult,
// so the summary reads consistently with the two lines above it.
func printComparisonSummary(baseline, challenger *backtest.Result) {
	diff := challenger.ProfitPercentage - baseline.ProfitPercentage
	switch {
	case diff > 0:
		fmt.Printf("Result: Challenger outperformed baseline by %.2f percentage points\n", diff)
	case diff < 0:
		fmt.Printf("Result: Baseline outperformed challenger by %.2f percentage points\n", -diff)
	default:
		fmt.Println("Result: tied")
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
