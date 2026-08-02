package cli

import (
	"bytes"
	"fmt"
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
	t.Chdir("../..")
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

func TestRunCLI_Backtest_Valid(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_UnknownStrategy(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_MissingArgs(t *testing.T) {
	err := RunCLI([]string{"backtest", "-asset", "PETR4"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRunCLI_Backtest_MissingBalance(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_InvalidBalance(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_ZeroBalance(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_InvalidDate(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_InvalidEndDate(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Backtest_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"backtest", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// TestRunCLI_Backtest_LoadCandlesError chdirs into a temp directory that mimics
// the resources/cotahist layout LoadCandles reads from, but with a
// malformed year file, so bt.RunCLI() fails while loading candles and that
// error propagates out of runBacktest.
func TestRunCLI_Backtest_LoadCandlesError(t *testing.T) {
	dir := t.TempDir()
	tickerDir := filepath.Join(dir, "resources", "cotahist", "BADTICKER")
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tickerDir, "BADTICKER_2010.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Chdir(dir)

	err := RunCLI([]string{
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

func TestRunCLI_Backtest_Verbose(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
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

func TestRunCLI_Compare_Valid(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
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

func TestRunCLI_Compare_Verbose(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
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

func TestRunCLI_Compare_MissingArgs(t *testing.T) {
	err := RunCLI([]string{"compare", "-asset", "PETR4"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRunCLI_Compare_MissingBalance(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Compare_UnknownStrategy(t *testing.T) {
	err := RunCLI([]string{
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

// TestRunCLI_Compare_BuyAndHoldChallengerRejected asserts -strategy buy-and-hold
// is rejected for the challenger: Buy & Hold is always the fixed baseline
// (see runCompare), so comparing it against itself is meaningless, mirroring
// StrategyComparison.tsx's exclusion of it from the pickable strategy list.
func TestRunCLI_Compare_BuyAndHoldChallengerRejected(t *testing.T) {
	err := RunCLI([]string{
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

func TestRunCLI_Compare_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"compare", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// writeRealAssetYearFixture copies a single real <TICKER>_<YEAR>.json file
// into destDir/resources/cotahist/<TICKER>/. Used instead of
// chdirToRepoRoot for scan tests: captureStdout only drains its pipe after
// the command finishes, and a scan over the full 2000+-ticker universe
// would produce enough output to deadlock that write before it gets there.
func writeRealAssetYearFixture(t *testing.T, destDir, ticker string, year int) {
	t.Helper()

	// The test binary's cwd is internal/cli/ (this package's directory), so
	// "../.." is the repo root - same relative step chdirToRepoRoot takes,
	// just resolved to an absolute path before any t.Chdir happens.
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	filename := fmt.Sprintf("%s_%d.json", ticker, year)
	srcPath := filepath.Join(repoRoot, "resources", "cotahist", ticker, filename)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("reading real fixture %s: %v", srcPath, err)
	}

	dstDir := filepath.Join(destDir, "resources", "cotahist", ticker)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, filename), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRunCLI_Scan_Valid(t *testing.T) {
	dir := t.TempDir()
	tickers := []string{"PETR4", "VALE3", "ITUB4"}
	for _, ticker := range tickers {
		writeRealAssetYearFixture(t, dir, ticker, 2015)
	}
	t.Chdir(dir)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
			"scan",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "two-candle-breakout",
			"-balance", "10000.00",
			"-year", "2015",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	for _, want := range []string{
		fmt.Sprintf("Scanning %d assets", len(tickers)),
		"vs Buy & Hold", "Asset", "Baseline%", "Challenger%", "Delta", "Won",
		fmt.Sprintf("/%d assets (", len(tickers)),
	} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("output = %q, want it to contain %q", output, want)
		}
	}
	for _, ticker := range tickers {
		if !bytes.Contains([]byte(output), []byte(ticker)) {
			t.Errorf("output = %q, want it to contain %q", output, ticker)
		}
	}
}

// TestRunCLI_Scan_Verbose checks -v additionally prints a per-asset operations
// table for an asset the challenger actually traded on.
func TestRunCLI_Scan_Verbose(t *testing.T) {
	dir := t.TempDir()
	writeRealAssetYearFixture(t, dir, "PETR4", 2015)
	t.Chdir(dir)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
			"scan",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "two-candle-breakout",
			"-balance", "10000.00",
			"-year", "2015",
			"-v",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("=== PETR4: two-candle-breakout operations ===")) {
		t.Errorf("output = %q, want it to contain the PETR4 operations section header", output)
	}
	if !bytes.Contains([]byte(output), []byte("Buy date")) {
		t.Errorf("output = %q, want it to contain the operations table header", output)
	}
}

func TestRunCLI_Scan_MissingArgs(t *testing.T) {
	err := RunCLI([]string{"scan", "-start", "2015-01-02"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRunCLI_Scan_MissingBalance(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
	})
	if err == nil {
		t.Fatal("expected error for missing balance")
	}
}

func TestRunCLI_Scan_UnknownStrategy(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "does-not-exist",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

// TestRunCLI_Scan_BuyAndHoldChallengerRejected mirrors
// TestRunCLI_Compare_BuyAndHoldChallengerRejected: scan's baseline is always
// Buy & Hold, so it can't also be the challenger.
func TestRunCLI_Scan_BuyAndHoldChallengerRejected(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error when challenger strategy is buy-and-hold")
	}
}

func TestRunCLI_Scan_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"scan", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestRunCLI_Scan_InvalidDate(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "not-a-date",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid start date")
	}
}

func TestRunCLI_Scan_InvalidBalance(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "not-a-number",
	})
	if err == nil {
		t.Fatal("expected error for invalid balance")
	}
}

// TestRunCLI_Scan_YearFiltersAssetUniverse: PETR4 has both a 2015 and 2016
// file, VALE3 only 2016, so -year 2015 must scan just PETR4 while -year
// 2016 scans both.
func TestRunCLI_Scan_YearFiltersAssetUniverse(t *testing.T) {
	dir := t.TempDir()
	writeRealAssetYearFixture(t, dir, "PETR4", 2015)
	writeRealAssetYearFixture(t, dir, "PETR4", 2016)
	writeRealAssetYearFixture(t, dir, "VALE3", 2016)
	t.Chdir(dir)

	scanFor := func(year string) string {
		return captureStdout(t, func() {
			err := RunCLI([]string{
				"scan",
				"-start", "2015-01-02",
				"-end", "2016-12-30",
				"-strategy", "two-candle-breakout",
				"-balance", "10000.00",
				"-year", year,
			})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
		})
	}

	output2015 := scanFor("2015")
	if !bytes.Contains([]byte(output2015), []byte("Scanning 1 assets")) {
		t.Errorf("year=2015 output = %q, want it to contain %q", output2015, "Scanning 1 assets")
	}
	if !bytes.Contains([]byte(output2015), []byte("PETR4")) {
		t.Errorf("year=2015 output = %q, want it to contain PETR4", output2015)
	}
	if bytes.Contains([]byte(output2015), []byte("VALE3")) {
		t.Errorf("year=2015 output = %q, want it to NOT contain VALE3 (no 2015 data)", output2015)
	}

	output2016 := scanFor("2016")
	if !bytes.Contains([]byte(output2016), []byte("Scanning 2 assets")) {
		t.Errorf("year=2016 output = %q, want it to contain %q", output2016, "Scanning 2 assets")
	}
	for _, ticker := range []string{"PETR4", "VALE3"} {
		if !bytes.Contains([]byte(output2016), []byte(ticker)) {
			t.Errorf("year=2016 output = %q, want it to contain %q", output2016, ticker)
		}
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

func TestRunCLI_Backtest_Config_AllFieldsFromFile(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	output := captureStdout(t, func() {
		err := RunCLI([]string{"backtest", "-config", configPath})
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

// TestRunCLI_Backtest_Config_StrategyParams asserts a config file's
// "strategyParams" object is threaded into strategies.LoadStrategy,
// letting a parameterized strategy like "sma-crossover" (which requires
// "shortPeriod"/"longPeriod" - see internal/strategies/crossover.go) run
// successfully from -config alone, with no per-key CLI flag involved.
func TestRunCLI_Backtest_Config_StrategyParams(t *testing.T) {
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
		err := RunCLI([]string{"backtest", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("SMA Crossover")) {
		t.Errorf("output = %q, want it to contain the config file's strategy", output)
	}
}

// TestRunCLI_Backtest_Config_StrategyParams_MissingRequiredParam asserts a
// parameterized strategy's own validation still runs when its params come
// from the config file: sma-crossover requires both shortPeriod and
// longPeriod, so a config file that only sets one of them must fail with
// an error, not silently fall back to some default.
func TestRunCLI_Backtest_Config_StrategyParams_MissingRequiredParam(t *testing.T) {
	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "sma-crossover",
		"strategyParams": {"shortPeriod": 5}
	}`)

	err := RunCLI([]string{"backtest", "-config", configPath})
	if err == nil {
		t.Fatal("expected error for a strategyParams object missing a required key")
	}
}

// TestRunCLI_Backtest_Config_FlagOverridesFile passes -strategy explicitly
// alongside -config (whose strategy field names a different strategy), and
// asserts the explicit flag wins, proving flags always override the config
// file rather than the other way around.
func TestRunCLI_Backtest_Config_FlagOverridesFile(t *testing.T) {
	chdirToRepoRoot(t)

	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
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

func TestRunCLI_Backtest_Config_FileNotFound(t *testing.T) {
	err := RunCLI([]string{"backtest", "-config", "/does/not/exist/config.json"})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRunCLI_Backtest_Config_MalformedJSON(t *testing.T) {
	configPath := writeConfigFile(t, `{not valid json`)

	err := RunCLI([]string{"backtest", "-config", configPath})
	if err == nil {
		t.Fatal("expected error for malformed config file")
	}
}

// TestRunCLI_Backtest_Config_UnknownField asserts a typo'd field name (e.g.
// "assset" instead of "asset") is rejected with an error rather than
// silently ignored, which would otherwise leave that flag at its empty
// default with no clue why.
func TestRunCLI_Backtest_Config_UnknownField(t *testing.T) {
	configPath := writeConfigFile(t, `{
		"assset": "PETR4",
		"start": "2015-01-02",
		"end": "2015-12-30",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	err := RunCLI([]string{"backtest", "-config", configPath})
	if err == nil {
		t.Fatal("expected error for unknown config field")
	}
}

// TestRunCLI_Backtest_Config_VerboseFromFile asserts "verbose": true in the
// config file turns on the operations table just like -v would, when -v
// itself isn't passed on the command line.
func TestRunCLI_Backtest_Config_VerboseFromFile(t *testing.T) {
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
		err := RunCLI([]string{"backtest", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Operations (")) {
		t.Errorf("output = %q, want it to contain the operations table from the config file's verbose:true", output)
	}
}

// TestRunCLI_Backtest_Config_ExplicitFalseOverridesFileVerbose is the trickiest
// case in the -v/verbose merge: Go's flag.Visit reports a bool flag as
// visited even when it's explicitly set to its own zero value, so
// "-v=false" on the command line must still be treated as an explicit
// override of the config file's "verbose": true - not indistinguishable
// from -v simply not being passed at all.
func TestRunCLI_Backtest_Config_ExplicitFalseOverridesFileVerbose(t *testing.T) {
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
		err := RunCLI([]string{"backtest", "-config", configPath, "-v=false"})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if bytes.Contains([]byte(output), []byte("Operations (")) {
		t.Errorf("output = %q, want it to NOT contain the operations table since -v=false was passed explicitly", output)
	}
}

// TestRunCLI_Compare_Config_BuyAndHoldChallengerRejected asserts the
// baseline-collision check (see TestRunCLI_Compare_BuyAndHoldChallengerRejected)
// still applies when -strategy comes from the config file rather than the
// flag, proving the check runs against the merged effective value.
func TestRunCLI_Compare_Config_BuyAndHoldChallengerRejected(t *testing.T) {
	configPath := writeConfigFile(t, `{
		"asset": "PETR4",
		"start": "2010-01-01",
		"end": "2010-12-31",
		"balance": "10000.00",
		"strategy": "buy-and-hold"
	}`)

	err := RunCLI([]string{"compare", "-config", configPath})
	if err == nil {
		t.Fatal("expected error when config file's challenger strategy is buy-and-hold")
	}
}

// TestRunCLI_Compare_Config_StrategyParams mirrors
// TestRunCLI_Backtest_Config_StrategyParams for runCompare: the config file's
// "strategyParams" must reach the challenger's strategies.LoadStrategy
// call, not just runBacktest's.
func TestRunCLI_Compare_Config_StrategyParams(t *testing.T) {
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
		err := RunCLI([]string{"compare", "-config", configPath})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Challenger: sma-crossover")) {
		t.Errorf("output = %q, want it to contain the challenger label", output)
	}
}

func TestRunCLI_Info_MissingAsset(t *testing.T) {
	err := RunCLI([]string{"info"})
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}

func TestRunCLI_Info_UnknownAsset(t *testing.T) {
	err := RunCLI([]string{"info", "-asset", "DOESNOTEXIST9"})
	if err == nil {
		t.Fatal("expected error for unknown asset")
	}
}

func TestRunCLI_Info_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"info", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestRunCLI_Info_Valid(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := RunCLI([]string{"info", "-asset", "PETR4"})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("PETR4: data available from")) {
		t.Errorf("output = %q, want it to describe PETR4's available date range", output)
	}
}
