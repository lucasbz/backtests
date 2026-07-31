package strategies

import (
	"github.com/lucasbz/backtests/internal/domain"
)

// BuyAndHold buys once, at the low of the first candle it's asked about,
// and sells once, at the high of the last candle of the backtest - signaled
// to Decide via isLast, since a pure per-candle decision has no other way
// to know it has reached the end. Position size is as many whole shares as
// the running balance affords at the buy price, computed by Backtest (see
// domain.Strategy.Decide), not here.
//
// Because Decide only ever proposes one action per candle - a buy while
// flat, or an exit while holding, never both on the same candle - a
// backtest consisting of a single candle buys but never sells: there's no
// second candle on which to ask the "should I exit, and is this the last
// one" question. This is an intentional consequence of moving from the old
// self-driven implementation (which computed first/last from the whole
// traversed slice up front, even if they were the same candle) to Backtest
// driving a real streaming, candle-by-candle loop; see
// TestBuyAndHold_Decide_SingleCandleNeverCloses.
type BuyAndHold struct{}

func (s *BuyAndHold) Name() string {
	return "Buy & Hold"
}

func (s *BuyAndHold) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
	if position == nil {
		return &domain.Order{
			Date:      candle.Date,
			Price:     candle.Low,
			OrderType: domain.Buy,
		}
	}

	if !isLast {
		return nil
	}
	return &domain.Order{
		Date:      candle.Date,
		Price:     candle.High,
		Quantity:  position.Buy.Quantity,
		OrderType: domain.Sell,
	}
}
