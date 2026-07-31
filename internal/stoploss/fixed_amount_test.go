package stoploss

import (
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestFixedAmountStopLoss_Name(t *testing.T) {
	s := &FixedAmountStopLoss{Amount: stopLossTestMoney(200)}
	if s.Name() != "Fixed Amount" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Fixed Amount")
	}
}

func TestFixedAmountStopLoss_TriggersWhenLowTouchesThreshold(t *testing.T) {
	// entry 1000, amount 200 -> trigger price 800. Candle's low (750)
	// reaches it.
	s := &FixedAmountStopLoss{Amount: stopLossTestMoney(200)}
	position := stopLossTestPosition(1000, 10)
	candle := stopLossTestCandle("2010-01-05", 750, 900)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order")
	}
	assertOrder(t, "SellOrder", *order, "2010-01-05", 800, 10, domain.Sell)
}

func TestFixedAmountStopLoss_DoesNotTriggerAboveThreshold(t *testing.T) {
	// entry 1000, amount 200 -> trigger price 800. Candle's low (850)
	// never reaches it.
	s := &FixedAmountStopLoss{Amount: stopLossTestMoney(200)}
	position := stopLossTestPosition(1000, 10)
	candle := stopLossTestCandle("2010-01-05", 850, 900)

	if order := s.Check(candle, position); order != nil {
		t.Errorf("Check() = %+v, want nil (trigger never touched)", order)
	}
}

func TestFixedAmountStopLoss_TriggersExactlyAtThreshold(t *testing.T) {
	s := &FixedAmountStopLoss{Amount: stopLossTestMoney(150)}
	position := stopLossTestPosition(1000, 3)
	candle := stopLossTestCandle("2010-01-05", 850, 900)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order (low exactly at trigger)")
	}
	assertOrder(t, "SellOrder", *order, "2010-01-05", 850, 3, domain.Sell)
}
