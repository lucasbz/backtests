package stoploss

import (
	"math"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// PercentStopLoss closes a position once a candle's low touches a fixed
// percentage below the position's entry price - e.g. Percent: 5 exits at
// 95% of the entry price.
type PercentStopLoss struct {
	// Percent is a plain percentage number, not a fraction: 5 means 5%, not
	// 0.05.
	Percent float64
}

func (s *PercentStopLoss) Name() string {
	return "Percent"
}

// triggerPrice is entry price x (1 - Percent/100), rounded to the nearest
// cent the same way util.MoneyFromFloat rounds a decimal input.
func (s *PercentStopLoss) triggerPrice(entry money.Money) money.Money {
	factor := 1 - s.Percent/100
	amount := int64(math.Round(float64(entry.Amount()) * factor))
	return *money.New(amount, domain.Currency)
}

// Check implements domain.StopLoss.
func (s *PercentStopLoss) Check(candle domain.Candle, position domain.Position) *domain.Order {
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
