package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/domain"
)

// mustParseDate parses a YYYY-MM-DD date the same way runBacktest does,
// failing the test on error instead of returning it.
func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("time.Parse(%q): %v", s, err)
	}
	return d
}

// chdirToRepoRoot points the test's working directory at the repo root, so
// cotahist.DateRange/LoadCandles (which read from the fixed relative path
// resources/cotahist) can find the real imported data checked into the repo.
// Mirrors the identically-named helper in internal/api/api_test.go.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	t.Chdir("..")
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured output: %v", err)
	}
	return buf.String()
}

func TestRun_Backtest_Valid(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRun_Backtest_UnknownStrategy(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "does-not-exist",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestRun_Backtest_MissingArgs(t *testing.T) {
	err := run([]string{"backtest", "-ticker", "PETR4"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRun_Backtest_MissingBalance(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
	})
	if err == nil {
		t.Fatal("expected error for missing balance")
	}
}

func TestRun_Backtest_InvalidBalance(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "not-a-number",
	})
	if err == nil {
		t.Fatal("expected error for invalid balance")
	}
}

func TestRun_Backtest_ZeroBalance(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "0",
	})
	if err == nil {
		t.Fatal("expected error for zero balance")
	}
}

func TestRun_Backtest_InvalidDate(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "not-a-date",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid start date")
	}
}

func TestRun_Backtest_InvalidEndDate(t *testing.T) {
	err := run([]string{
		"backtest",
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "not-a-date",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid end date")
	}
}

func TestRun_Backtest_InvalidFlag(t *testing.T) {
	err := run([]string{"backtest", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// TestRun_Backtest_LoadCandlesError chdirs into a temp directory that mimics
// the resources/cotahist layout LoadCandles reads from, but with a
// malformed year file, so bt.Run() fails while loading candles and that
// error propagates out of runBacktest.
func TestRun_Backtest_LoadCandlesError(t *testing.T) {
	dir := t.TempDir()
	tickerDir := filepath.Join(dir, "resources", "cotahist", "BADTICKER")
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tickerDir, "BADTICKER_2010.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Chdir(dir)

	err := run([]string{
		"backtest",
		"-ticker", "BADTICKER",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for malformed candle data")
	}
}

func TestRun_Backtest_Verbose(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := run([]string{
			"backtest",
			"-ticker", "PETR4",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "buy-and-hold",
			"-balance", "10000.00",
			"-v",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Operations:")) {
		t.Errorf("output = %q, want it to contain %q", output, "Operations:")
	}
	if !bytes.Contains([]byte(output), []byte("BUY")) {
		t.Errorf("output = %q, want it to contain a BUY line", output)
	}
	if !bytes.Contains([]byte(output), []byte("SELL")) {
		t.Errorf("output = %q, want it to contain a SELL line", output)
	}
}

// TestPrintResult_Verbose exercises printResult directly with a synthetic
// result, so the verbose operations-printing branch doesn't depend on real
// data producing at least one operation.
func TestPrintResult_Verbose(t *testing.T) {
	result := &backtest.Result{
		StrategyName:    "Test Strategy",
		StartingBalance: *money.New(10000, domain.Currency),
		EndingBalance:   *money.New(11000, domain.Currency),
		Profit:          *money.New(1000, domain.Currency),
		TotalOperations: 1,
		Gains:           1,
		Losses:          0,
		Operations: []domain.Operation{
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
		},
	}

	start := mustParseDate(t, "2020-01-01")
	end := mustParseDate(t, "2020-02-01")

	output := captureStdout(t, func() {
		printResult("PETR4", start, end, result, true)
	})

	for _, want := range []string{"Operations:", "BUY", "SELL", "2020-01-01", "2020-02-01"} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("output = %q, want it to contain %q", output, want)
		}
	}
}

func TestRun_Info_MissingTicker(t *testing.T) {
	err := run([]string{"info"})
	if err == nil {
		t.Fatal("expected error for missing ticker")
	}
}

func TestRun_Info_UnknownTicker(t *testing.T) {
	err := run([]string{"info", "-ticker", "DOESNOTEXIST9"})
	if err == nil {
		t.Fatal("expected error for unknown ticker")
	}
}

func TestRun_Info_InvalidFlag(t *testing.T) {
	err := run([]string{"info", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestRun_Info_Valid(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := run([]string{"info", "-ticker", "PETR4"})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("PETR4: data available from")) {
		t.Errorf("output = %q, want it to describe PETR4's available date range", output)
	}
}
