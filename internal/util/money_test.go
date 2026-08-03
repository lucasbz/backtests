package util

import (
	"testing"

	"github.com/Rhymond/go-money"
)

func assertMoneyAmount(t *testing.T, field string, got money.Money, want int64) {
	t.Helper()
	if got.Amount() != want {
		t.Errorf("%s.Amount() = %d, want %d", field, got.Amount(), want)
	}
	if got.Currency().Code != money.BRL {
		t.Errorf("%s.Currency = %s, want %s", field, got.Currency().Code, money.BRL)
	}
}

func TestParseMoney(t *testing.T) {
	got, err := ParseMoney("10000.00", money.BRL)
	if err != nil {
		t.Fatalf("ParseMoney: %v", err)
	}
	assertMoneyAmount(t, "ParseMoney(10000.00)", got, 1000000)
}

func TestParseMoney_Invalid(t *testing.T) {
	if _, err := ParseMoney("not-a-number", money.BRL); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

// TestParseMoney_NonFinite checks that Inf/-Inf/NaN - all of which
// strconv.ParseFloat parses without error - are explicitly rejected by
// ParseMoney instead of silently saturating to math.MaxInt64 (Inf) or 0
// (NaN) cents (see docs/plans/code-review-findings.md, finding #2).
func TestParseMoney_NonFinite(t *testing.T) {
	for _, s := range []string{"Inf", "+Inf", "-Inf", "inf", "NaN", "nan"} {
		if _, err := ParseMoney(s, money.BRL); err == nil {
			t.Errorf("ParseMoney(%q) = nil error, want an error (non-finite input)", s)
		}
	}
}

func TestParsePositiveMoney(t *testing.T) {
	got, err := ParsePositiveMoney("10000.00", money.BRL, "balance must be greater than zero")
	if err != nil {
		t.Fatalf("ParsePositiveMoney: %v", err)
	}
	assertMoneyAmount(t, "ParsePositiveMoney(10000.00)", got, 1000000)
}

func TestParsePositiveMoney_Invalid(t *testing.T) {
	if _, err := ParsePositiveMoney("not-a-number", money.BRL, "balance must be greater than zero"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

// TestParsePositiveMoney_NotPositive checks the rule ParseMoney itself
// doesn't enforce (it's specific to a *starting balance*, not every
// money value): zero and negative amounts are both rejected.
func TestParsePositiveMoney_NotPositive(t *testing.T) {
	for _, s := range []string{"0", "0.00", "-1", "-10000.00"} {
		if _, err := ParsePositiveMoney(s, money.BRL, "balance must be greater than zero"); err == nil {
			t.Errorf("ParsePositiveMoney(%q) = nil error, want an error (not positive)", s)
		}
	}
}
