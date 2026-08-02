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
		"-asset", "PETR4",
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
		"-asset", "PETR4",
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
	err := run([]string{"backtest", "-asset", "PETR4"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRun_Backtest_MissingBalance(t *testing.T) {
	err := run([]string{
		"backtest",
		"-asset", "PETR4",
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
		"-asset", "PETR4",
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
		"-asset", "PETR4",
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
		"-asset", "PETR4",
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
		"-asset", "PETR4",
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
		"-asset", "BADTICKER",
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
			"-asset", "PETR4",
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

	if !bytes.Contains([]byte(output), []byte("Operations (1):")) {
		t.Errorf("output = %q, want it to contain %q", output, "Operations (1):")
	}
	if !bytes.Contains([]byte(output), []byte("Buy date")) {
		t.Errorf("output = %q, want it to contain the operations table header", output)
	}
}

// TestPrintResult_Verbose exercises printResult directly with a synthetic
// result, so the verbose operations-printing branch doesn't depend on real
// data producing at least one operation.
func TestPrintResult_Verbose(t *testing.T) {
	result := &backtest.Result{
		StrategyName:      "Test Strategy",
		StartingBalance:   *money.New(10000, domain.Currency),
		EndingBalance:     *money.New(11000, domain.Currency),
		Profit:            *money.New(1000, domain.Currency),
		TotalOperations:   1,
		Gains:             1,
		Losses:            0,
		MaxDrawdownAmount: *money.New(0, domain.Currency),
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

	for _, want := range []string{"Operations (1):", "Buy date", "Sell date", "2020-01-01", "2020-02-01"} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("output = %q, want it to contain %q", output, want)
		}
	}
}

func TestRun_Compare_Valid(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := run([]string{
			"compare",
			"-asset", "PETR4",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "two-candle-breakout",
			"-balance", "10000.00",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Baseline: Buy & Hold")) {
		t.Errorf("output = %q, want it to contain the baseline label", output)
	}
	if !bytes.Contains([]byte(output), []byte("Challenger: two-candle-breakout")) {
		t.Errorf("output = %q, want it to contain the challenger label", output)
	}
	if !bytes.Contains([]byte(output), []byte("Result:")) {
		t.Errorf("output = %q, want it to contain a comparison summary line", output)
	}
}

func TestRun_Compare_Verbose(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := run([]string{
			"compare",
			"-asset", "PETR4",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "two-candle-breakout",
			"-balance", "10000.00",
			"-v",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Operations (")) {
		t.Errorf("output = %q, want it to contain the verbose operations header", output)
	}
	if !bytes.Contains([]byte(output), []byte("Buy date")) {
		t.Errorf("output = %q, want it to contain the operations table header", output)
	}
}

func TestRun_Compare_MissingArgs(t *testing.T) {
	err := run([]string{"compare", "-asset", "PETR4"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRun_Compare_MissingBalance(t *testing.T) {
	err := run([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
	})
	if err == nil {
		t.Fatal("expected error for missing balance")
	}
}

func TestRun_Compare_UnknownStrategy(t *testing.T) {
	err := run([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "does-not-exist",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

// TestRun_Compare_BuyAndHoldChallengerRejected asserts -strategy buy-and-hold
// is rejected for the challenger: Buy & Hold is always the fixed baseline
// (see runCompare), so comparing it against itself is meaningless, mirroring
// StrategyComparison.tsx's exclusion of it from the pickable strategy list.
func TestRun_Compare_BuyAndHoldChallengerRejected(t *testing.T) {
	err := run([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error when challenger strategy is buy-and-hold")
	}
}

func TestRun_Compare_InvalidFlag(t *testing.T) {
	err := run([]string{"compare", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// writeConfigFile writes contents as a JSON config file in t.TempDir() and
// returns its path.
func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestRun_Backtest_Config_AllFieldsFromFile(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	output := captureStdout(t, func() {
		err := run([]string{"backtest", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("PETR4")) {
		t.Errorf("output = %q, want it to contain the config file's asset", output)
	}
	if !bytes.Contains([]byte(output), []byte("Buy & Hold")) {
		t.Errorf("output = %q, want it to contain the config file's strategy", output)
	}
}

// TestRun_Backtest_Config_StrategyParams asserts a config file's
// "strategyParams" object is threaded into strategies.LoadStrategy,
// letting a parameterized strategy like "sma-crossover" (which requires
// "shortPeriod"/"longPeriod" - see internal/strategies/crossover.go) run
// successfully from -config alone, with no per-key CLI flag involved.
func TestRun_Backtest_Config_StrategyParams(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "sma-crossover",
		"strategyParams": {"shortPeriod": 5, "longPeriod": 20}
	}`)

	output := captureStdout(t, func() {
		err := run([]string{"backtest", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("SMA Crossover")) {
		t.Errorf("output = %q, want it to contain the config file's strategy", output)
	}
}

// TestRun_Backtest_Config_StrategyParams_MissingRequiredParam asserts a
// parameterized strategy's own validation still runs when its params come
// from the config file: sma-crossover requires both shortPeriod and
// longPeriod, so a config file that only sets one of them must fail with
// an error, not silently fall back to some default.
func TestRun_Backtest_Config_StrategyParams_MissingRequiredParam(t *testing.T) {
	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "sma-crossover",
		"strategyParams": {"shortPeriod": 5}
	}`)

	err := run([]string{"backtest", "-config", configPath})
	if err == nil {
		t.Fatal("expected error for a strategyParams object missing a required key")
	}
}

// TestRun_Backtest_Config_FlagOverridesFile passes -strategy explicitly
// alongside -config (whose strategy field names a different strategy), and
// asserts the explicit flag wins, proving flags always override the config
// file rather than the other way around.
func TestRun_Backtest_Config_FlagOverridesFile(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	output := captureStdout(t, func() {
		err := run([]string{
			"backtest",
			"-config", configPath,
			"-strategy", "two-candle-breakout",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Two-Candle Breakout")) {
		t.Errorf("output = %q, want it to contain the explicitly-passed strategy", output)
	}
	if bytes.Contains([]byte(output), []byte("Buy & Hold")) {
		t.Errorf("output = %q, want it to NOT contain the config file's strategy", output)
	}
}

func TestRun_Backtest_Config_FileNotFound(t *testing.T) {
	err := run([]string{"backtest", "-config", "/does/not/exist/config.json"})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_Backtest_Config_MalformedJSON(t *testing.T) {
	configPath := writeConfigFile(t, `{not valid json`)

	err := run([]string{"backtest", "-config", configPath})
	if err == nil {
		t.Fatal("expected error for malformed config file")
	}
}

// TestRun_Backtest_Config_UnknownField asserts a typo'd field name (e.g.
// "assset" instead of "asset") is rejected with an error rather than
// silently ignored, which would otherwise leave that flag at its empty
// default with no clue why.
func TestRun_Backtest_Config_UnknownField(t *testing.T) {
	configPath := writeConfigFile(t, `{
		"assset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	err := run([]string{"backtest", "-config", configPath})
	if err == nil {
		t.Fatal("expected error for unknown config field")
	}
}

// TestRun_Backtest_Config_VerboseFromFile asserts "verbose": true in the
// config file turns on the operations table just like -v would, when -v
// itself isn't passed on the command line.
func TestRun_Backtest_Config_VerboseFromFile(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold",
		"verbose": true
	}`)

	output := captureStdout(t, func() {
		err := run([]string{"backtest", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Operations (")) {
		t.Errorf("output = %q, want it to contain the operations table from the config file's verbose:true", output)
	}
}

// TestRun_Backtest_Config_ExplicitFalseOverridesFileVerbose is the trickiest
// case in the -v/verbose merge: Go's flag.Visit reports a bool flag as
// visited even when it's explicitly set to its own zero value, so
// "-v=false" on the command line must still be treated as an explicit
// override of the config file's "verbose": true - not indistinguishable
// from -v simply not being passed at all.
func TestRun_Backtest_Config_ExplicitFalseOverridesFileVerbose(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold",
		"verbose": true
	}`)

	output := captureStdout(t, func() {
		err := run([]string{"backtest", "-config", configPath, "-v=false"})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if bytes.Contains([]byte(output), []byte("Operations (")) {
		t.Errorf("output = %q, want it to NOT contain the operations table since -v=false was passed explicitly", output)
	}
}

// TestRun_Compare_Config_BuyAndHoldChallengerRejected asserts the
// baseline-collision check (see TestRun_Compare_BuyAndHoldChallengerRejected)
// still applies when -strategy comes from the config file rather than the
// flag, proving the check runs against the merged effective value.
func TestRun_Compare_Config_BuyAndHoldChallengerRejected(t *testing.T) {
	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2010-01-01",
		"end": "2010-12-31",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	err := run([]string{"compare", "-config", configPath})
	if err == nil {
		t.Fatal("expected error when config file's challenger strategy is buy-and-hold")
	}
}

// TestRun_Compare_Config_StrategyParams mirrors
// TestRun_Backtest_Config_StrategyParams for runCompare: the config file's
// "strategyParams" must reach the challenger's strategies.LoadStrategy
// call, not just runBacktest's.
func TestRun_Compare_Config_StrategyParams(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "sma-crossover",
		"strategyParams": {"shortPeriod": 5, "longPeriod": 20}
	}`)

	output := captureStdout(t, func() {
		err := run([]string{"compare", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Challenger: sma-crossover")) {
		t.Errorf("output = %q, want it to contain the challenger label", output)
	}
}

func TestRun_Info_MissingAsset(t *testing.T) {
	err := run([]string{"info"})
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}

func TestRun_Info_UnknownAsset(t *testing.T) {
	err := run([]string{"info", "-asset", "DOESNOTEXIST9"})
	if err == nil {
		t.Fatal("expected error for unknown asset")
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
		err := run([]string{"info", "-asset", "PETR4"})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("PETR4: data available from")) {
		t.Errorf("output = %q, want it to describe PETR4's available date range", output)
	}
}
