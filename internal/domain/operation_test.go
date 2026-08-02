package domain

import (
	"encoding/json"
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

func TestOperation_Outcome(t *testing.T) {
	tests := []struct {
		name      string
		buyPrice  int64
		sellPrice int64
		want      OperationOutcome
	}{
		{"gain", 1000, 1500, Gain},
		{"loss", 1500, 1000, Loss},
		{"break-even counts as gain", 1000, 1000, Gain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := Operation{
				BuyOrder:  Order{Price: *money.New(tt.buyPrice, Currency), Quantity: 1, OrderType: Buy},
				SellOrder: Order{Price: *money.New(tt.sellPrice, Currency), Quantity: 1, OrderType: Sell},
			}

			got, err := op.Outcome()
			if err != nil {
				t.Fatalf("Outcome: %v", err)
			}
			if got != tt.want {
				t.Errorf("Outcome() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOperation_Days(t *testing.T) {
	tests := []struct {
		name string
		buy  string
		sell string
		want int
	}{
		{"same day", "2020-01-01", "2020-01-01", 0},
		{"multi day", "2020-01-01", "2020-01-15", 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := Operation{
				BuyOrder:  Order{Date: tt.buy, Price: *money.New(1000, Currency), Quantity: 1, OrderType: Buy},
				SellOrder: Order{Date: tt.sell, Price: *money.New(1000, Currency), Quantity: 1, OrderType: Sell},
			}

			got, err := op.Days()
			if err != nil {
				t.Fatalf("Days: %v", err)
			}
			if got != tt.want {
				t.Errorf("Days() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOperation_Days_InvalidDate(t *testing.T) {
	op := Operation{
		BuyOrder:  Order{Date: "not-a-date", Price: *money.New(1000, Currency), Quantity: 1, OrderType: Buy},
		SellOrder: Order{Date: "2020-01-15", Price: *money.New(1000, Currency), Quantity: 1, OrderType: Sell},
	}

	if _, err := op.Days(); err == nil {
		t.Fatal("Days() error = nil, want an error for an unparseable buy date")
	}
}

func TestOperation_MarshalJSON_IncludesDays(t *testing.T) {
	op := Operation{
		Date:      "2020-01-01",
		BuyOrder:  Order{Date: "2020-01-01", Price: *money.New(1000, Currency), Quantity: 10, OrderType: Buy},
		SellOrder: Order{Date: "2020-01-15", Price: *money.New(1500, Currency), Quantity: 10, OrderType: Sell},
	}

	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	days, ok := got["days"].(float64)
	if !ok {
		t.Fatalf("days field missing or not a number, got %v", got["days"])
	}
	if int(days) != 14 {
		t.Errorf("days = %v, want 14", days)
	}
}
