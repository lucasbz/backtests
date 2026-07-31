package strategies

import (
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

func newMoney(amount int64) money.Money {
	return *money.New(amount, domain.Currency)
}

func candleWithLowHigh(date string, low, high int64) domain.Candle {
	return domain.Candle{Date: date, Low: newMoney(low), High: newMoney(high)}
}

func assertOrder(t *testing.T, label string, got domain.Order, wantDate string, wantPrice int64, wantQuantity int64, wantType domain.OrderType) {
	t.Helper()
	if got.Date != wantDate {
		t.Errorf("%s.Date = %q, want %q", label, got.Date, wantDate)
	}
	if got.Price.Amount() != wantPrice {
		t.Errorf("%s.Price = %d, want %d", label, got.Price.Amount(), wantPrice)
	}
	if got.Quantity != wantQuantity {
		t.Errorf("%s.Quantity = %d, want %d", label, got.Quantity, wantQuantity)
	}
	if got.OrderType != wantType {
		t.Errorf("%s.OrderType = %q, want %q", label, got.OrderType, wantType)
	}
}

func TestBuyAndHold_Operations_NoCandles(t *testing.T) {
	s := &BuyAndHold{}
	if ops := s.Operations(); ops != nil {
		t.Errorf("Operations() = %+v, want nil", ops)
	}
}

func TestBuyAndHold_Operations_BuysLowSellsHigh(t *testing.T) {
	s := &BuyAndHold{Balance: newMoney(1200)}
	s.Traverse(candleWithLowHigh("2010-01-04", 1200, 1242))
	s.Traverse(candleWithLowHigh("2010-06-15", 1500, 1550))
	s.Traverse(candleWithLowHigh("2010-12-30", 1800, 1899))

	ops := s.Operations()
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}

	op := ops[0]
	assertOrder(t, "BuyOrder", op.BuyOrder, "2010-01-04", 1200, 1, domain.Buy)
	assertOrder(t, "SellOrder", op.SellOrder, "2010-12-30", 1899, 1, domain.Sell)
	if op.Date != op.BuyOrder.Date {
		t.Errorf("Operation.Date = %q, want %q", op.Date, op.BuyOrder.Date)
	}
}

func TestBuyAndHold_Operations_SingleCandleBuysAndSellsSameDay(t *testing.T) {
	s := &BuyAndHold{Balance: newMoney(1200)}
	s.Traverse(candleWithLowHigh("2010-01-04", 1200, 1242))

	ops := s.Operations()
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}
	if ops[0].BuyOrder.Price.Amount() != 1200 || ops[0].SellOrder.Price.Amount() != 1242 {
		t.Errorf("op = %+v, want buy@1200 sell@1242", ops[0])
	}
}

func TestBuyAndHold_Operations_QuantityScalesWithBalance(t *testing.T) {
	s := &BuyAndHold{Balance: newMoney(3600)}
	s.Traverse(candleWithLowHigh("2010-01-04", 1200, 1242))

	ops := s.Operations()
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}
	if ops[0].BuyOrder.Quantity != 3 || ops[0].SellOrder.Quantity != 3 {
		t.Errorf("quantity = buy:%d sell:%d, want 3 for both", ops[0].BuyOrder.Quantity, ops[0].SellOrder.Quantity)
	}
}

func TestBuyAndHold_Operations_InsufficientBalance(t *testing.T) {
	s := &BuyAndHold{Balance: newMoney(500)}
	s.Traverse(candleWithLowHigh("2010-01-04", 1200, 1242))

	if ops := s.Operations(); ops != nil {
		t.Errorf("Operations() = %+v, want nil (balance can't afford one share)", ops)
	}
}
