package cli

import (
	"flag"
	"fmt"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/strategies"
)

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

	params, err := resolveParams(fs, *configPath, *asset, *start, *end, *balance, *strategyName, *verbose)
	if err != nil {
		return err
	}

	if params.Asset == "" || params.Start == "" || params.End == "" || params.Strategy == "" || params.Balance == "" {
		fs.Usage()
		return fmt.Errorf("-asset, -start, -end, -balance and -strategy are all required")
	}

	if params.Strategy == baselineStrategyName {
		return fmt.Errorf("-strategy cannot be %q: buy-and-hold is always run as the baseline, so comparing it against itself is meaningless", baselineStrategyName)
	}

	// The baseline is always plain Buy & Hold, which ignores params, so it
	// never needs params.StrategyParams (that's only meaningful for the
	// challenger, named by -strategy/the config file's "strategy" field).
	baselineResult, startDate, endDate, err := executeBacktest(params, baselineStrategyName, nil)
	if err != nil {
		return err
	}

	challengerResult, _, _, err := executeBacktest(params, params.Strategy, params.StrategyParams)
	if err != nil {
		return err
	}

	fmt.Println("=== Baseline: Buy & Hold ===")
	printResult(params.Asset, startDate, endDate, baselineResult, params.Verbose)

	fmt.Printf("=== Challenger: %s ===\n", params.Strategy)
	printResult(params.Asset, startDate, endDate, challengerResult, params.Verbose)

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
