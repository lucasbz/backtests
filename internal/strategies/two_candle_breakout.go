package strategies

import (
	"github.com/Rhymond/go-money"
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
// starts a new buy search on the very next candle (no cooldown), reinvesting
// the full proceeds into the next buy's position size.
type TwoCandleBreakout struct {
	Balance money.Money

	candles []domain.Candle
}

func (s *TwoCandleBreakout) Traverse(candle domain.Candle) {
	s.candles = append(s.candles, candle)
}

func (s *TwoCandleBreakout) Name() string {
	return "Two-Candle Breakout"
}

// Operations replays the traversed candles through the buy/sell rules
// described on TwoCandleBreakout, reinvesting each sell's proceeds as the
// next buy's available balance. If a buy is still open when the traversed
// candles run out (no candle ever touched the sell trigger), it's dropped:
// only completed buy/sell pairs are reported.

func (s *TwoCandleBreakout) Operations() []domain.Operation {
	if len(s.candles) < 3 {
		return nil
	}

	var operations []domain.Operation
	balance := s.Balance
	holding := false
	var buyOrder domain.Order

	for i := 2; i < len(s.candles); i++ {
		prev1, prev2, current := s.candles[i-2], s.candles[i-1], s.candles[i]

		if !holding {
			minPrice := prev1.Low
			if prev2.Low.Amount() < minPrice.Amount() {
				minPrice = prev2.Low
			}
			if current.Low.Amount() > minPrice.Amount() {
				continue // low of the last two candles wasn't reached yet
			}

			quantity := balance.Amount() / minPrice.Amount()
			if quantity <= 0 {
				continue // can't afford even one share at this price yet
			}

			buyOrder = domain.Order{
				Date:      current.Date,
				Price:     minPrice,
				Quantity:  quantity,
				OrderType: domain.Buy,
			}
			holding = true
			continue
		}

		maxPrice := prev1.High
		if prev2.High.Amount() > maxPrice.Amount() {
			maxPrice = prev2.High
		}
		if current.High.Amount() < maxPrice.Amount() {
			continue // high of the last two candles wasn't reached yet
		}

		sellOrder := domain.Order{
			Date:      current.Date,
			Price:     maxPrice,
			Quantity:  buyOrder.Quantity,
			OrderType: domain.Sell,
		}
		operations = append(operations, domain.Operation{
			Date:      buyOrder.Date,
			BuyOrder:  buyOrder,
			SellOrder: sellOrder,
		})

		balance = sellOrder.Total()
		holding = false
	}

	return operations
}
