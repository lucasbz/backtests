package strategies

import (
	"fmt"

	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/indicators"
)

// crossoverStrategy is the shared implementation behind "sma-crossover"
// and "ema-crossover": two indicators of the same kind (both SMA, or both
// EMA), a "short" period and a "long" period. It buys, while flat, when
// the short indicator crosses above the long one, and sells, while
// holding, when it crosses back below.
type crossoverStrategy struct {
	short, long indicators.Indicator
	// wasShortAbove is nil until both indicators have been ready on at
	// least one prior candle - there's no "prior state" to compare against
	// on the very first ready candle, so it can't possibly be a cross yet.
	wasShortAbove *bool
	name          string
}

// newCrossoverStrategy builds a crossoverStrategy backed by two
// indicatorKind (e.g. "sma" or "ema") instances, reading their periods
// from params' "shortPeriod"/"longPeriod" keys. name is the strategy's
// display Name().
func newCrossoverStrategy(name, indicatorKind string, params map[string]float64) (domain.Strategy, error) {
	shortPeriod, ok := params["shortPeriod"]
	if !ok {
		return nil, fmt.Errorf("%s: missing required %q param", name, "shortPeriod")
	}
	longPeriod, ok := params["longPeriod"]
	if !ok {
		return nil, fmt.Errorf("%s: missing required %q param", name, "longPeriod")
	}
	if shortPeriod >= longPeriod {
		return nil, fmt.Errorf("%s: shortPeriod (%v) must be less than longPeriod (%v)", name, shortPeriod, longPeriod)
	}

	short, err := indicators.LoadIndicator(indicatorKind, map[string]float64{"period": shortPeriod})
	if err != nil {
		return nil, fmt.Errorf("%s: shortPeriod: %w", name, err)
	}
	long, err := indicators.LoadIndicator(indicatorKind, map[string]float64{"period": longPeriod})
	if err != nil {
		return nil, fmt.Errorf("%s: longPeriod: %w", name, err)
	}

	return &crossoverStrategy{short: short, long: long, name: name}, nil
}

func (s *crossoverStrategy) Name() string {
	return s.name
}

// Decide implements domain.Strategy. It ignores isLast, same as
// TwoCandleBreakout: an unclosed position at the end of the backtest is
// left open and dropped, not force-sold.
//
// Both indicators are updated with candle FIRST, before their Value() is
// read - unlike TwoCandleBreakout, which compares a window of PRIOR
// candles against the CURRENT candle's raw Low/High (passed directly as
// this method's argument, never round-tripped through the window) to
// decide whether today touched a threshold set before today. That
// staleness is harmless there because the actual trigger check still uses
// today's real data directly. Here, by contrast, the indicator value IS
// the entire signal - there's no separate "check against today's raw
// data" step - so reading a value that excludes today's candle would
// detect a cross using yesterday's trend, a full candle after it actually
// happened, while still filling the order at today's (by then unrelated)
// closing price. Updating first keeps the crossing signal and the
// candle.Close fill price describing the same candle - "the candle that
// confirmed the cross" is also the candle the trade fills on.
func (s *crossoverStrategy) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
	s.short.Update(candle)
	s.long.Update(candle)

	shortVal, shortReady := s.short.Value()
	longVal, longReady := s.long.Value()
	if !shortReady || !longReady {
		return nil // still warming up
	}

	nowAbove := shortVal > longVal
	defer func() { s.wasShortAbove = &nowAbove }()
	if s.wasShortAbove == nil {
		return nil // first ready candle - no prior state to compare against, no cross yet
	}

	crossedUp := !*s.wasShortAbove && nowAbove
	crossedDown := *s.wasShortAbove && !nowAbove

	if crossedUp && position == nil {
		return &domain.Order{
			Date:      candle.Date,
			Price:     candle.Close,
			OrderType: domain.Buy,
		}
	}
	if crossedDown && position != nil {
		return &domain.Order{
			Date:      candle.Date,
			Price:     candle.Close,
			Quantity:  position.Buy.Quantity,
			OrderType: domain.Sell,
		}
	}
	return nil
}
