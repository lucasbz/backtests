// Package indicators provides reusable technical indicators (SMA, EMA,
// RSI) that internal/strategies implementations compose to make entry/exit
// decisions, instead of each strategy hand-rolling its own rolling
// calculation - see docs/plans/indicators.md for the design this package
// implements.
//
// Indicators compute in float64, not money.Money: unlike orders and
// balances, an indicator's value is a decision signal, not a settled
// currency amount, and money.Money's integer-cents arithmetic is awkward
// for rolling averages. Prices are converted to float64 at the boundary,
// via candle.Close.AsMajorUnits(), when Update is called.
package indicators

import "github.com/lucasbz/backtests/internal/domain"

// Indicator is fed one candle at a time, in chronological order, and
// reports its current computed value plus whether it has seen enough
// candles yet to produce a meaningful one.
type Indicator interface {
	// Update feeds the next candle into the indicator's rolling state.
	Update(candle domain.Candle)
	// Value returns the indicator's current value and whether it's ready
	// (false during warm-up, e.g. the first Period-1 candles of an SMA).
	// The value returned while !ready is unspecified and must not be used.
	Value() (value float64, ready bool)
}
