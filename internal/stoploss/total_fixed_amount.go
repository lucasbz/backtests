package stoploss

import (
	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// TotalFixedAmountStopLoss closes a position once its total realized loss
// would reach a fixed currency amount, regardless of position size - e.g.
// Amount of R$500 exits a position (however many shares it holds) once its
// total loss would reach R$500.
//
// Unlike FixedAmountStopLoss (whose Amount is a PER-SHARE price drop, with
// the realized total loss scaling with quantity - see its doc comment),
// TotalFixedAmountStopLoss's Amount is the cap on the position's TOTAL
// realized loss. To achieve that, the per-share trigger is derived by
// dividing Amount across the position's quantity: perShareDropCents :=
// Amount.Amount() / position.Buy.Quantity. Both are int64, so this is
// integer (floor) division - the per-share drop is rounded down to the
// nearest cent, meaning the realized total loss when this fires
// (perShareDropCents * quantity) is always <= Amount, short by at most
// quantity-1 cents. It will never let the total loss exceed Amount by more
// than a fraction of a cent per share.
type TotalFixedAmountStopLoss struct {
	Amount money.Money
}

func (s *TotalFixedAmountStopLoss) Name() string {
	return "Total Fixed Amount"
}

// triggerPrice is entry price - (Amount / quantity), floored to the nearest
// cent per share. quantity is assumed > 0: Check is only ever called on an
// already-open position, and internal/backtest/backtest.go's traverse only
// ever opens a position after confirming order.Quantity > 0, so there's no
// division-by-zero to guard against here.
func (s *TotalFixedAmountStopLoss) triggerPrice(entry money.Money, quantity int64) money.Money {
	perShareDropCents := s.Amount.Amount() / quantity
	return *money.New(entry.Amount()-perShareDropCents, domain.Currency)
}

// Check implements domain.StopLoss.
func (s *TotalFixedAmountStopLoss) Check(candle domain.Candle, position domain.Position) *domain.Order {
	trigger := s.triggerPrice(position.Buy.Price, position.Buy.Quantity)
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
