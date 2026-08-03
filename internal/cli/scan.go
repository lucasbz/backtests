package cli

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
	"github.com/lucasbz/backtests/internal/util"
)

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
		// scan has no -asset flag; any config file "asset" is simply not
		// consulted.
		startVal, endVal, balanceVal, strategyVal = cfg.Start, cfg.End, cfg.Balance, cfg.Strategy
		strategyParamsVal = cfg.StrategyParams
		if !verboseWasSet(fs) {
			verboseVal = cfg.Verbose
		}
	}

	if startVal == "" || endVal == "" || strategyVal == "" || balanceVal == "" {
		fs.Usage()
		return fmt.Errorf("-start, -end, -balance and -strategy are all required")
	}

	if strategyVal == baselineStrategyName {
		return fmt.Errorf("-strategy cannot be %q: buy-and-hold is always run as the baseline, so comparing it against itself is meaningless", baselineStrategyName)
	}

	startingBalance, err := util.ParsePositiveMoney(balanceVal, domain.Currency, "balance must be greater than zero")
	if err != nil {
		return err
	}

	// Validate before printing anything, so a bad -strategy doesn't first
	// print a misleading "Scanning N assets..." header.
	if _, err := strategies.LoadStrategy(strategyVal, strategyParamsVal); err != nil {
		return err
	}

	startDate, endDate, err := util.ParseDateRange(startVal, endVal)
	if err != nil {
		return err
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
