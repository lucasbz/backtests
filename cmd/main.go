package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lucasbz/backtests/internal/api"
	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: backtests <backtest|info|serve> [flags]")
	}

	switch args[0] {
	case "backtest":
		return runBacktest(args[1:])
	case "info":
		return runInfo(args[1:])
	case "serve":
		return runServe(args[1:])
	default:
		return fmt.Errorf("unknown command %q, want backtest, info or serve", args[0])
	}
}

func runBacktest(args []string) error {
	fs := flag.NewFlagSet("backtest", flag.ContinueOnError)

	ticker := fs.String("ticker", "", "ticker to backtest (e.g. PETR4)")
	start := fs.String("start", "", "timeframe start date (YYYY-MM-DD)")
	end := fs.String("end", "", "timeframe end date (YYYY-MM-DD)")
	balance := fs.String("balance", "", "starting cash balance (e.g. 10000.00)")
	verbose := fs.Bool("v", false, "print each buy/sell operation")

	strategyName := fs.String("strategy", "", fmt.Sprintf("strategy to run: %s", strategies.AvailableStrategyNames()))
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *ticker == "" || *start == "" || *end == "" || *strategyName == "" || *balance == "" {
		fs.Usage()
		return fmt.Errorf("-ticker, -start, -end, -balance and -strategy are all required")
	}

	startingBalance, err := domain.ParseMoney(*balance)
	if err != nil {
		return fmt.Errorf("parsing -balance: %w", err)
	}
	if !startingBalance.IsPositive() {
		return fmt.Errorf("-balance must be greater than zero, got %q", *balance)
	}

	newStrategy, err := strategies.LoadStrategy(*strategyName, startingBalance)
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
		Ticker:   *ticker,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: newStrategy,
	}

	result, err := bt.Run()
	if err != nil {
		return err
	}

	printResult(*ticker, startDate, endDate, result, *verbose)
	return nil
}

func printResult(ticker string, start, end time.Time, result *backtest.Result, verbose bool) {
	fmt.Printf(
		"Running Backtest for: %s %s to %s | Strategy: %s | Balance: %s -> %s (Profit: %s, %.2f%%) | Gains: %d Losses: %d (Win rate: %.2f%%)\n",
		ticker, start.Format("2006-01-02"), end.Format("2006-01-02"), result.StrategyName,
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
	ticker := fs.String("ticker", "", "ticker to look up (e.g. PETR4)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *ticker == "" {
		fs.Usage()
		return fmt.Errorf("-ticker is required")
	}

	earliest, latest, err := cotahist.DateRange(*ticker)
	if err != nil {
		return err
	}

	fmt.Printf("%s: data available from %s to %s\n", *ticker, earliest, latest)
	return nil
}

// runServe starts an HTTP server exposing the backtest/info commands as a
// JSON API (see internal/api and openapi.yaml). It runs until the server exits
// with an error (e.g. the address is already in use); it never exits on
// its own otherwise.
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "address to listen on (e.g. :8080)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("listening on %s\n", *addr)
	return http.ListenAndServe(*addr, api.NewHandler())
}
