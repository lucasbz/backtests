package backtest

import (
	"math"
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

// stubStrategy is a minimal domain.Strategy for testing result compilation
// without depending on real candle data or a concrete strategy's logic.
type stubStrategy struct {
	traversed  []domain.Candle
	operations []domain.Operation
}

func (s *stubStrategy) Traverse(candle domain.Candle) { s.traversed = append(s.traversed, candle) }
func (s *stubStrategy) Name() string                  { return "Stub" }
func (s *stubStrategy) Operations() []domain.Operation {
	return s.operations
}

func newMoney(amount int64) money.Money {
	return *money.New(amount, domain.Currency)
}

func TestCompileResult_ProfitAndEndingBalance(t *testing.T) {
	strategy := &stubStrategy{
		operations: []domain.Operation{
			{
				Date:      "2010-01-04",
				BuyOrder:  domain.Order{Date: "2010-01-04", Price: newMoney(1200), Quantity: 10, OrderType: domain.Buy},
				SellOrder: domain.Order{Date: "2010-12-30", Price: newMoney(1500), Quantity: 10, OrderType: domain.Sell},
			},
		},
	}

	result, err := compileResult(strategy, newMoney(5000))
	if err != nil {
		t.Fatalf("compileResult: %v", err)
	}

	if result.StrategyName != "Stub" {
		t.Errorf("StrategyName = %q, want %q", result.StrategyName, "Stub")
	}
	if len(result.Operations) != 1 {
		t.Fatalf("got %d operations, want 1", len(result.Operations))
	}
	if result.StartingBalance.Amount() != 5000 {
		t.Errorf("StartingBalance = %d, want 5000", result.StartingBalance.Amount())
	}
	// bought 10 @ 1200 = 12000, sold 10 @ 1500 = 15000, profit = 3000
	if result.Profit.Amount() != 3000 {
		t.Errorf("Profit = %d, want 3000", result.Profit.Amount())
	}
	if result.EndingBalance.Amount() != 8000 {
		t.Errorf("EndingBalance = %d, want 8000", result.EndingBalance.Amount())
	}
	if result.TotalOperations != 1 {
		t.Errorf("TotalOperations = %d, want 1", result.TotalOperations)
	}
	if result.Gains != 1 || result.Losses != 0 {
		t.Errorf("Gains=%d Losses=%d, want Gains=1 Losses=0", result.Gains, result.Losses)
	}
	// profit 3000 / starting 5000 * 100
	if !approxEqual(result.ProfitPercentage, 60) {
		t.Errorf("ProfitPercentage = %v, want 60", result.ProfitPercentage)
	}
	if !approxEqual(result.WinRate, 100) {
		t.Errorf("WinRate = %v, want 100", result.WinRate)
	}
}

func TestCompileResult_NoOperationsBreaksEven(t *testing.T) {
	strategy := &stubStrategy{}

	result, err := compileResult(strategy, newMoney(5000))
	if err != nil {
		t.Fatalf("compileResult: %v", err)
	}

	if len(result.Operations) != 0 {
		t.Errorf("got %d operations, want 0", len(result.Operations))
	}
	if result.TotalOperations != 0 {
		t.Errorf("TotalOperations = %d, want 0", result.TotalOperations)
	}
	if !result.Profit.IsZero() {
		t.Errorf("Profit = %d, want 0", result.Profit.Amount())
	}
	if result.EndingBalance.Amount() != result.StartingBalance.Amount() {
		t.Errorf("EndingBalance = %d, want %d (unchanged)", result.EndingBalance.Amount(), result.StartingBalance.Amount())
	}
	if result.Gains != 0 || result.Losses != 0 {
		t.Errorf("Gains=%d Losses=%d, want both 0", result.Gains, result.Losses)
	}
	if result.ProfitPercentage != 0 {
		t.Errorf("ProfitPercentage = %v, want 0", result.ProfitPercentage)
	}
	if result.WinRate != 0 {
		t.Errorf("WinRate = %v, want 0", result.WinRate)
	}
}

func TestCompileResult_ZeroStartingBalanceDoesNotPanic(t *testing.T) {
	strategy := &stubStrategy{
		operations: []domain.Operation{
			{
				BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
				SellOrder: domain.Order{Price: newMoney(2000), Quantity: 1, OrderType: domain.Sell},
			},
		},
	}

	result, err := compileResult(strategy, newMoney(0))
	if err != nil {
		t.Fatalf("compileResult: %v", err)
	}
	if result.ProfitPercentage != 0 {
		t.Errorf("ProfitPercentage = %v, want 0 (zero starting balance)", result.ProfitPercentage)
	}
}

func TestCompileResult_Loss(t *testing.T) {
	strategy := &stubStrategy{
		operations: []domain.Operation{
			{
				BuyOrder:  domain.Order{Price: newMoney(2000), Quantity: 5, OrderType: domain.Buy},
				SellOrder: domain.Order{Price: newMoney(1000), Quantity: 5, OrderType: domain.Sell},
			},
		},
	}

	result, err := compileResult(strategy, newMoney(10000))
	if err != nil {
		t.Fatalf("compileResult: %v", err)
	}

	// bought 5 @ 2000 = 10000, sold 5 @ 1000 = 5000, profit = -5000
	if result.Profit.Amount() != -5000 {
		t.Errorf("Profit = %d, want -5000", result.Profit.Amount())
	}
	if result.EndingBalance.Amount() != 5000 {
		t.Errorf("EndingBalance = %d, want 5000", result.EndingBalance.Amount())
	}
	if result.TotalOperations != 1 {
		t.Errorf("TotalOperations = %d, want 1", result.TotalOperations)
	}
	if result.Gains != 0 || result.Losses != 1 {
		t.Errorf("Gains=%d Losses=%d, want Gains=0 Losses=1", result.Gains, result.Losses)
	}
	// profit -5000 / starting 10000 * 100
	if !approxEqual(result.ProfitPercentage, -50) {
		t.Errorf("ProfitPercentage = %v, want -50", result.ProfitPercentage)
	}
	if !approxEqual(result.WinRate, 0) {
		t.Errorf("WinRate = %v, want 0", result.WinRate)
	}
}

func TestCompileResult_MixedGainsAndLossesWithBreakEvenCountedAsGain(t *testing.T) {
	strategy := &stubStrategy{
		operations: []domain.Operation{
			{ // gain: +1000
				BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
				SellOrder: domain.Order{Price: newMoney(2000), Quantity: 1, OrderType: domain.Sell},
			},
			{ // loss: -500
				BuyOrder:  domain.Order{Price: newMoney(2000), Quantity: 1, OrderType: domain.Buy},
				SellOrder: domain.Order{Price: newMoney(1500), Quantity: 1, OrderType: domain.Sell},
			},
			{ // break-even: 0, counted as a gain
				BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
				SellOrder: domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Sell},
			},
			{ // gain: +200
				BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
				SellOrder: domain.Order{Price: newMoney(1200), Quantity: 1, OrderType: domain.Sell},
			},
		},
	}

	result, err := compileResult(strategy, newMoney(10000))
	if err != nil {
		t.Fatalf("compileResult: %v", err)
	}

	if result.TotalOperations != 4 {
		t.Errorf("TotalOperations = %d, want 4", result.TotalOperations)
	}
	if result.Gains != 3 {
		t.Errorf("Gains = %d, want 3", result.Gains)
	}
	if result.Losses != 1 {
		t.Errorf("Losses = %d, want 1", result.Losses)
	}
	// 3 gains out of 4 operations
	if !approxEqual(result.WinRate, 75) {
		t.Errorf("WinRate = %v, want 75", result.WinRate)
	}
}
