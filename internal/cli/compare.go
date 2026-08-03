package cli

import (
	"flag"
	"fmt"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
	"github.com/lucasbz/backtests/internal/util"
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

	if strategyVal == baselineStrategyName {
		return fmt.Errorf("-strategy cannot be %q: buy-and-hold is always run as the baseline, so comparing it against itself is meaningless", baselineStrategyName)
	}

	startingBalance, err := util.ParsePositiveMoney(balanceVal, domain.Currency, "balance must be greater than zero")
	if err != nil {
		return err
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

	startDate, endDate, err := util.ParseDateRange(startVal, endVal)
	if err != nil {
		return err
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
