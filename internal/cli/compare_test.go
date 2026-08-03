package cli

import (
	"bytes"
	"testing"
)

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

func TestRunCLI_Compare_InvalidBalance(t *testing.T) {
	err := RunCLI([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "not-a-number",
	})
	if err == nil {
		t.Fatal("expected error for invalid balance")
	}
}

func TestRunCLI_Compare_ZeroBalance(t *testing.T) {
	err := RunCLI([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "0",
	})
	if err == nil {
		t.Fatal("expected error for zero balance")
	}
}

func TestRunCLI_Compare_InvalidDate(t *testing.T) {
	err := RunCLI([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "not-a-date",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid start date")
	}
}

func TestRunCLI_Compare_InvalidEndDate(t *testing.T) {
	err := RunCLI([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-01-01",
		"-end", "not-a-date",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid end date")
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

// TestRunCLI_Compare_EndBeforeStart mirrors
// TestRunCLI_Backtest_EndBeforeStart.
func TestRunCLI_Compare_EndBeforeStart(t *testing.T) {
	err := RunCLI([]string{
		"compare",
		"-asset", "PETR4",
		"-start", "2010-12-31",
		"-end", "2010-01-01",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for end before start")
	}
}

func TestRunCLI_Compare_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"compare", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
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
