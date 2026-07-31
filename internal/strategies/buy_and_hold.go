package strategies

import (
	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// BuyAndHold buys once, at the low of the first quote in the time series,
// and sells once, at the high of the last quote in the time series. Position
// size is as many whole shares as Balance affords at the buy price.
type BuyAndHold struct {
	Balance money.Money

	quotes []domain.Quote
}

func (s *BuyAndHold) Traverse(quote domain.Quote) {
	s.quotes = append(s.quotes, quote)
}

func (s *BuyAndHold) Name() string {
	return "Buy & Hold"
}

// Operations returns the single buy/sell pair for the traversed time series.
// It's empty if no quotes were traversed, or if Balance can't afford even one
// share at the buy price.
func (s *BuyAndHold) Operations() []domain.Operation {
	if len(s.quotes) == 0 {
		return nil
	}

	first := s.quotes[0]
	last := s.quotes[len(s.quotes)-1]

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
