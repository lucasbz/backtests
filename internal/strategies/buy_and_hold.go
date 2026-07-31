package strategies

import (
	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// BuyAndHold buys once, at the low of the first candle in the time series,
// and sells once, at the high of the last candle in the time series. Position
// size is as many whole shares as Balance affords at the buy price.
type BuyAndHold struct {
	Balance money.Money

	candles []domain.Candle
}

func (s *BuyAndHold) Traverse(candle domain.Candle) {
	s.candles = append(s.candles, candle)
}

func (s *BuyAndHold) Name() string {
	return "Buy & Hold"
}

// Operations returns the single buy/sell pair for the traversed time series.
// It's empty if no candles were traversed, or if Balance can't afford even
// one share at the buy price.
func (s *BuyAndHold) Operations() []domain.Operation {
	if len(s.candles) == 0 {
		return nil
	}

	first := s.candles[0]
	last := s.candles[len(s.candles)-1]

	quantity := s.Balance.Amount() / first.Low.Amount()
	if quantity <= 0 {
		return nil
	}

	buy := domain.Order{
		Date:      first.Date,
		Price:     first.Low,
		Quantity:  quantity,
		OrderType: domain.Buy,
	}
	sell := domain.Order{
		Date:      last.Date,
		Price:     last.High,
		Quantity:  quantity,
		OrderType: domain.Sell,
	}

	return []domain.Operation{
		{Date: buy.Date, BuyOrder: buy, SellOrder: sell},
	}
}
