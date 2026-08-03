package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

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

// TestRunCLI_Backtest_EndBeforeStart locks in a gap util.ParseDateRange
// closed: runBacktest previously parsed -start/-end independently with no
// ordering check, so a swapped date pair silently ran a backtest over an
// unintended/empty range instead of being rejected.
func TestRunCLI_Backtest_EndBeforeStart(t *testing.T) {
	err := RunCLI([]string{
		"backtest",
		"-asset", "PETR4",
		"-start", "2010-12-31",
		"-end", "2010-01-01",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for end before start")
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

// TestRunCLI_Backtest_Config_IgnoresOtherFlags passes -strategy explicitly
// alongside -config (whose strategy field names a different strategy), and
// asserts the config file wins: -config is all-or-nothing, so every other
// flag except -v is ignored once it's given, rather than being merged in
// as a per-flag override.
func TestRunCLI_Backtest_Config_IgnoresOtherFlags(t *testing.T) {
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

	if !bytes.Contains([]byte(output), []byte("Buy & Hold")) {
		t.Errorf("output = %q, want it to contain the config file's strategy", output)
	}
	if bytes.Contains([]byte(output), []byte("Two-Candle Breakout")) {
		t.Errorf("output = %q, want it to NOT contain the explicitly-passed strategy, since -config ignores it", output)
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
