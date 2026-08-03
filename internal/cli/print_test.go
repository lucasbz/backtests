package cli

import (
	"bytes"
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/domain"
)

// TestPrintResult_Verbose exercises printResult directly with a synthetic
// result, so the verbose operations-printing branch doesn't depend on real
// data producing at least one operation. Built via backtest.NewBacktestResult
// (the only way to construct a valid BacktestResult - see its doc comment)
// rather than a BacktestResult{} struct literal, so EndingBalance/Profit/
// Gains/etc. are derived from operations instead of hand-typed values that
// could silently drift out of sync with them.
func TestPrintResult_Verbose(t *testing.T) {
	operations := []domain.Operation{
		{
			Date: "2020-01-01",
			BuyOrder: domain.Order{
				Date:      "2020-01-01",
				Price:     *money.New(1000, domain.Currency),
				Quantity:  10,
				OrderType: domain.Buy,
			},
			SellOrder: domain.Order{
				Date:      "2020-02-01",
				Price:     *money.New(1100, domain.Currency),
				Quantity:  10,
				OrderType: domain.Sell,
			},
		},
	}
	result, err := backtest.NewBacktestResult("Test Strategy", operations, *money.New(10000, domain.Currency))
	if err != nil {
		t.Fatalf("NewBacktestResult: %v", err)
	}

	start := mustParseDate(t, "2020-01-01")
	end := mustParseDate(t, "2020-02-01")

	output := captureStdout(t, func() {
		printResult("PETR4", start, end, result, true)
	})

	for _, want := range []string{"Operations (1):", "Buy date", "Sell date", "2020-01-01", "2020-02-01"} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("output = %q, want it to contain %q", output, want)
		}
	}
}
