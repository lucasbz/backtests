package stoploss

import (
	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// FixedAmountStopLoss closes a position once a candle's low touches a fixed
// currency amount below the position's entry price - e.g. Amount of R$2.00
// exits once the per-share price has dropped 2 reais from the entry.
//
// Amount is a PER-SHARE price drop, not a cap on the total position's
// currency loss: the realized loss on the whole position when this fires is
// approximately Amount * position.Buy.Quantity (it exits at exactly
// triggerPrice, so it's usually a little under that in magnitude, absent a
// gap-through where the candle's low is below the trigger and Check still
// fills at triggerPrice rather than the worse low - see Check). E.g. a
// 1000-share position with Amount=R$2.00 can realize a loss of roughly
// R$2000 when the stop-loss fires, not R$2.00. Callers wanting a cap on the
// total position's loss need to size Amount themselves as
// desired-total-loss / quantity, or use a different stop-loss type; there's
// currently no "total position loss cap" stop-loss type in this registry.
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
