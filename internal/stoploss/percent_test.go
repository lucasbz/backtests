package stoploss

import (
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

func stopLossTestMoney(amount int64) money.Money {
	return *money.New(amount, domain.Currency)
}

func stopLossTestCandle(date string, low, high int64) domain.Candle {
	return domain.Candle{Date: date, Low: stopLossTestMoney(low), High: stopLossTestMoney(high)}
}

func stopLossTestPosition(buyPrice, quantity int64) domain.Position {
	return domain.Position{Buy: domain.Order{
		Date:      "2010-01-04",
		Price:     stopLossTestMoney(buyPrice),
		Quantity:  quantity,
		OrderType: domain.Buy,
	}}
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

func TestPercentStopLoss_Name(t *testing.T) {
	s := &PercentStopLoss{Percent: 5}
	if s.Name() != "Percent" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Percent")
	}
}

func TestPercentStopLoss_TriggersWhenLowTouchesThreshold(t *testing.T) {
	// entry 1000, 5% -> trigger price 950. Candle's low (940) reaches it.
	s := &PercentStopLoss{Percent: 5}
	position := stopLossTestPosition(1000, 10)
	candle := stopLossTestCandle("2010-01-05", 940, 980)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order")
	}
	assertOrder(t, "SellOrder", *order, "2010-01-05", 950, 10, domain.Sell)
}

func TestPercentStopLoss_DoesNotTriggerAboveThreshold(t *testing.T) {
	// entry 1000, 5% -> trigger price 950. Candle's low (960) never
	// reaches it.
	s := &PercentStopLoss{Percent: 5}
	position := stopLossTestPosition(1000, 10)
	candle := stopLossTestCandle("2010-01-05", 960, 980)

	if order := s.Check(candle, position); order != nil {
		t.Errorf("Check() = %+v, want nil (trigger never touched)", order)
	}
}

func TestPercentStopLoss_TriggersExactlyAtThreshold(t *testing.T) {
	s := &PercentStopLoss{Percent: 10}
	position := stopLossTestPosition(1000, 5)
	candle := stopLossTestCandle("2010-01-05", 900, 950)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order (low exactly at trigger)")
	}
	assertOrder(t, "SellOrder", *order, "2010-01-05", 900, 5, domain.Sell)
}
