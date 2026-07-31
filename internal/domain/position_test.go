package domain

import (
	"testing"

	"github.com/Rhymond/go-money"
)

func positionTestMoney(amount int64) money.Money {
	return *money.New(amount, Currency)
}

func TestPosition_Close(t *testing.T) {
	buy := Order{Date: "2010-01-04", Price: positionTestMoney(1200), Quantity: 10, OrderType: Buy}
	sell := Order{Date: "2010-12-30", Price: positionTestMoney(1500), Quantity: 10, OrderType: Sell}

	p := Position{Buy: buy}
	op := p.Close(sell)

	if op.Date != buy.Date {
		t.Errorf("Date = %q, want %q (buy's date)", op.Date, buy.Date)
	}
	if op.BuyOrder != buy {
		t.Errorf("BuyOrder = %+v, want %+v", op.BuyOrder, buy)
	}
	if op.SellOrder != sell {
		t.Errorf("SellOrder = %+v, want %+v", op.SellOrder, sell)
	}
}
