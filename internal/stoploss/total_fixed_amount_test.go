package stoploss

import (
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestTotalFixedAmountStopLoss_Name(t *testing.T) {
	s := &TotalFixedAmountStopLoss{Amount: stopLossTestMoney(50000)}
	if s.Name() != "Total Fixed Amount" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Total Fixed Amount")
	}
}

func TestTotalFixedAmountStopLoss_TriggerPrice(t *testing.T) {
	// Amount is a TOTAL cap, so the per-share drop is Amount/quantity,
	// floored to the nearest cent.
	tests := []struct {
		name          string
		amountCents   int64
		entryCents    int64
		quantity      int64
		wantTriggerCt int64
	}{
		{"evenly divisible", 20000, 1000, 10, 1000 - 2000}, // 20000/10 = 2000 exactly
		{"floors remainder", 50000, 1000, 517, 1000 - 96},  // 50000/517 = 96.7... -> 96
		{"single share is unaffected by division", 20000, 1000, 1, 1000 - 20000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TotalFixedAmountStopLoss{Amount: stopLossTestMoney(tt.amountCents)}
			got := s.triggerPrice(stopLossTestMoney(tt.entryCents), tt.quantity)
			if got.Amount() != tt.wantTriggerCt {
				t.Errorf("triggerPrice() = %d, want %d", got.Amount(), tt.wantTriggerCt)
			}
		})
	}
}

func TestTotalFixedAmountStopLoss_TriggersWhenLowTouchesThreshold(t *testing.T) {
	// entry 1000, amount 2000 total, quantity 10 -> per-share drop 200,
	// trigger price 800. Candle's low (750) reaches it.
	s := &TotalFixedAmountStopLoss{Amount: stopLossTestMoney(2000)}
	position := stopLossTestPosition(1000, 10)
	candle := stopLossTestCandle("2010-01-05", 750, 900)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order")
	}
	assertOrder(t, "SellOrder", *order, "2010-01-05", 800, 10, domain.Sell)
}

func TestTotalFixedAmountStopLoss_DoesNotTriggerAboveThreshold(t *testing.T) {
	// entry 1000, amount 2000 total, quantity 10 -> per-share drop 200,
	// trigger price 800. Candle's low (850) never reaches it.
	s := &TotalFixedAmountStopLoss{Amount: stopLossTestMoney(2000)}
	position := stopLossTestPosition(1000, 10)
	candle := stopLossTestCandle("2010-01-05", 850, 900)

	if order := s.Check(candle, position); order != nil {
		t.Errorf("Check() = %+v, want nil (trigger never touched)", order)
	}
}

func TestTotalFixedAmountStopLoss_TriggersExactlyAtThreshold(t *testing.T) {
	// entry 1000, amount 1500 total, quantity 3 -> per-share drop 500,
	// trigger price 500.
	s := &TotalFixedAmountStopLoss{Amount: stopLossTestMoney(1500)}
	position := stopLossTestPosition(1000, 3)
	candle := stopLossTestCandle("2010-01-05", 500, 900)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order (low exactly at trigger)")
	}
	assertOrder(t, "SellOrder", *order, "2010-01-05", 500, 3, domain.Sell)
}

// TestTotalFixedAmountStopLoss_CapsTotalLossForLargeQuantity is the point of
// this stop-loss type: unlike FixedAmountStopLoss (whose realized total
// loss scales with quantity, potentially far exceeding its configured
// Amount), TotalFixedAmountStopLoss must keep the realized total loss at or
// under Amount, no matter how many shares the position holds.
func TestTotalFixedAmountStopLoss_CapsTotalLossForLargeQuantity(t *testing.T) {
	const quantity = 517
	const amountCents = 50000 // R$500.00 total cap
	const entryCents = 100000 // R$1000.00 per share entry price

	s := &TotalFixedAmountStopLoss{Amount: stopLossTestMoney(amountCents)}
	position := stopLossTestPosition(entryCents, quantity)
	// A deep candle low, well past the trigger, to simulate a gap-through:
	// Check must still fill at the trigger price, not the worse low.
	candle := stopLossTestCandle("2010-01-05", 1, entryCents)

	order := s.Check(candle, position)
	if order == nil {
		t.Fatal("Check() = nil, want a sell order")
	}
	if order.Quantity != quantity {
		t.Fatalf("order.Quantity = %d, want %d", order.Quantity, quantity)
	}

	operation := domain.Operation{
		Date:      position.Buy.Date,
		BuyOrder:  position.Buy,
		SellOrder: *order,
	}
	profit, err := operation.Profit()
	if err != nil {
		t.Fatalf("Profit(): %v", err)
	}
	if !profit.IsNegative() {
		t.Fatalf("Profit() = %+v, want a loss (negative)", profit)
	}

	totalLossCents := -profit.Amount()
	if totalLossCents > amountCents {
		t.Errorf("total realized loss = %d cents, want <= %d cents (the configured Amount)", totalLossCents, amountCents)
	}
	// The floor division means the realized loss is short of Amount by at
	// most quantity-1 cents, never over it.
	if amountCents-totalLossCents >= quantity {
		t.Errorf("total realized loss = %d cents, unexpectedly far under Amount (%d); want within %d cents of it", totalLossCents, amountCents, quantity)
	}

	// Contrast: FixedAmountStopLoss with the same Amount treated as a
	// PER-SHARE drop would realize a vastly larger loss for this quantity -
	// this is exactly the distinction this new type exists to avoid.
	perShareLossCents := int64(amountCents * quantity)
	if totalLossCents >= perShareLossCents {
		t.Fatalf("total-fixed-amount loss (%d) should be far below what a per-share fixed-amount stop-loss with the same value would realize (%d)", totalLossCents, perShareLossCents)
	}
}
