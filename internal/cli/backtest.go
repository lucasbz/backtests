package cli

import (
	"flag"
	"fmt"
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

	params, err := resolveParams(fs, *configPath, *asset, *start, *end, *balance, *strategyName, *verbose)
	if err != nil {
		return err
	}

	if params.Asset == "" || params.Start == "" || params.End == "" || params.Strategy == "" || params.Balance == "" {
		fs.Usage()
		return fmt.Errorf("-asset, -start, -end, -balance and -strategy are all required")
	}

	result, startDate, endDate, err := executeBacktest(params, params.Strategy, params.StrategyParams)
	if err != nil {
		return err
	}

	printResult(params.Asset, startDate, endDate, result, params.Verbose)
	return nil
}

// executeBacktest runs a single backtest from a resolvedParams (Asset,
// Start, End, Balance) plus an explicit strategy name/params - separate from
// params.Strategy/params.StrategyParams so runCompare can override them for
// its fixed Buy & Hold baseline while reusing the same resolved
// asset/dates/balance. Returns the parsed start/end dates alongside the
// result since callers print both.
func executeBacktest(params resolvedParams, strategyName string, strategyParams map[string]float64) (*backtest.BacktestResult, time.Time, time.Time, error) {
	startingBalance, err := util.ParsePositiveMoney(params.Balance, domain.Currency, "balance must be greater than zero")
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	strategy, err := strategies.LoadStrategy(strategyName, strategyParams)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	startDate, endDate, err := util.ParseDateRange(params.Start, params.End)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	bt := &backtest.Backtest{
		Asset:    params.Asset,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: strategy,
	}

	result, err := bt.Run()
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	return result, startDate, endDate, nil
}
