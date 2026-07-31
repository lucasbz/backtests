package strategies

import (
	"github.com/lucasbz/backtests/internal/domain"
)

// TwoCandleBreakout repeatedly buys and sells based on a sliding two-candle
// range: on every candle from the third one onward, it looks at the low/high
// of the two candles immediately before it.
//
//   - While flat, it buys at the minimum of those two candles' lows, but only
//     once the current candle's low actually reaches that price.
//   - Once holding, it sells at the maximum of those two candles' highs, but
//     only once the current candle's high actually reaches that price.
//
// If a candle doesn't reach the trigger price, nothing happens and the
// window slides forward to the next candle. A completed sell immediately
// starts a new buy search on the very next candle (no cooldown); Backtest
// reinvests the full sell proceeds into the next buy's position size.
//
// TwoCandleBreakout never force-closes a position: if the backtest's
// candles run out while it's still holding, that position is left open and
// Backtest drops it, exactly as it did under the old self-driven
// implementation (see the package tests for this documented behavior).
type TwoCandleBreakout struct {
	// window holds the (up to) two candles immediately preceding the one
	// currently being decided on - the sliding two-candle range described
	// above. It's the only state this strategy keeps; "am I holding" is
	// Backtest's job now, reflected in Decide's position argument.
	window []domain.Candle
}

func (s *TwoCandleBreakout) Name() string {
	return "Two-Candle Breakout"
}

// Decide implements domain.Strategy. It ignores isLast: per the type doc
// above, an unclosed position at the end of the backtest is intentionally
// dropped, not force-sold.
func (s *TwoCandleBreakout) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
	defer s.remember(candle)

	if len(s.window) < 2 {
		return nil // not enough history yet to have a two-candle range
	}
	prev1, prev2 := s.window[0], s.window[1]

	if position == nil {
		minPrice := prev1.Low
		if prev2.Low.Amount() < minPrice.Amount() {
			minPrice = prev2.Low
		}
		if candle.Low.Amount() > minPrice.Amount() {
			return nil // low of the last two candles wasn't reached yet
		}
		return &domain.Order{
			Date:      candle.Date,
			Price:     minPrice,
			OrderType: domain.Buy,
		}
	}

	maxPrice := prev1.High
	if prev2.High.Amount() > maxPrice.Amount() {
		maxPrice = prev2.High
	}
	if candle.High.Amount() < maxPrice.Amount() {
		return nil // high of the last two candles wasn't reached yet
	}
	return &domain.Order{
		Date:      candle.Date,
		Price:     maxPrice,
		Quantity:  position.Buy.Quantity,
		OrderType: domain.Sell,
	}
}

// remember slides the window forward by candle, keeping only the most
// recent two candles.
func (s *TwoCandleBreakout) remember(candle domain.Candle) {
	s.window = append(s.window, candle)
	if len(s.window) > 2 {
		s.window = s.window[len(s.window)-2:]
	}
}
