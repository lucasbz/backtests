package main

import "testing"

func TestRun_NoCommand(t *testing.T) {
	if err := run([]string{}); err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	if err := run([]string{"nope"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
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
