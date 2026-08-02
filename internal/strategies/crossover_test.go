package strategies

import (
	"strings"
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestCrossover_WarmUp(t *testing.T) {
	s, err := newCrossoverStrategy("SMA Crossover", "sma", map[string]float64{"shortPeriod": 1, "longPeriod": 2})
	if err != nil {
		t.Fatalf("newCrossoverStrategy: %v", err)
	}

	candles := []domain.Candle{candleWithClose("2010-01-04", 1000)}
	if ops := runStrategy(s, newMoney(10000), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (long indicator not ready yet)", ops)
	}
}

// TestSMACrossover_BuysAndSellsOnCross feeds closes 10, 10, 20, 5, 5 (in
// major units) through a shortPeriod=1/longPeriod=2 SMA crossover.
//
//   - c1: short SMA(1) ready at 10, long SMA(2) not ready yet -> no signal.
//   - c2: short=10, long=avg(10,10)=10 -> not above -> first ready candle,
//     baseline set, no cross yet.
//   - c3: short=20, long=avg(10,20)=15 -> short is now above -> crossed up
//     -> buy at candle.Close (20).
//   - c4: short=5, long=avg(20,5)=12.5 -> short is now below -> crossed
//     down -> sell at candle.Close (5).
//   - c5: short=5, long=avg(5,5)=5 -> not above -> no cross (still below).
//
// Both indicators are updated with each candle BEFORE its Value() is read
// (see crossoverStrategy.Decide's doc comment), so the cross detected on a
// given candle, and the price it fills at, both reflect that same candle's
// close.
func TestSMACrossover_BuysAndSellsOnCross(t *testing.T) {
	s, err := newCrossoverStrategy("SMA Crossover", "sma", map[string]float64{"shortPeriod": 1, "longPeriod": 2})
	if err != nil {
		t.Fatalf("newCrossoverStrategy: %v", err)
	}

	candles := []domain.Candle{
		candleWithClose("2010-01-04", 1000),
		candleWithClose("2010-01-05", 1000),
		candleWithClose("2010-01-06", 2000),
		candleWithClose("2010-01-07", 500),
		candleWithClose("2010-01-08", 500),
	}

	ops := runStrategy(s, newMoney(10000), candles)
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}

	op := ops[0]
	assertOrder(t, "BuyOrder", op.BuyOrder, "2010-01-06", 2000, 5, domain.Buy)
	assertOrder(t, "SellOrder", op.SellOrder, "2010-01-07", 500, 5, domain.Sell)
}

// TestEMACrossover_BuysAndSellsOnCross runs the exact same candle sequence
// as TestSMACrossover_BuysAndSellsOnCross through an EMA crossover instead:
// with shortPeriod=1, the short EMA's multiplier is 2/(1+1)=1, so it
// always equals the current close (same as a period-1 SMA); the long
// EMA(2) is seeded with avg(10,10)=10, then blends in via multiplier
// 2/3: ema(20)=(20-10)*(2/3)+10=16.666..., ema(5)=(5-16.666...)*(2/3)+
// 16.666...=8.888.... Short crosses above long at c3 (20 > 16.67) and back
// below at c4 (5 < 8.89), same as the SMA case, producing the same
// buy/sell prices.
func TestEMACrossover_BuysAndSellsOnCross(t *testing.T) {
	s, err := newCrossoverStrategy("EMA Crossover", "ema", map[string]float64{"shortPeriod": 1, "longPeriod": 2})
	if err != nil {
		t.Fatalf("newCrossoverStrategy: %v", err)
	}

	candles := []domain.Candle{
		candleWithClose("2010-01-04", 1000),
		candleWithClose("2010-01-05", 1000),
		candleWithClose("2010-01-06", 2000),
		candleWithClose("2010-01-07", 500),
		candleWithClose("2010-01-08", 500),
	}

	ops := runStrategy(s, newMoney(10000), candles)
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}

	op := ops[0]
	assertOrder(t, "BuyOrder", op.BuyOrder, "2010-01-06", 2000, 5, domain.Buy)
	assertOrder(t, "SellOrder", op.SellOrder, "2010-01-07", 500, 5, domain.Sell)
}

func TestNewCrossoverStrategy_MissingShortPeriod(t *testing.T) {
	_, err := newCrossoverStrategy("SMA Crossover", "sma", map[string]float64{"longPeriod": 5})
	if err == nil {
		t.Fatal("expected error for missing shortPeriod")
	}
	if !strings.Contains(err.Error(), "shortPeriod") {
		t.Errorf("error = %q, want it to mention shortPeriod", err.Error())
	}
}

func TestNewCrossoverStrategy_MissingLongPeriod(t *testing.T) {
	_, err := newCrossoverStrategy("SMA Crossover", "sma", map[string]float64{"shortPeriod": 2})
	if err == nil {
		t.Fatal("expected error for missing longPeriod")
	}
	if !strings.Contains(err.Error(), "longPeriod") {
		t.Errorf("error = %q, want it to mention longPeriod", err.Error())
	}
}

func TestNewCrossoverStrategy_ShortNotLessThanLong(t *testing.T) {
	cases := []map[string]float64{
		{"shortPeriod": 5, "longPeriod": 5},
		{"shortPeriod": 10, "longPeriod": 5},
	}
	for _, params := range cases {
		if _, err := newCrossoverStrategy("SMA Crossover", "sma", params); err == nil {
			t.Errorf("newCrossoverStrategy(%+v) = nil error, want an error (shortPeriod must be < longPeriod)", params)
		}
	}
}

func TestNewCrossoverStrategy_InvalidPeriod(t *testing.T) {
	if _, err := newCrossoverStrategy("SMA Crossover", "sma", map[string]float64{"shortPeriod": -1, "longPeriod": 5}); err == nil {
		t.Fatal("expected error for invalid shortPeriod")
	}
}

func TestLoadStrategy_SMACrossover(t *testing.T) {
	s, err := LoadStrategy("sma-crossover", map[string]float64{"shortPeriod": 2, "longPeriod": 5})
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	if got := s.Name(); got != "SMA Crossover" {
		t.Errorf("Name() = %q, want %q", got, "SMA Crossover")
	}
}

func TestLoadStrategy_EMACrossover(t *testing.T) {
	s, err := LoadStrategy("ema-crossover", map[string]float64{"shortPeriod": 2, "longPeriod": 5})
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	if got := s.Name(); got != "EMA Crossover" {
		t.Errorf("Name() = %q, want %q", got, "EMA Crossover")
	}
}
