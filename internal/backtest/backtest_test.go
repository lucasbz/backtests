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

func newMoney(amount int64) money.Money {
	return *money.New(amount, domain.Currency)
}

func newCandle(date string, low, high int64) domain.Candle {
	return domain.Candle{Date: date, Low: newMoney(low), High: newMoney(high)}
}

func assertOrder(t *testing.T, label string, got domain.Order, wantDate string, wantPrice, wantQuantity int64, wantType domain.OrderType) {
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

// stubStrategy is a minimal, fully controllable domain.Strategy for testing
// traverse without depending on any real strategy's trading logic.
type stubStrategy struct {
	decide func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order
}

func (s *stubStrategy) Name() string { return "Stub" }
func (s *stubStrategy) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
	return s.decide(candle, position, isLast)
}

// stubStopLoss is a minimal, fully controllable domain.StopLoss for testing
// traverse's OCO race.
type stubStopLoss struct {
	check func(candle domain.Candle, position domain.Position) *domain.Order
}

func (s *stubStopLoss) Name() string { return "Stub Stop-Loss" }
func (s *stubStopLoss) Check(candle domain.Candle, position domain.Position) *domain.Order {
	return s.check(candle, position)
}

func TestNewResult_ProfitAndEndingBalance(t *testing.T) {
	operations := []domain.Operation{
		{
			Date:      "2010-01-04",
			BuyOrder:  domain.Order{Date: "2010-01-04", Price: newMoney(1200), Quantity: 10, OrderType: domain.Buy},
			SellOrder: domain.Order{Date: "2010-12-30", Price: newMoney(1500), Quantity: 10, OrderType: domain.Sell},
		},
	}

	result, err := NewResult("Stub", operations, newMoney(5000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
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
	if !approxEqual(result.ProfitPercentage(), 60) {
		t.Errorf("ProfitPercentage = %v, want 60", result.ProfitPercentage())
	}
	if !approxEqual(result.WinRate(), 100) {
		t.Errorf("WinRate = %v, want 100", result.WinRate())
	}
}

func TestNewResult_NoOperationsBreaksEven(t *testing.T) {
	result, err := NewResult("Stub", nil, newMoney(5000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
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
	if result.ProfitPercentage() != 0 {
		t.Errorf("ProfitPercentage = %v, want 0", result.ProfitPercentage())
	}
	if result.WinRate() != 0 {
		t.Errorf("WinRate = %v, want 0", result.WinRate())
	}
	if result.MaxDrawdownPercentage() != 0 {
		t.Errorf("MaxDrawdownPercentage = %v, want 0 (no operations)", result.MaxDrawdownPercentage())
	}
	maxDrawdownAmount := result.MaxDrawdownAmount()
	if !maxDrawdownAmount.IsZero() {
		t.Errorf("MaxDrawdownAmount = %d, want 0 (no operations)", maxDrawdownAmount.Amount())
	}
}

func TestNewResult_ZeroStartingBalanceDoesNotPanic(t *testing.T) {
	operations := []domain.Operation{
		{
			BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(2000), Quantity: 1, OrderType: domain.Sell},
		},
	}

	result, err := NewResult("Stub", operations, newMoney(0))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
	}
	if result.ProfitPercentage() != 0 {
		t.Errorf("ProfitPercentage = %v, want 0 (zero starting balance)", result.ProfitPercentage())
	}
}

func TestNewResult_Loss(t *testing.T) {
	operations := []domain.Operation{
		{
			BuyOrder:  domain.Order{Price: newMoney(2000), Quantity: 5, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(1000), Quantity: 5, OrderType: domain.Sell},
		},
	}

	result, err := NewResult("Stub", operations, newMoney(10000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
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
	if !approxEqual(result.ProfitPercentage(), -50) {
		t.Errorf("ProfitPercentage = %v, want -50", result.ProfitPercentage())
	}
	if !approxEqual(result.WinRate(), 0) {
		t.Errorf("WinRate = %v, want 0", result.WinRate())
	}
}

func TestNewResult_MixedGainsAndLossesWithBreakEvenCountedAsGain(t *testing.T) {
	operations := []domain.Operation{
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
	}

	result, err := NewResult("Stub", operations, newMoney(10000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
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
	if !approxEqual(result.WinRate(), 75) {
		t.Errorf("WinRate = %v, want 75", result.WinRate())
	}
}

// TestCompileResult_MaxDrawdown_BalanceOnlyIncreasesIsZero checks that a
// balance that only ever climbs (a new peak after every operation) never
// registers a drawdown.
func TestNewResult_MaxDrawdown_BalanceOnlyIncreasesIsZero(t *testing.T) {
	operations := []domain.Operation{
		{ // +1000
			BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(2000), Quantity: 1, OrderType: domain.Sell},
		},
		{ // +500
			BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(1500), Quantity: 1, OrderType: domain.Sell},
		},
	}

	result, err := NewResult("Stub", operations, newMoney(10000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
	}
	if result.MaxDrawdownPercentage() != 0 {
		t.Errorf("MaxDrawdownPercentage = %v, want 0 (balance only ever increased)", result.MaxDrawdownPercentage())
	}
	maxDrawdownAmount := result.MaxDrawdownAmount()
	if !maxDrawdownAmount.IsZero() {
		t.Errorf("MaxDrawdownAmount = %d, want 0 (balance only ever increased)", maxDrawdownAmount.Amount())
	}
}

// TestCompileResult_MaxDrawdown_PeakThenTroughThenPartialRecovery checks the
// straightforward case: balance climbs to a peak, drops to a trough, then
// partially recovers without setting a new peak - the drawdown should be
// measured from the peak to the trough, not affected by the partial
// recovery.
func TestNewResult_MaxDrawdown_PeakThenTroughThenPartialRecovery(t *testing.T) {
	operations := []domain.Operation{
		{ // +5000: 10000 -> 15000 (new peak)
			BuyOrder:  domain.Order{Price: newMoney(10000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(15000), Quantity: 1, OrderType: domain.Sell},
		},
		{ // -3000: 15000 -> 12000 (trough)
			BuyOrder:  domain.Order{Price: newMoney(13000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(10000), Quantity: 1, OrderType: domain.Sell},
		},
		{ // +1000: 12000 -> 13000 (partial recovery, still below the 15000 peak)
			BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(2000), Quantity: 1, OrderType: domain.Sell},
		},
	}

	result, err := NewResult("Stub", operations, newMoney(10000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
	}
	// (15000 - 12000) / 15000 * 100
	if !approxEqual(result.MaxDrawdownPercentage(), 20) {
		t.Errorf("MaxDrawdownPercentage = %v, want 20", result.MaxDrawdownPercentage())
	}
	// 15000 (peak) - 12000 (trough)
	maxDrawdownAmount := result.MaxDrawdownAmount()
	if maxDrawdownAmount.Amount() != 3000 {
		t.Errorf("MaxDrawdownAmount = %d, want 3000", maxDrawdownAmount.Amount())
	}
}

// TestCompileResult_MaxDrawdown_PicksLargestOfSeveralDrawdowns checks that,
// with several distinct peak/trough cycles of different sizes, the reported
// MaxDrawdownPercentage is the largest one encountered - not the first one
// and not the last one.
func TestNewResult_MaxDrawdown_PicksLargestOfSeveralDrawdowns(t *testing.T) {
	operations := []domain.Operation{
		{ // +5000: 10000 -> 15000 (peak)
			BuyOrder:  domain.Order{Price: newMoney(10000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(15000), Quantity: 1, OrderType: domain.Sell},
		},
		{ // -1500: 15000 -> 13500 (first, smaller drawdown: 10%)
			BuyOrder:  domain.Order{Price: newMoney(1500), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(0), Quantity: 1, OrderType: domain.Sell},
		},
		{ // +4000: 13500 -> 17500 (new peak)
			BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(5000), Quantity: 1, OrderType: domain.Sell},
		},
		{ // -7000: 17500 -> 10500 (largest, middle drawdown: ~40%)
			BuyOrder:  domain.Order{Price: newMoney(7000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(0), Quantity: 1, OrderType: domain.Sell},
		},
		{ // +3000: 10500 -> 13500 (still below the 17500 peak)
			BuyOrder:  domain.Order{Price: newMoney(1000), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(4000), Quantity: 1, OrderType: domain.Sell},
		},
		{ // -500: 13500 -> 13000 (last, smaller drawdown: ~25.7%)
			BuyOrder:  domain.Order{Price: newMoney(500), Quantity: 1, OrderType: domain.Buy},
			SellOrder: domain.Order{Price: newMoney(0), Quantity: 1, OrderType: domain.Sell},
		},
	}

	result, err := NewResult("Stub", operations, newMoney(10000))
	if err != nil {
		t.Fatalf("NewResult: %v", err)
	}
	// (17500 - 10500) / 17500 * 100
	if !approxEqual(result.MaxDrawdownPercentage(), 40) {
		t.Errorf("MaxDrawdownPercentage = %v, want 40 (the largest drawdown, not the first ~10%% or the last ~25.7%%)", result.MaxDrawdownPercentage())
	}
	// 17500 (peak) - 10500 (trough), the largest drawdown - not the first
	// (1500, ~10%) or the last (500, ~25.7%)
	maxDrawdownAmount := result.MaxDrawdownAmount()
	if maxDrawdownAmount.Amount() != 7000 {
		t.Errorf("MaxDrawdownAmount = %d, want 7000 (the largest drawdown)", maxDrawdownAmount.Amount())
	}
}

// TestTraverse_NoStopLoss_BuysAndSells is the baseline: with no stop-loss
// configured (nil), traverse behaves exactly like a plain strategy-driven
// backtest - the strategy buys while flat and sells while holding, and
// quantity is sized from the running balance divided by the entry price.
func TestTraverse_NoStopLoss_BuysAndSells(t *testing.T) {
	candles := []domain.Candle{
		newCandle("2010-01-04", 1000, 1050),
		newCandle("2010-01-05", 1100, 1500),
	}

	strategy := &stubStrategy{
		decide: func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
			if position == nil {
				return &domain.Order{Date: candle.Date, Price: candle.Low, OrderType: domain.Buy}
			}
			return &domain.Order{Date: candle.Date, Price: candle.High, OrderType: domain.Sell}
		},
	}

	ops, err := traverse(strategy, nil, newMoney(10000), candles)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}
	// 10000 / 1000 = 10 shares
	assertOrder(t, "BuyOrder", ops[0].BuyOrder, "2010-01-04", 1000, 10, domain.Buy)
	assertOrder(t, "SellOrder", ops[0].SellOrder, "2010-01-05", 1500, 10, domain.Sell)
}

// TestTraverse_UnclosedPositionIsDropped mirrors two-candle-breakout's
// documented behavior: if the strategy never signals an exit, the open
// position at the end of the candle list produces no completed operation.
func TestTraverse_UnclosedPositionIsDropped(t *testing.T) {
	candles := []domain.Candle{
		newCandle("2010-01-04", 1000, 1050),
		newCandle("2010-01-05", 1100, 1200),
	}

	strategy := &stubStrategy{
		decide: func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
			if position == nil {
				return &domain.Order{Date: candle.Date, Price: candle.Low, OrderType: domain.Buy}
			}
			return nil // never exits
		},
	}

	ops, err := traverse(strategy, nil, newMoney(10000), candles)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if ops != nil {
		t.Errorf("ops = %+v, want nil (unclosed position dropped)", ops)
	}
}

// TestTraverse_StopLossTriggersExit checks that a configured StopLoss's
// Check is consulted while a position is open, and closes it when the
// strategy itself has no exit signal.
func TestTraverse_StopLossTriggersExit(t *testing.T) {
	candles := []domain.Candle{
		newCandle("2010-01-04", 1000, 1050),
		newCandle("2010-01-05", 900, 1200), // low dips to 900, below the stop
	}

	strategy := &stubStrategy{
		decide: func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
			if position == nil {
				return &domain.Order{Date: candle.Date, Price: candle.Low, OrderType: domain.Buy}
			}
			return nil // strategy never wants to exit on its own in this test
		},
	}
	stopLoss := &stubStopLoss{
		check: func(candle domain.Candle, position domain.Position) *domain.Order {
			trigger := newMoney(950) // 5% below the 1000 entry, for example
			if candle.Low.Amount() > trigger.Amount() {
				return nil
			}
			return &domain.Order{Date: candle.Date, Price: trigger, OrderType: domain.Sell}
		},
	}

	ops, err := traverse(strategy, stopLoss, newMoney(10000), candles)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}
	assertOrder(t, "SellOrder", ops[0].SellOrder, "2010-01-05", 950, 10, domain.Sell)
}

// TestTraverse_StopLossNotConsultedWhileFlat checks that StopLoss.Check is
// never called before a position is open - it has no entry-side role (see
// domain.StopLoss).
func TestTraverse_StopLossNotConsultedWhileFlat(t *testing.T) {
	candles := []domain.Candle{newCandle("2010-01-04", 1000, 1050)}

	strategy := &stubStrategy{
		decide: func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
			return nil // never buys, so position always stays nil
		},
	}
	checked := false
	stopLoss := &stubStopLoss{
		check: func(candle domain.Candle, position domain.Position) *domain.Order {
			checked = true
			return nil
		},
	}

	if _, err := traverse(strategy, stopLoss, newMoney(10000), candles); err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if checked {
		t.Error("StopLoss.Check was called while flat, want it only consulted while a position is open")
	}
}

// TestTraverse_TieBreak_StopLossWinsOverStrategyExit is the documented OCO
// default (see docs/plans/stop-loss-oco.md's "Tie-break rule"): if both the
// strategy's own exit signal and the stop-loss fire on the same candle, the
// stop-loss's order is the one that closes the position.
func TestTraverse_TieBreak_StopLossWinsOverStrategyExit(t *testing.T) {
	candles := []domain.Candle{
		newCandle("2010-01-04", 1000, 1050),
		newCandle("2010-01-05", 900, 1300),
	}

	strategy := &stubStrategy{
		decide: func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
			if position == nil {
				return &domain.Order{Date: candle.Date, Price: candle.Low, OrderType: domain.Buy}
			}
			// strategy also wants to exit this same candle, at a different
			// price than the stop-loss.
			return &domain.Order{Date: candle.Date, Price: newMoney(1300), OrderType: domain.Sell}
		},
	}
	stopLoss := &stubStopLoss{
		check: func(candle domain.Candle, position domain.Position) *domain.Order {
			return &domain.Order{Date: candle.Date, Price: newMoney(950), OrderType: domain.Sell}
		},
	}

	ops, err := traverse(strategy, stopLoss, newMoney(10000), candles)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}
	// stop-loss's 950, not the strategy's 1300, should have won the race.
	assertOrder(t, "SellOrder", ops[0].SellOrder, "2010-01-05", 950, 10, domain.Sell)
}

// TestTraverse_AccumulatesStrandedCashAcrossTrades locks in the fix for
// finding #5 (docs/plans/code-review-findings.md): integer-division
// quantity sizing leaves a fractional remainder of cash unspent on a buy
// ("dust"); traverse now carries that remainder forward - added to each
// later sale's proceeds - instead of discarding it when a position closes.
//
//   - cycle 1: buy @300 with 1000 available -> qty = 1000/300 = 3, spends
//     900, leaves 100 stranded ("dust").
//   - cycle 1 sell @400 x3 -> proceeds 1200. Available for the next buy is
//     the stranded 100 *plus* the 1200 proceeds = 1300, not just 1200.
//   - cycle 2: buy @100 with 1300 available -> qty = 1300/100 = 13. Before
//     this fix, traverse discarded the 100 dust on close (balance =
//     exit.Total() = 1200 only), which would have sized this buy at
//     1200/100 = 12 shares instead - one fewer than the true available
//     cash affords.
func TestTraverse_AccumulatesStrandedCashAcrossTrades(t *testing.T) {
	candles := []domain.Candle{
		newCandle("2010-01-04", 300, 350), // cycle 1 buy candle
		newCandle("2010-01-05", 380, 400), // cycle 1 sell candle
		newCandle("2010-01-06", 100, 150), // cycle 2 buy candle
		newCandle("2010-01-07", 140, 160), // cycle 2 sell candle
	}

	strategy := &stubStrategy{
		decide: func(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
			if position == nil {
				return &domain.Order{Date: candle.Date, Price: candle.Low, OrderType: domain.Buy}
			}
			return &domain.Order{Date: candle.Date, Price: candle.High, OrderType: domain.Sell}
		},
	}

	ops, err := traverse(strategy, nil, newMoney(1000), candles)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if len(ops) != 2 {
		t.Fatalf("got %d operations, want 2: %+v", len(ops), ops)
	}

	assertOrder(t, "cycle 1 BuyOrder", ops[0].BuyOrder, "2010-01-04", 300, 3, domain.Buy)
	assertOrder(t, "cycle 1 SellOrder", ops[0].SellOrder, "2010-01-05", 400, 3, domain.Sell)

	// The key assertion: qty 13, not 12 - only possible by carrying the
	// cycle 1 buy's 100 dust forward into cycle 2's sizing pool alongside
	// the 1200 sale proceeds (1300 / 100 = 13).
	assertOrder(t, "cycle 2 BuyOrder", ops[1].BuyOrder, "2010-01-06", 100, 13, domain.Buy)
	assertOrder(t, "cycle 2 SellOrder", ops[1].SellOrder, "2010-01-07", 160, 13, domain.Sell)
}

func TestPickExit(t *testing.T) {
	strategyExit := &domain.Order{Price: newMoney(100)}
	stopExit := &domain.Order{Price: newMoney(90)}

	if got := pickExit(nil, nil); got != nil {
		t.Errorf("pickExit(nil, nil) = %+v, want nil", got)
	}
	if got := pickExit(strategyExit, nil); got != strategyExit {
		t.Errorf("pickExit(strategyExit, nil) = %+v, want strategyExit", got)
	}
	if got := pickExit(nil, stopExit); got != stopExit {
		t.Errorf("pickExit(nil, stopExit) = %+v, want stopExit", got)
	}
	if got := pickExit(strategyExit, stopExit); got != stopExit {
		t.Errorf("pickExit(strategyExit, stopExit) = %+v, want stopExit (stop-loss wins ties)", got)
	}
}
