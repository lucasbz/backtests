package stoploss

import (
	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// FixedAmountStopLoss closes a position once a candle's low touches a fixed
// currency amount below the position's entry price - e.g. Amount of R$2.00
// exits once the price has dropped 2 reais from the entry.
type FixedAmountStopLoss struct {
	Amount money.Money
}

func (s *FixedAmountStopLoss) Name() string {
	return "Fixed Amount"
}

// triggerPrice is entry price - Amount.
func (s *FixedAmountStopLoss) triggerPrice(entry money.Money) money.Money {
	return *money.New(entry.Amount()-s.Amount.Amount(), domain.Currency)
}

// Check implements domain.StopLoss.
func (s *FixedAmountStopLoss) Check(candle domain.Candle, position domain.Position) *domain.Order {
	trigger := s.triggerPrice(position.Buy.Price)
	if candle.Low.Amount() > trigger.Amount() {
		return nil // candle never dropped low enough to touch the trigger
	}
	return &domain.Order{
		Date:      candle.Date,
		Price:     trigger,
		Quantity:  position.Buy.Quantity,
		OrderType: domain.Sell,
	}
}
