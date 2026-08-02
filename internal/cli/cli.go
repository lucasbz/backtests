package cli

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

func RunCLI(args []string) error {
	switch args[0] {
	case "backtest":
		return RunBacktest(args[1:])
	case "compare":
		return RunCompare(args[1:])
	case "scan":
		return RunScan(args[1:])
	case "info":
		return RunInfo(args[1:])
	default:
		return fmt.Errorf("unknown command %q, want backtest, compare, scan, info or serve", args[0])
	}
}

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

const baselineStrategyName = "buy-and-hold"

func RunCompare(args []string) error {
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
func printComparisonSummary(baseline, challenger *backtest.BacktestResult) {
	diff := challenger.ProfitPercentage() - baseline.ProfitPercentage()
	switch {
	case diff > 0:
		fmt.Printf("Result: Challenger outperformed baseline by %.2f percentage points\n", diff)
	case diff < 0:
		fmt.Printf("Result: Baseline outperformed challenger by %.2f percentage points\n", -diff)
	default:
		fmt.Println("Result: tied")
	}
}

// RunScan runs a single "challenger" strategy against every available
// asset (optionally restricted to -year), each compared against a Buy &
// Hold baseline, via backtest.Scan - the CLI counterpart of
// POST /api/scan, and the batch/many-assets sibling of runCompare (which
// does the same comparison for a single asset). Flags mirror runCompare's,
// minus -asset (a scan runs every asset, not one) plus -year.
func RunScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)

	start := fs.String("start", "", "timeframe start date (YYYY-MM-DD)")
	end := fs.String("end", "", "timeframe end date (YYYY-MM-DD)")
	balance := fs.String("balance", "", "starting cash balance (e.g. 10000.00)")
	verbose := fs.Bool("v", false, "also print each buy/sell operation for every asset the challenger actually traded on")
	year := fs.Int("year", 0, "restrict the scanned assets to those with imported data for this year (0 = every imported asset)")
	configPath := fs.String("config", "", configFlagUsage)

	strategyName := fs.String("strategy", "", fmt.Sprintf(
		"challenger strategy to run against the buy-and-hold baseline: %s", strategies.AvailableStrategyNames(),
	))
	if err := fs.Parse(args); err != nil {
		return err
	}

	startVal, endVal, balanceVal, strategyVal, verboseVal := *start, *end, *balance, *strategyName, *verbose
	var strategyParamsVal map[string]float64
	if *configPath != "" {
		cfg, err := loadConfig(*configPath)
		if err != nil {
			return err
		}
		// scan has no -asset flag; a dummy string satisfies
		// applyConfigDefaults' signature and any config file "asset" is
		// simply ignored.
		var unusedAsset string
		applyConfigDefaults(cfg, explicitFlags(fs), &unusedAsset, &startVal, &endVal, &balanceVal, &strategyVal, &verboseVal)
		strategyParamsVal = cfg.StrategyParams
	}

	if startVal == "" || endVal == "" || strategyVal == "" || balanceVal == "" {
		fs.Usage()
		return fmt.Errorf("-start, -end, -balance and -strategy are all required")
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

	// Validate before printing anything, so a bad -strategy doesn't first
	// print a misleading "Scanning N assets..." header.
	if _, err := strategies.LoadStrategy(strategyVal, strategyParamsVal); err != nil {
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

	assets, err := cotahist.ListAssets(*year)
	if err != nil {
		return err
	}

	fmt.Printf(
		"Scanning %d assets: %s vs Buy & Hold (%s to %s)...\n\n",
		len(assets), strategyVal, startVal, endVal,
	)

	results, err := backtest.Scan(backtest.ScanParams{
		Assets:         assets,
		Start:          startDate,
		End:            endDate,
		Balance:        startingBalance,
		StrategyName:   strategyVal,
		StrategyParams: strategyParamsVal,
	})
	if err != nil {
		return err
	}

	printScanResults(results, strategyVal, verboseVal)
	return nil
}

// printScanResults prints runScan's output: an aligned table of every
// successfully-scanned asset, a won/total summary, and, if any assets
// failed to load/run, an error list.
func printScanResults(results []backtest.ScanResult, strategyVal string, verbose bool) {
	var succeeded, won int
	var failed []backtest.ScanResult

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Asset\tBaseline%\tChallenger%\tDelta\tWon")
	for _, r := range results {
		if r.Err != nil {
			failed = append(failed, r)
			continue
		}
		succeeded++
		if r.Won {
			won++
		}

		wonStr := "no"
		if r.Won {
			wonStr = "yes"
		}
		fmt.Fprintf(
			w, "%s\t%.2f\t%.2f\t%+.2f\t%s\n",
			r.Asset, r.Baseline.ProfitPercentage(), r.Challenger.ProfitPercentage(), r.Delta, wonStr,
		)
	}
	w.Flush()

	fmt.Println()
	winRate := 0.0
	if succeeded > 0 {
		winRate = float64(won) / float64(succeeded) * 100
	}
	if len(failed) == 0 {
		fmt.Printf("Won on %d/%d assets (%.2f%%)\n", won, succeeded, winRate)
	} else {
		fmt.Printf(
			"Won on %d/%d assets (%.2f%%) - %d assets failed to load, see below:\n",
			won, succeeded, winRate, len(failed),
		)
		for _, r := range failed {
			fmt.Printf("  %s: %v\n", r.Asset, r.Err)
		}
	}

	if verbose {
		for _, r := range results {
			if r.Err != nil || r.Challenger == nil || r.Challenger.TotalOperations == 0 {
				continue
			}
			fmt.Printf("\n=== %s: %s operations ===\n", r.Asset, strategyVal)
			printOperations(r.Challenger.Operations)
		}
	}
}

func RunInfo(args []string) error {
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
