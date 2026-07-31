package main

import "testing"

func TestRun_Valid(t *testing.T) {
	err := run([]string{
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

func TestRun_UnknownStrategy(t *testing.T) {
	err := run([]string{
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

func TestRun_MissingArgs(t *testing.T) {
	err := run([]string{"-ticker", "PETR4"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRun_MissingBalance(t *testing.T) {
	err := run([]string{
		"-ticker", "PETR4",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
	})
	if err == nil {
		t.Fatal("expected error for missing balance")
	}
}

func TestRun_InvalidBalance(t *testing.T) {
	err := run([]string{
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

func TestRun_ZeroBalance(t *testing.T) {
	err := run([]string{
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

func TestRun_InvalidDate(t *testing.T) {
	err := run([]string{
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
