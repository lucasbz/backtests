package strategies

import (
	"strings"
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestRSIThreshold_WarmUp(t *testing.T) {
	s, err := newRSIThreshold(map[string]float64{"period": 1, "oversold": 30, "overbought": 70})
	if err != nil {
		t.Fatalf("newRSIThreshold: %v", err)
	}

	candles := []domain.Candle{candleWithClose("2010-01-04", 10000)}
	if ops := runStrategy(s, newMoney(90000), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (no delta yet, RSI not ready)", ops)
	}
}

// TestRSIThreshold_BuysAndSellsOnThresholdCross feeds closes 100, 105,
// 110, 90, 95 (major units) through a period=1 RSI (oversold=30,
// overbought=70). With Period=1, Wilder's smoothing forgets all history
// except the latest delta, so avgGain/avgLoss - and hence RSI - are fully
// determined by whether the most recent close rose or fell:
//
//   - c2 (delta +5): avgGain=5, avgLoss=0 -> RSI=100. First ready candle -
//     baseline set (aboveOverbought=true), no signal yet.
//   - c3 (delta +5): RSI=100 again -> aboveOverbought stays true -> no
//     cross.
//   - c4 (delta -20): avgGain=0, avgLoss=20 -> RSI=0 -> belowOversold
//     crosses true (was false) -> buy at candle.Close (90).
//   - c5 (delta +5): avgGain=5, avgLoss=0 -> RSI=100 -> aboveOverbought
//     crosses true (was false, reset at c4) -> sell at candle.Close (95).
//
// RSI is updated with each candle BEFORE its Value() is read (see
// RSIThreshold.Decide's doc comment), so the crossing detected on a given
// candle, and the price it fills at, both reflect that same candle's
// close.
func TestRSIThreshold_BuysAndSellsOnThresholdCross(t *testing.T) {
	s, err := newRSIThreshold(map[string]float64{"period": 1, "oversold": 30, "overbought": 70})
	if err != nil {
		t.Fatalf("newRSIThreshold: %v", err)
	}

	candles := []domain.Candle{
		candleWithClose("2010-01-04", 10000),
		candleWithClose("2010-01-05", 10500),
		candleWithClose("2010-01-06", 11000),
		candleWithClose("2010-01-07", 9000),
		candleWithClose("2010-01-08", 9500),
	}

	ops := runStrategy(s, newMoney(90000), candles)
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}

	op := ops[0]
	assertOrder(t, "BuyOrder", op.BuyOrder, "2010-01-07", 9000, 10, domain.Buy)
	assertOrder(t, "SellOrder", op.SellOrder, "2010-01-08", 9500, 10, domain.Sell)
}

func TestNewRSIThreshold_MissingParams(t *testing.T) {
	cases := []map[string]float64{
		{"oversold": 30, "overbought": 70},
		{"period": 14, "overbought": 70},
		{"period": 14, "oversold": 30},
	}
	for _, params := range cases {
		if _, err := newRSIThreshold(params); err == nil {
			t.Errorf("newRSIThreshold(%+v) = nil error, want an error for a missing param", params)
		}
	}
}

func TestNewRSIThreshold_InvalidThresholds(t *testing.T) {
	cases := []map[string]float64{
		{"period": 14, "oversold": -1, "overbought": 70},  // oversold out of range
		{"period": 14, "oversold": 30, "overbought": 101}, // overbought out of range
		{"period": 14, "oversold": 70, "overbought": 30},  // oversold >= overbought
		{"period": 14, "oversold": 50, "overbought": 50},  // oversold == overbought
	}
	for _, params := range cases {
		if _, err := newRSIThreshold(params); err == nil {
			t.Errorf("newRSIThreshold(%+v) = nil error, want an error", params)
		}
	}
}

func TestNewRSIThreshold_InvalidPeriod(t *testing.T) {
	_, err := newRSIThreshold(map[string]float64{"period": 0, "oversold": 30, "overbought": 70})
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
	if !strings.Contains(err.Error(), "period") {
		t.Errorf("error = %q, want it to mention period", err.Error())
	}
}

func TestLoadStrategy_RSIThreshold(t *testing.T) {
	s, err := LoadStrategy("rsi-threshold", map[string]float64{"period": 14, "oversold": 30, "overbought": 70})
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	if got := s.Name(); got != "RSI Threshold" {
		t.Errorf("Name() = %q, want %q", got, "RSI Threshold")
	}
}
