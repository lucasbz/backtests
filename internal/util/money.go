package util

import (
	"fmt"
	"math"
	"strconv"

	"github.com/Rhymond/go-money"
)

// MoneyFromFloat converts a decimal value (e.g. 19.9) to money.Money in
// currency (a go-money currency code, e.g. money.BRL).
// money.NewFromFloat truncates (int64(amount*100)) instead of rounding, so
// float64 imprecision silently corrupts values like 19.9 -> 1989 instead of
// 1990; rounding first and going through money.New avoids that.
func MoneyFromFloat(f float64, currency string) money.Money {
	return *money.New(int64(math.Round(f*100)), currency)
}

// ParseMoney parses a plain decimal string (e.g. "10000.00", from a CLI
// flag or JSON request field) into a money.Money in currency.
//
// strconv.ParseFloat happily accepts "Inf"/"+Inf"/"-Inf"/"NaN" (and their
// case-insensitive variants) as valid float64 values with no error; feeding
// one of those into MoneyFromFloat's math.Round(f*100) then int64(...)
// conversion silently saturates to math.MaxInt64 (or NaN converts to 0)
// instead of erroring, which would let a caller like ParsePositiveMoney's
// positivity check pass trivially on a bogus, astronomically large value.
// Reject non-finite input explicitly instead.
func ParseMoney(s, currency string) (money.Money, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return money.Money{}, fmt.Errorf("parsing money %q: %w", s, err)
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return money.Money{}, fmt.Errorf("parsing money %q: value must be a finite number", s)
	}
	return MoneyFromFloat(f, currency), nil
}

// ParsePositiveMoney parses s the same way ParseMoney does, plus
// validates it's positive - the one rule every backtest entry point (CLI
// flags, the JSON API) applies to its balance input, previously duplicated
// at each call site with its own copy of the positivity check.
func ParsePositiveMoney(s, currency string, errMsg string) (money.Money, error) {
	balance, err := ParseMoney(s, currency)
	if err != nil {
		return money.Money{}, fmt.Errorf("parsing money: %w", err)
	}
	if !balance.IsPositive() {
		return money.Money{}, fmt.Errorf(errMsg, s)
	}
	return balance, nil
}
