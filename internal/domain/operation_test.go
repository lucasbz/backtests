package domain

import (
	"testing"

	"github.com/Rhymond/go-money"
)

func TestOperation_Profit_Gain(t *testing.T) {
	op := Operation{
		Date:      "2010-01-04",
		BuyOrder:  Order{Price: *money.New(1200, Currency), Quantity: 10, OrderType: Buy},
		SellOrder: Order{Price: *money.New(1500, Currency), Quantity: 10, OrderType: Sell},
	}

	profit, err := op.Profit()
	if err != nil {
		t.Fatalf("Profit: %v", err)
	}
	// bought 10 @ 1200 = 12000, sold 10 @ 1500 = 15000, profit = 3000
	if profit.Amount() != 3000 {
		t.Errorf("Profit = %d, want 3000", profit.Amount())
	}
}

func TestOperation_Profit_Loss(t *testing.T) {
	op := Operation{
		BuyOrder:  Order{Price: *money.New(2000, Currency), Quantity: 5, OrderType: Buy},
		SellOrder: Order{Price: *money.New(1000, Currency), Quantity: 5, OrderType: Sell},
	}

	profit, err := op.Profit()
	if err != nil {
		t.Fatalf("Profit: %v", err)
	}
	// bought 5 @ 2000 = 10000, sold 5 @ 1000 = 5000, profit = -5000
	if profit.Amount() != -5000 {
		t.Errorf("Profit = %d, want -5000", profit.Amount())
	}
}

func TestOperation_Profit_BreakEven(t *testing.T) {
	op := Operation{
		BuyOrder:  Order{Price: *money.New(1000, Currency), Quantity: 3, OrderType: Buy},
		SellOrder: Order{Price: *money.New(1000, Currency), Quantity: 3, OrderType: Sell},
	}

	profit, err := op.Profit()
	if err != nil {
		t.Fatalf("Profit: %v", err)
	}
	if !profit.IsZero() {
		t.Errorf("Profit = %d, want 0", profit.Amount())
	}
}
