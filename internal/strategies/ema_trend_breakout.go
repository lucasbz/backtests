package strategies

import (
	"fmt"

	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/indicators"
)

// EMATrendBreakout is TwoCandleBreakout's sliding two-candle range trigger
// (see that type's doc comment for the full rule), with one extra condition
// ANDed onto the buy side only: a short EMA must be above a long EMA
// (uptrend) and price must be above the short EMA. The sell/exit side is
// completely unchanged from TwoCandleBreakout - it never looks at the EMAs.
//
//   - While flat, it buys at the minimum of the last two candles' lows, but
//     only once the current candle's low reaches that price AND the trend
//     filter (short EMA > long EMA, and candle.Close above the short EMA)
//     holds.
//   - Once holding, it sells at the maximum of the last two candles' highs,
//     exactly like TwoCandleBreakout, regardless of EMA state.
//
// Like TwoCandleBreakout, it never force-closes a position: an unclosed
// position when candles run out is left open and dropped by Backtest, not
// force-sold.
type EMATrendBreakout struct {
	short, long indicators.Indicator

	// window holds the (up to) two candles immediately preceding the one
	// currently being decided on, same role as TwoCandleBreakout.window.
	window []domain.Candle
}

// newEMATrendBreakout builds an EMATrendBreakout, reading its EMA periods
// from params' "shortPeriod"/"longPeriod" keys - same keys and validation
// (shortPeriod < longPeriod) as newCrossoverStrategy, for consistency
// across this package's indicator-backed strategies.
func newEMATrendBreakout(params map[string]float64) (domain.Strategy, error) {
	const name = "EMA Trend Breakout"

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

	short, err := indicators.LoadIndicator("ema", map[string]float64{"period": shortPeriod})
	if err != nil {
		return nil, fmt.Errorf("%s: shortPeriod: %w", name, err)
	}
	long, err := indicators.LoadIndicator("ema", map[string]float64{"period": longPeriod})
	if err != nil {
		return nil, fmt.Errorf("%s: longPeriod: %w", name, err)
	}

	return &EMATrendBreakout{short: short, long: long}, nil
}

func (s *EMATrendBreakout) Name() string {
	return "EMA Trend Breakout"
}

// Decide implements domain.Strategy. It ignores isLast, same as
// TwoCandleBreakout: an unclosed position at the end of the backtest is
// left open and dropped, not force-sold.
//
// Both EMAs are updated with candle FIRST, before their Value() is read -
// same ordering, and same rationale, as crossoverStrategy.Decide: the buy
// fill price (minPrice) is derived from the window of PRIOR candles, same
// as TwoCandleBreakout, but the trend filter's signal must still reflect
// today's candle, same as the window's Low check does against
// candle.Low/candle.Close. Updating the EMAs after reading Value() would
// make the trend filter lag a full candle behind the price it's being
// compared against.
func (s *EMATrendBreakout) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
	defer s.remember(candle)

	s.short.Update(candle)
	s.long.Update(candle)

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

		shortVal, shortReady := s.short.Value()
		longVal, longReady := s.long.Value()
		if !shortReady || !longReady {
			return nil // still warming up
		}
		if shortVal <= longVal {
			return nil // not an uptrend
		}
		// candle.Close > shortVal is checked explicitly for readability,
		// even though shortVal > longVal above already mathematically
		// implies close above shortVal is also close above longVal (i.e.
		// above both EMAs) - it's not derived silently from the first
		// check.
		if candle.Close.AsMajorUnits() <= shortVal {
			return nil // price hasn't confirmed the uptrend yet
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
// recent two candles - identical to TwoCandleBreakout.remember.
func (s *EMATrendBreakout) remember(candle domain.Candle) {
	s.window = append(s.window, candle)
	if len(s.window) > 2 {
		s.window = s.window[len(s.window)-2:]
	}
}
